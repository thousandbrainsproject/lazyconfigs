package config

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want tcell.Color
	}{
		{"empty string", "", tcell.ColorDefault},
		{"default keyword", "default", tcell.ColorDefault},
		{"named red", "red", tcell.ColorRed},
		{"named green", "green", tcell.ColorGreen},
		{"hex color", "#ff0000", tcell.NewRGBColor(255, 0, 0)},
		{"hex color mixed", "#6a9fb5", tcell.NewRGBColor(0x6a, 0x9f, 0xb5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseColor(tt.s)
			if got != tt.want {
				t.Errorf("parseColor(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseKeyDescriptor(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		wantKey tcell.Key
		wantCh  rune
		wantErr bool
	}{
		{"single lowercase", "j", tcell.KeyRune, 'j', false},
		{"single uppercase", "J", tcell.KeyRune, 'J', false},
		{"Enter", "Enter", tcell.KeyEnter, 0, false},
		{"Tab", "Tab", tcell.KeyTab, 0, false},
		{"Shift-Tab", "Shift-Tab", tcell.KeyBacktab, 0, false},
		{"Space", "Space", tcell.KeyRune, ' ', false},
		{"Esc", "Esc", tcell.KeyEsc, 0, false},
		{"Ctrl-c", "Ctrl-c", tcell.KeyCtrlC, 0, false},
		{"Ctrl-a", "Ctrl-a", tcell.KeyCtrlA, 0, false},
		{"Backspace", "Backspace", tcell.KeyBackspace2, 0, false},
		{"empty string", "", 0, 0, true},
		{"unknown multi-char", "FooBar", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, ch, err := parseKeyDescriptor(tt.s)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("key = %v, want %v", key, tt.wantKey)
			}
			if ch != tt.wantCh {
				t.Errorf("rune = %q, want %q", ch, tt.wantCh)
			}
		})
	}
}

func TestCompileTheme(t *testing.T) {
	t.Run("default colors", func(t *testing.T) {
		cfg := defaultConfig()
		theme := CompileTheme(cfg.Colors)
		if theme.BorderFocused == tcell.ColorDefault {
			t.Error("BorderFocused should not be default")
		}
		if theme.Tags.BorderFocused != cfg.Colors.BorderFocused {
			t.Error("Tags should preserve original strings")
		}
	})

	t.Run("custom hex colors", func(t *testing.T) {
		cc := ColorConfig{
			BorderFocused:   "#ff0000",
			BorderUnfocused: "#00ff00",
			Cursor:          "#0000ff",
		}
		theme := CompileTheme(cc)
		if theme.BorderFocused != tcell.NewRGBColor(255, 0, 0) {
			t.Error("hex color not compiled correctly")
		}
		if theme.Tags.BorderFocused != "#ff0000" {
			t.Error("Tags should preserve original string")
		}
	})
}

func TestCompileBindings(t *testing.T) {
	t.Run("default bindings", func(t *testing.T) {
		cfg := defaultConfig()
		cb := CompileBindings(cfg.Keybindings)

		// Check that all default general bindings compiled
		if _, ok := cb.General["quit"]; !ok {
			t.Error("quit binding not compiled")
		}
		if _, ok := cb.General["help"]; !ok {
			t.Error("help binding not compiled")
		}
		// Check reverse map
		quitBinding := cb.General["quit"]
		id := KeyID{Key: quitBinding.Key, Rune: quitBinding.Rune}
		if action, ok := cb.GeneralByKey[id]; !ok || action != "quit" {
			t.Error("reverse lookup for quit failed")
		}
	})

	t.Run("single binding per group", func(t *testing.T) {
		kc := KeybindingsConfig{
			General:  map[string]string{"quit": "q"},
			Builder:  map[string]string{"expand_collapse": "Enter"},
			Variants: map[string]string{"select": "Space"},
		}
		cb := CompileBindings(kc)
		if len(cb.General) != 1 {
			t.Errorf("expected 1 general binding, got %d", len(cb.General))
		}
		if len(cb.Builder) != 1 {
			t.Errorf("expected 1 builder binding, got %d", len(cb.Builder))
		}
		if len(cb.Variants) != 1 {
			t.Errorf("expected 1 variants binding, got %d", len(cb.Variants))
		}
	})

	t.Run("empty config", func(t *testing.T) {
		cb := CompileBindings(KeybindingsConfig{})
		if cb.General == nil || cb.Builder == nil || cb.Variants == nil {
			t.Error("maps should be non-nil even for empty config")
		}
	})
}

func TestShouldWarn(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name   string
		w      WarningConfig
		action ConfirmAction
		want   bool
	}{
		{"nil defaults to true", WarningConfig{}, ConfirmDelete, true},
		{"explicit true", WarningConfig{Delete: boolPtr(true)}, ConfirmDelete, true},
		{"explicit false", WarningConfig{Delete: boolPtr(false)}, ConfirmDelete, false},
		{"mixed actions", WarningConfig{Delete: boolPtr(false), Rename: boolPtr(true)}, ConfirmRename, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.w.ShouldWarn(tt.action)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateHelpText(t *testing.T) {
	cfg := defaultConfig()
	bindings := CompileBindings(cfg.Keybindings)

	t.Run("builder panel", func(t *testing.T) {
		text := GenerateHelpText(0, bindings)
		if !strings.Contains(text, "Builder") {
			t.Error("should contain Builder title")
		}
		if !strings.Contains(text, "Move cursor") {
			t.Error("should contain navigation help")
		}
	})

	t.Run("variants panel", func(t *testing.T) {
		text := GenerateHelpText(1, bindings)
		if !strings.Contains(text, "Variants") {
			t.Error("should contain Variants title")
		}
		if !strings.Contains(text, "Select this variant") {
			t.Error("should contain select help")
		}
	})

	t.Run("invalid panel", func(t *testing.T) {
		text := GenerateHelpText(99, bindings)
		if text != "" {
			t.Errorf("expected empty string for invalid panel, got %q", text)
		}
	})
}

func TestGenerateStatusBarText(t *testing.T) {
	cfg := defaultConfig()
	bindings := CompileBindings(cfg.Keybindings)

	t.Run("builder panel", func(t *testing.T) {
		text := GenerateStatusBarText(0, false, bindings)
		if !strings.Contains(text, "Navigate") {
			t.Error("should contain Navigate")
		}
		if !strings.Contains(text, "Expand") {
			t.Error("should contain Expand")
		}
	})

	t.Run("variants panel normal", func(t *testing.T) {
		text := GenerateStatusBarText(1, false, bindings)
		if !strings.Contains(text, "Select") {
			t.Error("should contain Select")
		}
		if !strings.Contains(text, "Diff") {
			t.Error("should contain Diff")
		}
	})

	t.Run("variants panel diff mode", func(t *testing.T) {
		text := GenerateStatusBarText(1, true, bindings)
		if !strings.Contains(text, "Exit diff") {
			t.Error("should contain Exit diff")
		}
	})

	t.Run("invalid panel", func(t *testing.T) {
		text := GenerateStatusBarText(99, false, bindings)
		if !strings.Contains(text, "Panel 99") {
			t.Errorf("expected fallback text, got %q", text)
		}
	})
}
