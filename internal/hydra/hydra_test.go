package hydra

import (
	"testing"
)

func TestParseDefaultKey(t *testing.T) {
	tests := []struct {
		name        string
		rawKey      string
		value       string
		wantKey     string
		wantAbs     bool
		wantPkg     string
		wantRawKey  string
	}{
		{
			name:       "simple key with value",
			rawKey:     "model",
			value:      "gpt4",
			wantKey:    "model",
			wantRawKey: "model",
		},
		{
			name:       "absolute key",
			rawKey:     "/monty",
			value:      "informed",
			wantKey:    "monty",
			wantAbs:    true,
			wantRawKey: "/monty",
		},
		{
			name:       "key with package",
			rawKey:     "config@_group_",
			value:      "base",
			wantKey:    "config",
			wantPkg:    "_group_",
			wantRawKey: "config@_group_",
		},
		{
			name:       "absolute with package",
			rawKey:     "/foo@bar.baz",
			value:      "val",
			wantKey:    "foo",
			wantAbs:    true,
			wantPkg:    "bar.baz",
			wantRawKey: "/foo@bar.baz",
		},
		{
			name:       "bare string no value",
			rawKey:     "_self_",
			value:      "",
			wantKey:    "_self_",
			wantRawKey: "_self_",
		},
		{
			name:       "nested key with slash and package",
			rawKey:     "hypotheses_updater/burst@lm_0.args",
			value:      "",
			wantKey:    "hypotheses_updater/burst",
			wantPkg:    "lm_0.args",
			wantRawKey: "hypotheses_updater/burst@lm_0.args",
		},
		{
			name:       "empty raw key",
			rawKey:     "",
			value:      "",
			wantKey:    "",
			wantRawKey: "",
		},
		{
			name:       "at sign only",
			rawKey:     "@pkg",
			value:      "",
			wantKey:    "",
			wantPkg:    "pkg",
			wantRawKey: "@pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDefaultKey(tt.rawKey, tt.value)
			if got.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", got.Key, tt.wantKey)
			}
			if got.Value != tt.value {
				t.Errorf("Value = %q, want %q", got.Value, tt.value)
			}
			if got.Absolute != tt.wantAbs {
				t.Errorf("Absolute = %v, want %v", got.Absolute, tt.wantAbs)
			}
			if got.PackagePath != tt.wantPkg {
				t.Errorf("PackagePath = %q, want %q", got.PackagePath, tt.wantPkg)
			}
			if got.RawKey != tt.wantRawKey {
				t.Errorf("RawKey = %q, want %q", got.RawKey, tt.wantRawKey)
			}
		})
	}
}

func TestParseDefaultsFromData(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []DefaultEntry
		wantNil bool
		wantErr bool
	}{
		{
			name:    "no defaults key",
			yaml:    "foo: bar\n",
			wantNil: true,
		},
		{
			name: "empty defaults list",
			yaml: "defaults: []\n",
			want: nil,
		},
		{
			name: "single string entry",
			yaml: "defaults:\n  - model\n",
			want: []DefaultEntry{
				{Key: "model", RawKey: "model"},
			},
		},
		{
			name: "single map entry",
			yaml: "defaults:\n  - model: gpt4\n",
			want: []DefaultEntry{
				{Key: "model", Value: "gpt4", RawKey: "model"},
			},
		},
		{
			name: "mixed entries",
			yaml: "defaults:\n  - _self_\n  - model: gpt4\n",
			want: []DefaultEntry{
				{Key: "_self_", RawKey: "_self_"},
				{Key: "model", Value: "gpt4", RawKey: "model"},
			},
		},
		{
			name: "absolute entry",
			yaml: "defaults:\n  - /monty: informed\n",
			want: []DefaultEntry{
				{Key: "monty", Value: "informed", Absolute: true, RawKey: "/monty"},
			},
		},
		{
			name: "entry with package directive",
			yaml: "defaults:\n  - config@pkg.path: val\n",
			want: []DefaultEntry{
				{Key: "config", Value: "val", PackagePath: "pkg.path", RawKey: "config@pkg.path"},
			},
		},
		{
			name: "null value in map",
			yaml: "defaults:\n  - model: null\n",
			want: []DefaultEntry{
				{Key: "model", Value: "", RawKey: "model"},
			},
		},
		{
			name:    "invalid yaml",
			yaml:    ":\n  - :\n  [invalid",
			wantErr: true,
		},
		{
			name:    "defaults is not a list",
			yaml:    "defaults: notalist\n",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDefaultsFromData([]byte(tt.yaml))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Key != tt.want[i].Key {
					t.Errorf("[%d] Key = %q, want %q", i, got[i].Key, tt.want[i].Key)
				}
				if got[i].Value != tt.want[i].Value {
					t.Errorf("[%d] Value = %q, want %q", i, got[i].Value, tt.want[i].Value)
				}
				if got[i].Absolute != tt.want[i].Absolute {
					t.Errorf("[%d] Absolute = %v, want %v", i, got[i].Absolute, tt.want[i].Absolute)
				}
				if got[i].RawKey != tt.want[i].RawKey {
					t.Errorf("[%d] RawKey = %q, want %q", i, got[i].RawKey, tt.want[i].RawKey)
				}
				if got[i].PackagePath != tt.want[i].PackagePath {
					t.Errorf("[%d] PackagePath = %q, want %q", i, got[i].PackagePath, tt.want[i].PackagePath)
				}
			}
		})
	}
}

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name           string
		entry          DefaultEntry
		parentFilePath string
		confDir        string
		want           string
	}{
		{
			name:           "relative with value",
			entry:          DefaultEntry{Key: "model", Value: "gpt4"},
			parentFilePath: "/a/b/exp.yaml",
			confDir:        "/a",
			want:           "/a/b/model/gpt4.yaml",
		},
		{
			name:           "relative without value",
			entry:          DefaultEntry{Key: "model"},
			parentFilePath: "/a/b/exp.yaml",
			confDir:        "/a",
			want:           "/a/b/model.yaml",
		},
		{
			name:           "absolute with value",
			entry:          DefaultEntry{Key: "monty", Value: "informed", Absolute: true},
			parentFilePath: "/a/b/exp.yaml",
			confDir:        "/conf",
			want:           "/conf/monty/informed.yaml",
		},
		{
			name:           "absolute without value",
			entry:          DefaultEntry{Key: "monty", Absolute: true},
			parentFilePath: "/a/b/exp.yaml",
			confDir:        "/conf",
			want:           "/conf/monty.yaml",
		},
		{
			name:           "nested key with slashes",
			entry:          DefaultEntry{Key: "a/b", Value: "c"},
			parentFilePath: "/x/y/exp.yaml",
			confDir:        "/x",
			want:           "/x/y/a/b/c.yaml",
		},
		{
			name:           "absolute nested key",
			entry:          DefaultEntry{Key: "a/b", Value: "c", Absolute: true},
			parentFilePath: "/x/y/exp.yaml",
			confDir:        "/conf",
			want:           "/conf/a/b/c.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveFilePath(tt.entry, tt.parentFilePath, tt.confDir)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTreeNodePackageDir(t *testing.T) {
	tests := []struct {
		name    string
		node    *TreeNode
		confDir string
		want    string
	}{
		{
			name:    "absolute node",
			node:    &TreeNode{Key: "monty", Absolute: true},
			confDir: "/conf",
			want:    "/conf/monty",
		},
		{
			name:    "relative node",
			node:    &TreeNode{Key: "model", SourceFilePath: "/conf/experiment/exp.yaml"},
			confDir: "/conf",
			want:    "/conf/experiment/model",
		},
		{
			name:    "deeply nested key",
			node:    &TreeNode{Key: "a/b", Absolute: true},
			confDir: "/conf",
			want:    "/conf/a/b",
		},
		{
			name:    "root-level relative",
			node:    &TreeNode{Key: "model", SourceFilePath: "/conf/exp.yaml"},
			confDir: "/conf",
			want:    "/conf/model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.PackageDir(tt.confDir)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTreeContainsFile(t *testing.T) {
	tests := []struct {
		name       string
		nodes      []*TreeNode
		targetPath string
		want       bool
	}{
		{
			name:       "empty tree",
			nodes:      nil,
			targetPath: "/a.yaml",
			want:       false,
		},
		{
			name:       "match at root",
			nodes:      []*TreeNode{{FilePath: "/a.yaml"}},
			targetPath: "/a.yaml",
			want:       true,
		},
		{
			name: "match in children",
			nodes: []*TreeNode{
				{FilePath: "/a.yaml", Children: []*TreeNode{
					{FilePath: "/b.yaml"},
				}},
			},
			targetPath: "/b.yaml",
			want:       true,
		},
		{
			name: "no match",
			nodes: []*TreeNode{
				{FilePath: "/a.yaml", Children: []*TreeNode{
					{FilePath: "/b.yaml"},
				}},
			},
			targetPath: "/c.yaml",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := treeContainsFile(tt.nodes, tt.targetPath)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindRefsInTree(t *testing.T) {
	tests := []struct {
		name       string
		nodes      []*TreeNode
		targetPath string
		expName    string
		wantCount  int
	}{
		{
			name:       "no refs",
			nodes:      []*TreeNode{{FilePath: "/a.yaml"}},
			targetPath: "/b.yaml",
			expName:    "exp1",
			wantCount:  0,
		},
		{
			name: "single ref at root",
			nodes: []*TreeNode{
				{FilePath: "/target.yaml", SourceFilePath: "/src.yaml", RawKey: "model"},
			},
			targetPath: "/target.yaml",
			expName:    "exp1",
			wantCount:  1,
		},
		{
			name: "nested ref",
			nodes: []*TreeNode{
				{FilePath: "/a.yaml", Children: []*TreeNode{
					{FilePath: "/b.yaml", Children: []*TreeNode{
						{FilePath: "/target.yaml", SourceFilePath: "/b.yaml", RawKey: "deep"},
					}},
				}},
			},
			targetPath: "/target.yaml",
			expName:    "exp1",
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := findRefsInTree(tt.nodes, tt.targetPath, tt.expName)
			if len(refs) != tt.wantCount {
				t.Errorf("got %d refs, want %d", len(refs), tt.wantCount)
			}
			for _, ref := range refs {
				if ref.ExperimentName != tt.expName {
					t.Errorf("ExperimentName = %q, want %q", ref.ExperimentName, tt.expName)
				}
			}
		})
	}
}
