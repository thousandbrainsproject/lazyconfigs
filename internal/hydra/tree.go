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
	flattenInto(&result, roots)
	return result
}

func flattenInto(result *[]*TreeNode, nodes []*TreeNode) {
	for _, node := range nodes {
		*result = append(*result, node)
		if node.Expanded && len(node.Children) > 0 {
			flattenInto(result, node.Children)
		}
	}
}
