package hydra

import (
	"reflect"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name    string
		base    map[string]interface{}
		overlay map[string]interface{}
		want    map[string]interface{}
	}{
		{
			name:    "both empty",
			base:    map[string]interface{}{},
			overlay: map[string]interface{}{},
			want:    map[string]interface{}{},
		},
		{
			name:    "base only",
			base:    map[string]interface{}{"a": 1},
			overlay: map[string]interface{}{},
			want:    map[string]interface{}{"a": 1},
		},
		{
			name:    "overlay only",
			base:    map[string]interface{}{},
			overlay: map[string]interface{}{"b": 2},
			want:    map[string]interface{}{"b": 2},
		},
		{
			name:    "disjoint keys",
			base:    map[string]interface{}{"a": 1},
			overlay: map[string]interface{}{"b": 2},
			want:    map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:    "overlay scalar wins",
			base:    map[string]interface{}{"a": 1},
			overlay: map[string]interface{}{"a": 2},
			want:    map[string]interface{}{"a": 2},
		},
		{
			name: "nested map merge",
			base: map[string]interface{}{
				"a": map[string]interface{}{"x": 1, "y": 2},
			},
			overlay: map[string]interface{}{
				"a": map[string]interface{}{"y": 3, "z": 4},
			},
			want: map[string]interface{}{
				"a": map[string]interface{}{"x": 1, "y": 3, "z": 4},
			},
		},
		{
			name: "overlay scalar replaces map",
			base: map[string]interface{}{
				"a": map[string]interface{}{"x": 1},
			},
			overlay: map[string]interface{}{"a": "replaced"},
			want:    map[string]interface{}{"a": "replaced"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deepMerge(tt.base, tt.overlay)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeepMergeDoesNotMutateInputs(t *testing.T) {
	base := map[string]interface{}{"a": 1}
	overlay := map[string]interface{}{"b": 2}
	_ = deepMerge(base, overlay)
	if len(base) != 1 {
		t.Error("base was mutated")
	}
	if len(overlay) != 1 {
		t.Error("overlay was mutated")
	}
}

func TestSetNestedKey(t *testing.T) {
	tests := []struct {
		name    string
		dotPath string
		value   interface{}
		want    map[string]interface{}
	}{
		{
			name:    "single key",
			dotPath: "a",
			value:   1,
			want:    map[string]interface{}{"a": 1},
		},
		{
			name:    "dot path creates nested maps",
			dotPath: "a.b.c",
			value:   "deep",
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "deep",
					},
				},
			},
		},
		{
			name:    "overwrite existing",
			dotPath: "a",
			value:   "new",
			want:    map[string]interface{}{"a": "new"},
		},
		{
			name:    "empty dotPath is noop",
			dotPath: "",
			value:   "ignored",
			want:    map[string]interface{}{},
		},
		{
			name:    "intermediate non-map gets overwritten",
			dotPath: "a.b",
			value:   "val",
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "val",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[string]interface{})
			// Pre-populate for overwrite test
			if tt.name == "overwrite existing" {
				m["a"] = "old"
			}
			if tt.name == "intermediate non-map gets overwritten" {
				m["a"] = "scalar"
			}
			setNestedKey(m, tt.dotPath, tt.value)
			if !reflect.DeepEqual(m, tt.want) {
				t.Errorf("got %v, want %v", m, tt.want)
			}
		})
	}
}

func TestParsePackageDirective(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "no directive",
			data: "foo: bar\n",
			want: "",
		},
		{
			name: "global directive",
			data: "# @package _global_\nfoo: bar\n",
			want: "",
		},
		{
			name: "normal directive",
			data: "# @package foo.bar\nkey: val\n",
			want: "foo.bar",
		},
		{
			name: "directive after yaml content not found",
			data: "key: val\n# @package foo.bar\n",
			want: "",
		},
		{
			name: "directive with extra whitespace",
			data: "# @package   foo.bar  \nkey: val\n",
			want: "foo.bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePackageDirective([]byte(tt.data))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultPackageFromPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		confDir  string
		want     string
	}{
		{
			name:     "file at conf root",
			filePath: "/conf/file.yaml",
			confDir:  "/conf",
			want:     "",
		},
		{
			name:     "one level deep",
			filePath: "/conf/model/gpt4.yaml",
			confDir:  "/conf",
			want:     "model",
		},
		{
			name:     "two levels deep",
			filePath: "/conf/a/b/file.yaml",
			confDir:  "/conf",
			want:     "a.b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultPackageFromPath(tt.filePath, tt.confDir)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativePackage(t *testing.T) {
	tests := []struct {
		name      string
		parentPkg string
		childPkg  string
		want      string
	}{
		{
			name: "both empty",
			want: "",
		},
		{
			name:     "parent empty child has path",
			childPkg: "foo.bar",
			want:     "foo.bar",
		},
		{
			name:      "child extends parent",
			parentPkg: "foo",
			childPkg:  "foo.bar.baz",
			want:      "bar.baz",
		},
		{
			name:      "child equals parent",
			parentPkg: "foo.bar",
			childPkg:  "foo.bar",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativePackage(tt.parentPkg, tt.childPkg)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
