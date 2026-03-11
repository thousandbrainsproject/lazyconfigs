package main

import (
	"fmt"
	"strings"
)

// flattenTree performs a depth-first walk that only descends into expanded nodes,
// producing the visible list of items for the builder panel.
func flattenTree(roots []*TreeNode) []*TreeNode {
	var result []*TreeNode
	for _, root := range roots {
		result = append(result, root)
		if root.Expanded && len(root.Children) > 0 {
			result = append(result, flattenTree(root.Children)...)
		}
	}
	return result
}

// renderItem produces a display string for a tree node with indent, icon, and label.
// When selected is true, the line is rendered bold with a colored cursor marker.
//
// Format examples:
//
//	▼ /monty: informed_5          (depth=0, expanded, has children)
//	  ▶ motor_system_config: x    (depth=1, collapsed, has children)
//	  · sensor_module: camera      (depth=1, leaf)
func renderItem(node *TreeNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)

	var icon string
	switch {
	case node.IsLeaf:
		icon = " "
	case node.Expanded:
		icon = "▼"
	default:
		icon = "▶"
	}

	var label string
	if node.Absolute {
		label = "/" + node.Key
	} else {
		label = node.Key
	}
	if node.Value != "" {
		label = fmt.Sprintf("%s: [green]%s[-]", label, node.Value)
	}

	if selected {
		return fmt.Sprintf("[::b]%s[#6a9fb5]%s[-] %s[-:-:-]", indent, icon, label)
	}
	return fmt.Sprintf("[::d]%s%s %s[-:-:-]", indent, icon, label)
}

// renderVariantItem produces a display string for a variant list item.
func renderVariantItem(name string, selected bool) string {
	if selected {
		return fmt.Sprintf("[::b]%s[-:-:-]", name)
	}
	return fmt.Sprintf("[::d]%s[-:-:-]", name)
}
