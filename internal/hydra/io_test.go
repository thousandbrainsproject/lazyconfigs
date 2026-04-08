package hydra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture creates a YAML file in dir and returns its absolute path.
func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}
	return abs
}

func TestListVariants(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		dir := t.TempDir()
		got, err := ListVariants(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("several yaml files sorted", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"charlie.yaml", "alpha.yaml", "bravo.yaml"} {
			writeFixture(t, dir, name, "key: val\n")
		}
		got, err := ListVariants(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"alpha", "bravo", "charlie"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("mixed extensions", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "good.yaml", "a: 1\n")
		writeFixture(t, dir, "bad.txt", "text\n")
		writeFixture(t, dir, "also_good.yaml", "b: 2\n")
		got, err := ListVariants(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2, got %v", got)
		}
	})

	t.Run("subdirs skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "good.yaml", "a: 1\n")
		os.MkdirAll(filepath.Join(dir, "subdir.yaml"), 0o755)
		got, err := ListVariants(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1, got %v", got)
		}
	})

	t.Run("nonexistent dir", func(t *testing.T) {
		_, err := ListVariants("/nonexistent/path/xyz")
		if err == nil {
			t.Fatal("expected error for nonexistent dir")
		}
	})
}

func TestBuildTree(t *testing.T) {
	t.Run("leaf file no defaults", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "leaf.yaml", "key: value\n")
		nodes, err := BuildTree(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 0 {
			t.Errorf("expected 0 nodes for leaf, got %d", len(nodes))
		}
	})

	t.Run("single level defaults", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "model/gpt4.yaml", "layers: 12\n")
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - model: gpt4\n")
		nodes, err := BuildTree(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].Key != "model" {
			t.Errorf("Key = %q, want %q", nodes[0].Key, "model")
		}
		if nodes[0].Value != "gpt4" {
			t.Errorf("Value = %q, want %q", nodes[0].Value, "gpt4")
		}
		if !nodes[0].IsLeaf {
			t.Error("expected leaf node")
		}
	})

	t.Run("two levels deep", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "sensor/camera.yaml", "resolution: 640\n")
		writeFixture(t, dir, "model/gpt4.yaml", "defaults:\n  - sensor: camera\nlayers: 12\n")
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - model: gpt4\n")
		nodes, err := BuildTree(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(nodes))
		}
		// model node should have sensor as child
		// Note: sensor is relative to model/gpt4.yaml's dir
		modelNode := nodes[0]
		if len(modelNode.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(modelNode.Children))
		}
		if modelNode.Children[0].Key != "sensor" {
			t.Errorf("child Key = %q, want %q", modelNode.Children[0].Key, "sensor")
		}
	})

	t.Run("missing child file", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - model: nonexistent\n")
		nodes, err := BuildTree(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].Error != "file not found" {
			t.Errorf("Error = %q, want %q", nodes[0].Error, "file not found")
		}
		if !nodes[0].IsLeaf {
			t.Error("expected IsLeaf for missing file")
		}
	})

	t.Run("circular reference", func(t *testing.T) {
		dir := t.TempDir()
		// exp.yaml has "- model: alpha" -> model/alpha.yaml
		// model/alpha.yaml has "- model: beta" -> model/model/beta.yaml
		// model/model/beta.yaml has "- model: alpha" -> model/model/model/alpha.yaml
		// Instead, use absolute paths to create a real cycle:
		// exp.yaml -> model/alpha.yaml -> model/beta.yaml -> model/alpha.yaml
		writeFixture(t, dir, "model/alpha.yaml", "defaults:\n  - /model: beta\nval: a\n")
		writeFixture(t, dir, "model/beta.yaml", "defaults:\n  - /model: alpha\nval: b\n")
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - /model: alpha\n")
		nodes, err := BuildTree(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		// Should complete without infinite loop; cycle node should have error
		if len(nodes) == 0 {
			t.Fatal("expected at least 1 node")
		}
		// Walk to find the circular reference error
		var foundCycle bool
		var walk func([]*TreeNode)
		walk = func(ns []*TreeNode) {
			for _, n := range ns {
				if n.Error == "circular reference" {
					foundCycle = true
				}
				walk(n.Children)
			}
		}
		walk(nodes)
		if !foundCycle {
			t.Error("expected to find a circular reference error in the tree")
		}
	})
}

func TestResolveFile(t *testing.T) {
	t.Run("no defaults", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "simple.yaml", "key: value\nother: 123\n")
		got, err := ResolveFile(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "key: value") {
			t.Errorf("expected resolved output to contain 'key: value', got:\n%s", got)
		}
	})

	t.Run("one default merge", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "base/default.yaml", "base_key: base_val\n")
		f := writeFixture(t, dir, "main.yaml", "defaults:\n  - base: default\nown_key: own_val\n")
		got, err := ResolveFile(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "own_key: own_val") {
			t.Errorf("should contain own content, got:\n%s", got)
		}
	})

	t.Run("deep merge override", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, "base/default.yaml", "# @package _global_\nshared: from_base\nonly_base: yes\n")
		f := writeFixture(t, dir, "main.yaml", "# @package _global_\ndefaults:\n  - base: default\nshared: from_main\n")
		got, err := ResolveFile(f, dir)
		if err != nil {
			t.Fatal(err)
		}
		// main's "shared" should override base's
		if !strings.Contains(got, "shared: from_main") {
			t.Errorf("expected main to override base for 'shared', got:\n%s", got)
		}
		if !strings.Contains(got, "only_base: \"yes\"") && !strings.Contains(got, "only_base: yes") {
			t.Errorf("expected base-only key to be preserved, got:\n%s", got)
		}
	})

	t.Run("missing file error", func(t *testing.T) {
		_, err := ResolveFile("/nonexistent/file.yaml", "/nonexistent")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestUpdateDefaultValue(t *testing.T) {
	t.Run("simple swap", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - model: gpt4\n")
		err := UpdateDefaultValue(f, "model", "gpt3")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(f)
		if !strings.Contains(string(data), "model: gpt3") {
			t.Errorf("expected 'model: gpt3', got:\n%s", string(data))
		}
	})

	t.Run("preserves comments", func(t *testing.T) {
		dir := t.TempDir()
		content := "# This is important\ndefaults:\n  - model: old_val  # inline comment\nother: stuff\n"
		f := writeFixture(t, dir, "exp.yaml", content)
		err := UpdateDefaultValue(f, "model", "new_val")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(f)
		s := string(data)
		if !strings.Contains(s, "# This is important") {
			t.Error("top comment lost")
		}
		if !strings.Contains(s, "model: new_val") {
			t.Error("value not updated")
		}
		if !strings.Contains(s, "other: stuff") {
			t.Error("other content lost")
		}
	})

	t.Run("preserves other entries", func(t *testing.T) {
		dir := t.TempDir()
		content := "defaults:\n  - model: gpt4\n  - sensor: camera\n"
		f := writeFixture(t, dir, "exp.yaml", content)
		err := UpdateDefaultValue(f, "model", "gpt3")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(f)
		s := string(data)
		if !strings.Contains(s, "model: gpt3") {
			t.Error("model not updated")
		}
		if !strings.Contains(s, "sensor: camera") {
			t.Error("sensor entry lost")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "exp.yaml", "defaults:\n  - model: gpt4\n")
		err := UpdateDefaultValue(f, "nonexistent", "val")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("no defaults key", func(t *testing.T) {
		dir := t.TempDir()
		f := writeFixture(t, dir, "exp.yaml", "key: value\n")
		err := UpdateDefaultValue(f, "model", "val")
		if err == nil {
			t.Fatal("expected error for missing defaults")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		err := UpdateDefaultValue("/nonexistent/file.yaml", "model", "val")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}
