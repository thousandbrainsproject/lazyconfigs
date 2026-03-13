package hydra

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultEntry represents a single item in a Hydra defaults: list.
type DefaultEntry struct {
	Key         string // e.g. "monty", "motor_system_config"
	Value       string // e.g. "informed_5_evidence1_camera_dist1"
	Absolute    bool   // true if key had "/" prefix
	RawKey      string // original key for display (e.g. "/monty" or "config")
	PackagePath string // portion after @ in raw key (e.g., "learning_module_0.learning_module_args")
}

// TreeNode represents a node in the hierarchical config tree.
type TreeNode struct {
	Key            string
	Value          string
	FilePath       string // absolute path to the resolved .yaml file
	Absolute       bool
	Depth          int
	Expanded       bool
	Children       []*TreeNode
	Parent         *TreeNode
	IsLeaf         bool
	Error          string // non-empty if file couldn't be loaded
	SourceFilePath string // the file whose defaults: list contains this entry
	RawKey         string // original key as in YAML (e.g., "/monty", "config")
}

// PackageDir returns the directory containing variant files for this node.
func (node *TreeNode) PackageDir(confDir string) string {
	if node.Absolute {
		return filepath.Join(confDir, node.Key)
	}
	return filepath.Join(filepath.Dir(node.SourceFilePath), node.Key)
}

// ListVariants reads directory and returns sorted .yaml filenames without extension,
// skipping subdirectories.
func ListVariants(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading variant directory %s: %w", dir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// ParseDefaults reads a YAML file and extracts its defaults: list.
func ParseDefaults(filePath string) ([]DefaultEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading defaults from %s: %w", filePath, err)
	}
	entries, err := ParseDefaultsFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parsing defaults in %s: %w", filePath, err)
	}
	return entries, nil
}

// parseDefaultKey parses a raw key string (with optional @package and / prefix)
// into a DefaultEntry with the given value.
func parseDefaultKey(rawKey, value string) DefaultEntry {
	entry := DefaultEntry{
		RawKey: rawKey,
		Value:  value,
	}

	key := rawKey
	// Handle @ suffix — extract package path and strip from key
	if atIdx := strings.Index(key, "@"); atIdx >= 0 {
		entry.PackagePath = key[atIdx+1:]
		key = key[:atIdx]
	}

	// Handle "/" prefix for absolute references
	if strings.HasPrefix(key, "/") {
		entry.Absolute = true
		key = strings.TrimPrefix(key, "/")
	}

	entry.Key = key
	return entry
}

// ParseDefaultsFromData extracts defaults entries from already-loaded YAML data.
func ParseDefaultsFromData(data []byte) ([]DefaultEntry, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	defaultsRaw, ok := raw["defaults"]
	if !ok {
		return nil, nil
	}

	defaultsList, ok := defaultsRaw.([]interface{})
	if !ok {
		return nil, nil
	}

	var entries []DefaultEntry
	for _, item := range defaultsList {
		switch v := item.(type) {
		case string:
			// Bare string defaults entry, e.g.:
			//   - hypotheses_updater/burst_sampling_10nn@learning_module_0.learning_module_args
			entries = append(entries, parseDefaultKey(v, ""))
		case map[string]interface{}:
			for k, val := range v {
				valStr := ""
				if val != nil {
					s, ok := val.(string)
					if !ok {
						continue
					}
					valStr = s
				}
				entries = append(entries, parseDefaultKey(k, valStr))
			}
		}
	}

	return entries, nil
}

// ResolveFilePath resolves a DefaultEntry to an absolute file path.
// For absolute entries, the path is relative to confDir.
// For relative entries, the path is relative to the parent file's directory.
func ResolveFilePath(entry DefaultEntry, parentFilePath, confDir string) string {
	key := entry.Key
	value := entry.Value

	// The file is at key/value.yaml
	// key may contain "/" subpaths (e.g. "hypotheses_updater/burst_sampling_5nn")
	var relPath string
	if value != "" {
		relPath = filepath.Join(key, value+".yaml")
	} else {
		relPath = key + ".yaml"
	}

	if entry.Absolute {
		return filepath.Join(confDir, relPath)
	}

	parentDir := filepath.Dir(parentFilePath)
	return filepath.Join(parentDir, relPath)
}

// BuildTree recursively builds a tree of TreeNodes starting from a YAML file.
func BuildTree(filePath, confDir string) ([]*TreeNode, error) {
	visited := make(map[string]bool)
	return buildTreeRecursive(filePath, confDir, 0, nil, visited)
}

func buildTreeRecursive(filePath, confDir string, depth int, parent *TreeNode, visited map[string]bool) ([]*TreeNode, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	entries, err := ParseDefaults(absPath)
	if err != nil {
		return nil, err
	}

	var nodes []*TreeNode
	for _, entry := range entries {
		// Only show "package: variant" style entries in the builder tree.
		// Bare string entries (e.g. _self_, hypotheses_updater/burst@pkg)
		// have no selectable variants and are handled only during resolution.
		if entry.Value == "" {
			continue
		}

		childPath := ResolveFilePath(entry, absPath, confDir)
		childAbs, err := filepath.Abs(childPath)
		if err != nil {
			childAbs = childPath
		}

		node := &TreeNode{
			Key:            entry.Key,
			Value:          entry.Value,
			FilePath:       childAbs,
			Absolute:       entry.Absolute,
			Depth:          depth,
			Parent:         parent,
			SourceFilePath: absPath,
			RawKey:         entry.RawKey,
		}

		// Check if file exists
		if _, err := os.Stat(childAbs); err != nil {
			node.IsLeaf = true
			node.Error = "file not found"
			nodes = append(nodes, node)
			continue
		}

		// Cycle detection
		if visited[childAbs] {
			node.IsLeaf = true
			node.Error = "circular reference"
			nodes = append(nodes, node)
			continue
		}

		visited[childAbs] = true
		children, err := buildTreeRecursive(childAbs, confDir, depth+1, node, visited)
		delete(visited, childAbs) // allow the same file in different branches

		if err != nil || len(children) == 0 {
			node.IsLeaf = true
			if err != nil {
				node.Error = err.Error()
			}
		} else {
			node.Children = children
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// FindFileReferences scans experiment files and returns names of experiments
// whose resolved config tree contains a node resolving to targetFilePath.
func FindFileReferences(confDir, targetFilePath string) ([]string, error) {
	expDir := filepath.Join(confDir, "experiment")
	var refs []string

	err := filepath.WalkDir(expDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		tree, err := BuildTree(path, confDir)
		if err != nil {
			return nil // skip unparseable files
		}
		if treeContainsFile(tree, targetFilePath) {
			rel, err := filepath.Rel(expDir, path)
			if err != nil {
				rel = filepath.Base(path)
			}
			refs = append(refs, strings.TrimSuffix(rel, ".yaml"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(refs)
	return refs, nil
}

// FindVariantReferences scans experiment files and returns names of experiments
// that reference the given variant at any level of the resolved config hierarchy.
func FindVariantReferences(confDir, variantDir, variantName string) ([]string, error) {
	targetFile := filepath.Join(variantDir, variantName+".yaml")
	targetAbs, err := filepath.Abs(targetFile)
	if err != nil {
		return nil, err
	}
	return FindFileReferences(confDir, targetAbs)
}

// VariantRef tracks a detailed reference to a variant within an experiment's config tree.
type VariantRef struct {
	ExperimentName string
	SourceFilePath string // the file whose defaults: list references the variant
	RawKey         string // the key in that defaults: list
}

// findRefsInTree recursively walks tree nodes and emits VariantRef entries
// for every node whose FilePath matches targetPath.
func findRefsInTree(nodes []*TreeNode, targetPath, expName string) []VariantRef {
	var refs []VariantRef
	for _, node := range nodes {
		if node.FilePath == targetPath {
			refs = append(refs, VariantRef{
				ExperimentName: expName,
				SourceFilePath: node.SourceFilePath,
				RawKey:         node.RawKey,
			})
		}
		refs = append(refs, findRefsInTree(node.Children, targetPath, expName)...)
	}
	return refs
}

// FindVariantReferencesDetailed scans experiment files and returns detailed
// reference information for every location that references the given variant.
func FindVariantReferencesDetailed(confDir, variantDir, variantName string) ([]VariantRef, error) {
	targetFile := filepath.Join(variantDir, variantName+".yaml")
	targetAbs, err := filepath.Abs(targetFile)
	if err != nil {
		return nil, err
	}

	expDir := filepath.Join(confDir, "experiment")
	var allRefs []VariantRef

	err = filepath.WalkDir(expDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		tree, err := BuildTree(path, confDir)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(expDir, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		expName := strings.TrimSuffix(rel, ".yaml")
		allRefs = append(allRefs, findRefsInTree(tree, targetAbs, expName)...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Deduplicate by SourceFilePath+RawKey
	seen := make(map[string]bool)
	var unique []VariantRef
	for _, ref := range allRefs {
		key := ref.SourceFilePath + "\x00" + ref.RawKey
		if !seen[key] {
			seen[key] = true
			unique = append(unique, ref)
		}
	}

	return unique, nil
}

// treeContainsFile recursively checks if any node in the tree resolves to the target file path.
func treeContainsFile(nodes []*TreeNode, targetPath string) bool {
	for _, node := range nodes {
		if node.FilePath == targetPath {
			return true
		}
		if treeContainsFile(node.Children, targetPath) {
			return true
		}
	}
	return false
}
