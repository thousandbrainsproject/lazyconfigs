package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// updateDefaultValue modifies a defaults: entry in a YAML file, preserving
// comments and structure via the yaml.Node API.
// rawKey must match exactly what's in the YAML (e.g., "/monty", "config").
func updateDefaultValue(filePath, rawKey, newValue string) error {
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

	// Walk sequence items to find the matching rawKey
	found := false
	for _, item := range defaultsNode.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(item.Content)-1; i += 2 {
			keyNode := item.Content[i]
			valNode := item.Content[i+1]
			if keyNode.Value == rawKey {
				valNode.Value = newValue
				valNode.Tag = "!!str"
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("key %q not found in defaults of %s", rawKey, filePath)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, out, 0644)
}
