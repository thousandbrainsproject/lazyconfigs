package hydra

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// UpdateDefaultValue modifies a defaults: entry in a YAML file, preserving
// comments and structure by doing a surgical byte-level replacement.
// rawKey must match exactly what's in the YAML (e.g., "/monty", "config").
func UpdateDefaultValue(filePath, rawKey, newValue string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected YAML structure in %s", filePath)
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping at root of %s", filePath)
	}

	// Find the "defaults" key in the root mapping
	var defaultsNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "defaults" {
			defaultsNode = root.Content[i+1]
			break
		}
	}
	if defaultsNode == nil {
		return fmt.Errorf("no defaults: key found in %s", filePath)
	}
	if defaultsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("defaults: is not a sequence in %s", filePath)
	}

	// Walk sequence items to find the matching rawKey and its value node
	var valNode *yaml.Node
	for _, item := range defaultsNode.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(item.Content)-1; i += 2 {
			keyNode := item.Content[i]
			if keyNode.Value == rawKey {
				valNode = item.Content[i+1]
				break
			}
		}
		if valNode != nil {
			break
		}
	}
	if valNode == nil {
		return fmt.Errorf("key %q not found in defaults of %s", rawKey, filePath)
	}

	// Use the value node's line/column to do a surgical replacement
	// in the original file content, preserving all formatting.
	lines := strings.Split(string(data), "\n")
	lineIdx := valNode.Line - 1 // yaml.Node.Line is 1-based
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("value node line %d out of range in %s", valNode.Line, filePath)
	}

	line := lines[lineIdx]
	colIdx := valNode.Column - 1 // yaml.Node.Column is 1-based
	if colIdx < 0 || colIdx > len(line) {
		return fmt.Errorf("value node column %d out of range in %s", valNode.Column, filePath)
	}

	// Replace from the column position to end of the old value
	oldValue := valNode.Value
	rest := line[colIdx:]
	if !strings.HasPrefix(rest, oldValue) {
		return fmt.Errorf("expected %q at line %d col %d in %s, got %q",
			oldValue, valNode.Line, valNode.Column, filePath, rest)
	}
	lines[lineIdx] = line[:colIdx] + newValue + line[colIdx+len(oldValue):]

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}
