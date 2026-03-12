package ui

import (
	"fmt"
	"strings"

	"lazyconfigs/internal/config"
	"lazyconfigs/internal/hydra"
)

// RenderItem produces a display string for a tree node with indent, icon, and label.
// When selected is true, the line is rendered bold with a colored cursor marker.
//
// Format examples:
//
//	▼ /monty: informed_5          (depth=0, expanded, has children)
//	  ▶ motor_system_config: x    (depth=1, collapsed, has children)
//	  · sensor_module: camera      (depth=1, leaf)
func RenderItem(node *hydra.TreeNode, selected bool, theme config.ThemeColors) string {
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

// RenderVariantItem produces a display string for a variant list item.
// cursorSelected indicates the cursor is on this item; isActive indicates this is the currently selected variant.
func RenderVariantItem(name string, cursorSelected bool, isActive bool, isDiffFrom bool, theme config.ThemeColors) string {
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
