package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultEntry represents a single item in a Hydra defaults: list.
type DefaultEntry struct {
	Key      string // e.g. "monty", "motor_system_config"
	Value    string // e.g. "informed_5_evidence1_camera_dist1"
	Absolute bool   // true if key had "/" prefix
	RawKey   string // original key for display (e.g. "/monty" or "config")
}

// TreeNode represents a node in the hierarchical config tree.
type TreeNode struct {
	Key      string
	Value    string
	FilePath string // absolute path to the resolved .yaml file
	Absolute bool
	Depth    int
	Expanded bool
	Children []*TreeNode
	Parent   *TreeNode
	IsLeaf   bool
	Error    string // non-empty if file couldn't be loaded
}

// parseDefaults reads a YAML file and extracts its defaults: list.
func parseDefaults(filePath string) ([]DefaultEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

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
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			val := ""
			if v != nil {
				switch vt := v.(type) {
				case string:
					val = vt
				default:
					continue
				}
			}

			entry := DefaultEntry{
				RawKey: k,
			}

			key := k
			// Handle @ suffix — strip everything from @ onward
			if atIdx := strings.Index(key, "@"); atIdx >= 0 {
				key = key[:atIdx]
			}

			// Handle "/" prefix for absolute references
			if strings.HasPrefix(key, "/") {
				entry.Absolute = true
				key = strings.TrimPrefix(key, "/")
			}

			entry.Key = key
			entry.Value = val
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// resolveFilePath resolves a DefaultEntry to an absolute file path.
// For absolute entries, the path is relative to confDir.
// For relative entries, the path is relative to the parent file's directory.
func resolveFilePath(entry DefaultEntry, parentFilePath, confDir string) string {
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

// buildTree recursively builds a tree of TreeNodes starting from a YAML file.
func buildTree(filePath, confDir string) ([]*TreeNode, error) {
	visited := make(map[string]bool)
	return buildTreeRecursive(filePath, confDir, 0, nil, visited)
}

func buildTreeRecursive(filePath, confDir string, depth int, parent *TreeNode, visited map[string]bool) ([]*TreeNode, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	entries, err := parseDefaults(absPath)
	if err != nil {
		return nil, err
	}

	var nodes []*TreeNode
	for _, entry := range entries {
		childPath := resolveFilePath(entry, absPath, confDir)
		childAbs, err := filepath.Abs(childPath)
		if err != nil {
			childAbs = childPath
		}

		node := &TreeNode{
			Key:      entry.Key,
			Value:    entry.Value,
			FilePath: childAbs,
			Absolute: entry.Absolute,
			Depth:    depth,
			Parent:   parent,
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
