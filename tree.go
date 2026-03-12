package main

import (
	"fmt"
	"strings"
)

// collectExpanded returns a set of node identity keys (SourceFilePath + ":" + Key)
// for all expanded nodes in the tree.
func collectExpanded(roots []*TreeNode) map[string]bool {
	expanded := make(map[string]bool)
	var walk func([]*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, n := range nodes {
			if n.Expanded {
				expanded[n.SourceFilePath+":"+n.Key] = true
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(roots)
	return expanded
}

// restoreExpanded applies previously collected expanded state to a new tree.
func restoreExpanded(roots []*TreeNode, expanded map[string]bool) {
	var walk func([]*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, n := range nodes {
			if expanded[n.SourceFilePath+":"+n.Key] {
				n.Expanded = true
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(roots)
}

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
func renderItem(node *TreeNode, selected bool, theme ThemeColors) string {
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
	if node.Value == "??" || node.Error != "" {
		label = fmt.Sprintf("%s: [%s]%s[-]", label, theme.Tags.ValueError, node.Value)
	} else if node.Value != "" {
		label = fmt.Sprintf("%s: [%s]%s[-]", label, theme.Tags.ValueOk, node.Value)
	}

	if selected {
		return fmt.Sprintf("[::b]%s[%s]%s[-] %s[-:-:-]", indent, theme.Tags.Cursor, icon, label)
	}
	return fmt.Sprintf("[::d]%s%s %s[-:-:-]", indent, icon, label)
}

// renderVariantItem produces a display string for a variant list item.
// cursorSelected indicates the cursor is on this item; isActive indicates this is the currently selected variant.
func renderVariantItem(name string, cursorSelected bool, isActive bool, isDiffFrom bool, theme ThemeColors) string {
	if isDiffFrom {
		prefix := "  "
		if isActive {
			prefix = "* "
		}
		if cursorSelected {
			return fmt.Sprintf("[%s::b]%s%s[-:-:-]", theme.Tags.DiffFrom, prefix, name)
		}
		return fmt.Sprintf("[%s]%s%s[-]", theme.Tags.DiffFrom, prefix, name)
	}
	activeTag := theme.Tags.ActiveVariant
	switch {
	case isActive && cursorSelected:
		return fmt.Sprintf("[%s::b]* %s[-:-:-]", activeTag, name)
	case isActive:
		return fmt.Sprintf("[%s]* %s[-]", activeTag, name)
	case cursorSelected:
		return fmt.Sprintf("[::b]  %s[-:-:-]", name)
	default:
		return fmt.Sprintf("[::d]  %s[-:-:-]", name)
	}
}
