package hydra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// deepMerge recursively merges overlay into base. When both values are maps,
// it recurses. Otherwise overlay wins.
func deepMerge(base, overlay map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		if baseMap, ok := result[k].(map[string]interface{}); ok {
			if overlayMap, ok := v.(map[string]interface{}); ok {
				result[k] = deepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

// setNestedKey places value at a dot-separated path within m, creating
// intermediate maps as needed. dotPath must be non-empty.
func setNestedKey(m map[string]interface{}, dotPath string, value interface{}) {
	if dotPath == "" {
		return
	}
	parts := strings.Split(dotPath, ".")
	current := m
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

// parsePackageDirective extracts the @package directive from file content.
// Returns "" for _global_ or missing directive.
func parsePackageDirective(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# @package ") {
			pkg := strings.TrimSpace(strings.TrimPrefix(trimmed, "# @package "))
			if pkg == "_global_" {
				return ""
			}
			return pkg
		}
		// Stop after first non-comment, non-empty line
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}
	}
	return ""
}

// defaultPackageFromPath derives the default package from the file's config
// group path (used when no @package directive is present).
func defaultPackageFromPath(filePath, confDir string) string {
	rel, err := filepath.Rel(confDir, filePath)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return ""
	}
	return strings.ReplaceAll(dir, string(filepath.Separator), ".")
}

// relativePackage computes the child's package path relative to the parent's.
// If the child's package doesn't extend the parent's, returns the child's full path.
func relativePackage(parentPkg, childPkg string) string {
	if childPkg == "" {
		// _global_ child: no relative wrapping possible
		return ""
	}
	if parentPkg == "" {
		return childPkg
	}
	prefix := parentPkg + "."
	if strings.HasPrefix(childPkg, prefix) {
		return strings.TrimPrefix(childPkg, prefix)
	}
	if childPkg == parentPkg {
		return ""
	}
	// Child doesn't extend parent — use full path
	return childPkg
}

// resolveFileRecursive recursively resolves a Hydra config file by merging
// all defaults entries, placing each child at the correct position based on
// @package directives. Returns the resolved content (relative to this file's
// own @package) and the file's @package path.
func resolveFileRecursive(filePath, confDir string, visited map[string]bool) (map[string]interface{}, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", filePath, err)
	}

	myPkg := parsePackageDirective(data)
	if myPkg == "" {
		// No directive — check if file has no @package at all vs explicit _global_
		// For files without directive, derive from config group path
		if !strings.Contains(string(data), "# @package") {
			myPkg = defaultPackageFromPath(filePath, confDir)
		}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, "", fmt.Errorf("parsing %s: %w", filePath, err)
	}
	if raw == nil {
		raw = make(map[string]interface{})
	}

	// Extract and remove defaults (reuse already-loaded data)
	entries, err := ParseDefaultsFromData(data)
	if err != nil {
		return nil, "", fmt.Errorf("parsing defaults in %s: %w", filePath, err)
	}
	delete(raw, "defaults")

	// Accumulate resolved defaults
	accumulated := make(map[string]interface{})
	for _, entry := range entries {
		if entry.Key == "_self_" {
			continue
		}

		childPath := ResolveFilePath(entry, filePath, confDir)
		childPath, _ = filepath.Abs(childPath)

		if visited[childPath] {
			continue // cycle — skip
		}

		visited[childPath] = true
		resolved, childPkg, err := resolveFileRecursive(childPath, confDir, visited)
		delete(visited, childPath)

		if err != nil {
			return nil, "", err
		}

		// If defaults entry has @suffix, it overrides the child's @package
		effectivePkg := childPkg
		if entry.PackagePath != "" {
			effectivePkg = entry.PackagePath
		}

		// Compute where to place child content relative to this file
		relPath := relativePackage(myPkg, effectivePkg)
		if relPath != "" {
			wrapper := make(map[string]interface{})
			setNestedKey(wrapper, relPath, resolved)
			resolved = wrapper
		}

		accumulated = deepMerge(accumulated, resolved)
	}

	// Current file's own content overrides accumulated defaults
	return deepMerge(accumulated, raw), myPkg, nil
}

// ResolveFile is the public entry point. It resolves a Hydra config file and
// returns the result as a YAML string.
func ResolveFile(filePath, confDir string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("resolving path %s: %w", filePath, err)
	}

	visited := make(map[string]bool)
	visited[absPath] = true
	result, _, err := resolveFileRecursive(absPath, confDir, visited)
	if err != nil {
		return "", err
	}

	out, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling resolved config: %w", err)
	}
	return string(out), nil
}
