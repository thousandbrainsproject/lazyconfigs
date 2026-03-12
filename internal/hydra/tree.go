package hydra

// CollectExpanded returns a set of node identity keys (SourceFilePath + ":" + Key)
// for all expanded nodes in the tree.
func CollectExpanded(roots []*TreeNode) map[string]bool {
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

// RestoreExpanded applies previously collected expanded state to a new tree.
func RestoreExpanded(roots []*TreeNode, expanded map[string]bool) {
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

// FlattenTree performs a depth-first walk that only descends into expanded nodes,
// producing the visible list of items for the builder panel.
func FlattenTree(roots []*TreeNode) []*TreeNode {
	var result []*TreeNode
	for _, root := range roots {
		result = append(result, root)
		if root.Expanded && len(root.Children) > 0 {
			result = append(result, FlattenTree(root.Children)...)
		}
	}
	return result
}
