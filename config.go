// ABOUTME: File-based configuration system for the lazyconfigs TUI.
// ABOUTME: Loads config from $HOME/.config/lazyconfigs/config.yaml with sensible defaults.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration loaded from config.yaml.
type Config struct {
	ConfDir     string            `yaml:"conf_dir"`
	Editor      string            `yaml:"editor"`
	SyntaxStyle string            `yaml:"syntax_style"`
	Warnings    WarningConfig     `yaml:"warnings"`
	Colors      ColorConfig       `yaml:"colors"`
	Keybindings KeybindingsConfig `yaml:"keybindings"`
}

// WarningConfig controls whether confirmation modals are shown per action.
// Pointer fields: nil means "use default (true)".
type WarningConfig struct {
	Delete   *bool `yaml:"delete"`
	Rename   *bool `yaml:"rename"`
	Reassign *bool `yaml:"reassign"`
	Unassign *bool `yaml:"unassign"`
	Edit     *bool `yaml:"edit"`
}

// ShouldWarn returns whether a confirmation modal should be shown for the given action.
func (w WarningConfig) ShouldWarn(action confirmAction) bool {
	var ptr *bool
	switch action {
	case confirmDelete:
		ptr = w.Delete
	case confirmRename:
		ptr = w.Rename
	case confirmReassign:
		ptr = w.Reassign
	case confirmUnassign:
		ptr = w.Unassign
	case confirmEdit:
		ptr = w.Edit
	}
	if ptr == nil {
		return true
	}
	return *ptr
}

// ColorConfig holds string representations of all UI colors.
type ColorConfig struct {
	BorderFocused      string `yaml:"border_focused"`
	BorderUnfocused    string `yaml:"border_unfocused"`
	Cursor             string `yaml:"cursor"`
	DiffFrom           string `yaml:"diff_from"`
	ActiveVariant      string `yaml:"active_variant"`
	ModalDeleteBorder  string `yaml:"modal_delete_border"`
	ModalWarningBorder string `yaml:"modal_warning_border"`
	ModalHelpBorder    string `yaml:"modal_help_border"`
	ModalRefsBorder    string `yaml:"modal_refs_border"`
	DiffAdd            string `yaml:"diff_add"`
	DiffRemove         string `yaml:"diff_remove"`
	DiffHunk           string `yaml:"diff_hunk"`
	Error              string `yaml:"error"`
	ValueOk            string `yaml:"value_ok"`
	ValueError         string `yaml:"value_error"`
}

// KeybindingsConfig holds key descriptor strings grouped by context.
type KeybindingsConfig struct {
	General  map[string]string `yaml:"general"`
	Builder  map[string]string `yaml:"builder"`
	Variants map[string]string `yaml:"variants"`
}

// ThemeColors holds compiled tcell.Color values for all UI elements,
// plus original string tags for use in tview color markup.
type ThemeColors struct {
	BorderFocused      tcell.Color
	BorderUnfocused    tcell.Color
	Cursor             tcell.Color
	DiffFrom           tcell.Color
	ActiveVariant      tcell.Color
	ModalDeleteBorder  tcell.Color
	ModalWarningBorder tcell.Color
	ModalHelpBorder    tcell.Color
	ModalRefsBorder    tcell.Color
	DiffAdd            tcell.Color
	DiffRemove         tcell.Color
	DiffHunk           tcell.Color
	Error              tcell.Color
	ValueOk            tcell.Color
	ValueError         tcell.Color
	// Tags holds original color strings for tview markup (e.g. "red", "#6a9fb5").
	Tags ColorConfig
}

func defaultConfig() Config {
	return Config{
		Editor:      "vi",
		SyntaxStyle: "gruvbox",
		Warnings:    WarningConfig{},
		Colors: ColorConfig{
			BorderFocused:      "green",
			BorderUnfocused:    "default",
			Cursor:             "#6a9fb5",
			DiffFrom:           "#ff69b4",
			ActiveVariant:      "green",
			ModalDeleteBorder:  "red",
			ModalWarningBorder: "yellow",
			ModalHelpBorder:    "green",
			ModalRefsBorder:    "green",
			DiffAdd:            "green",
			DiffRemove:         "red",
			DiffHunk:           "yellow",
			Error:              "red",
			ValueOk:            "green",
			ValueError:         "red",
		},
		Keybindings: KeybindingsConfig{
			General: map[string]string{
				"quit":             "q",
				"help":             "?",
				"focus_builder":    "1",
				"focus_variants":   "2",
				"panel_next":       "l",
				"panel_prev":       "h",
				"panel_cycle_next": "Tab",
				"panel_cycle_prev": "Shift-Tab",
				"cursor_down":      "j",
				"cursor_up":        "k",
				"scroll_viewer_down": "J",
				"scroll_viewer_up":   "K",
				"toggle_resolved": "v",
				"search":          "/",
				"escape":          "Esc",
			},
			Builder: map[string]string{
				"expand_collapse": "Enter",
				"unassign":        "d",
			},
			Variants: map[string]string{
				"select":     "Space",
				"duplicate":  "d",
				"rename":     "r",
				"delete":     "D",
				"edit":       "e",
				"diff":       "w",
				"references": "Enter",
			},
		},
	}
}

// loadConfig reads config from $HOME/.config/lazyconfigs/config.yaml.
// Missing file returns defaults. Parse errors warn to stderr and return defaults.
func loadConfig() Config {
	cfg := defaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "lazyconfigs", "config.yaml"))
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: config parse error: %v, using defaults\n", err)
		return defaultConfig()
	}

	if cfg.ConfDir != "" {
		cfg.ConfDir = os.ExpandEnv(cfg.ConfDir)
	}

	return cfg
}

// findGitRoot walks up from cwd looking for a .git directory.
// Returns the default conf path under the git root, or an error if not found.
func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return filepath.Join(dir, "src", "tbp", "monty", "conf"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git directory found")
		}
		dir = parent
	}
}

// parseColor converts a color string (named or hex) to a tcell.Color.
func parseColor(s string) tcell.Color {
	if s == "" || s == "default" {
		return tcell.ColorDefault
	}
	return tcell.GetColor(s)
}

// compileTheme converts a ColorConfig into compiled ThemeColors.
func compileTheme(cc ColorConfig) ThemeColors {
	return ThemeColors{
		BorderFocused:      parseColor(cc.BorderFocused),
		BorderUnfocused:    parseColor(cc.BorderUnfocused),
		Cursor:             parseColor(cc.Cursor),
		DiffFrom:           parseColor(cc.DiffFrom),
		ActiveVariant:      parseColor(cc.ActiveVariant),
		ModalDeleteBorder:  parseColor(cc.ModalDeleteBorder),
		ModalWarningBorder: parseColor(cc.ModalWarningBorder),
		ModalHelpBorder:    parseColor(cc.ModalHelpBorder),
		ModalRefsBorder:    parseColor(cc.ModalRefsBorder),
		DiffAdd:            parseColor(cc.DiffAdd),
		DiffRemove:         parseColor(cc.DiffRemove),
		DiffHunk:           parseColor(cc.DiffHunk),
		Error:              parseColor(cc.Error),
		ValueOk:            parseColor(cc.ValueOk),
		ValueError:         parseColor(cc.ValueError),
		Tags:               cc,
	}
}

// keyID uniquely identifies a key event by its tcell key code and rune.
type keyID struct {
	Key  tcell.Key
	Rune rune
}

// KeyBinding holds a parsed key binding with its human-readable name.
type KeyBinding struct {
	Key  tcell.Key
	Rune rune
	Name string // human-readable: "j", "Enter", "Ctrl-d"
}

// CompiledBindings holds forward (action→binding) and reverse (key→action) maps per group.
type CompiledBindings struct {
	General      map[string]KeyBinding
	Builder      map[string]KeyBinding
	Variants     map[string]KeyBinding
	GeneralByKey  map[keyID]string
	BuilderByKey  map[keyID]string
	VariantsByKey map[keyID]string
}

// parseKeyDescriptor converts a key descriptor string into a tcell key and rune.
func parseKeyDescriptor(s string) (tcell.Key, rune, error) {
	if s == "" {
		return 0, 0, fmt.Errorf("empty key descriptor")
	}

	// Handle Ctrl- prefix
	if strings.HasPrefix(s, "Ctrl-") && len(s) == 6 {
		letter := rune(s[5])
		if unicode.IsLetter(letter) {
			upper := unicode.ToUpper(letter)
			key := tcell.KeyCtrlA + tcell.Key(upper-'A')
			return key, 0, nil
		}
		return 0, 0, fmt.Errorf("invalid Ctrl key descriptor: %q", s)
	}

	// Handle special keys
	switch s {
	case "Enter":
		return tcell.KeyEnter, 0, nil
	case "Tab":
		return tcell.KeyTab, 0, nil
	case "Shift-Tab":
		return tcell.KeyBacktab, 0, nil
	case "Space":
		return tcell.KeyRune, ' ', nil
	case "Esc":
		return tcell.KeyEsc, 0, nil
	case "Backspace":
		return tcell.KeyBackspace2, 0, nil
	}

	// Single character
	runes := []rune(s)
	if len(runes) == 1 {
		return tcell.KeyRune, runes[0], nil
	}

	return 0, 0, fmt.Errorf("unknown key descriptor: %q", s)
}

// compileBindings parses all keybinding config strings into compiled lookup maps.
func compileBindings(kc KeybindingsConfig) CompiledBindings {
	cb := CompiledBindings{
		General:       make(map[string]KeyBinding),
		Builder:       make(map[string]KeyBinding),
		Variants:      make(map[string]KeyBinding),
		GeneralByKey:  make(map[keyID]string),
		BuilderByKey:  make(map[keyID]string),
		VariantsByKey: make(map[keyID]string),
	}

	compileGroup(kc.General, cb.General, cb.GeneralByKey)
	compileGroup(kc.Builder, cb.Builder, cb.BuilderByKey)
	compileGroup(kc.Variants, cb.Variants, cb.VariantsByKey)

	return cb
}

// helpSection defines a titled group of entries in the help text.
type helpSection struct {
	title   string
	entries []helpEntry
}

// helpEntry maps action names to a human-readable label for help text generation.
type helpEntry struct {
	actions []string // action names to look up (paired: ["cursor_down", "cursor_up"] → "j / k")
	group   string   // "general", "builder", or "variants"
	label   string   // human description
}

// statusEntry defines a single entry in the status bar.
type statusEntry struct {
	label   string
	actions []string // 1 action or 2 for paired display
	group   string
}

var builderHelpSections = []helpSection{
	{title: "Navigation", entries: []helpEntry{
		{actions: []string{"cursor_down", "cursor_up"}, group: "general", label: "Move cursor up/down"},
		{actions: []string{"expand_collapse"}, group: "builder", label: "Expand/collapse node"},
		{actions: []string{"focus_builder"}, group: "general", label: "Jump to this panel"},
		{actions: []string{"panel_next", "panel_prev"}, group: "general", label: "Switch panels"},
		{actions: []string{"panel_cycle_next", "panel_cycle_prev"}, group: "general", label: "Cycle panels"},
		{actions: []string{"search"}, group: "general", label: "Search/filter items"},
	}},
	{title: "Actions", entries: []helpEntry{
		{actions: []string{"unassign"}, group: "builder", label: "Unassign package"},
	}},
	{title: "Viewer", entries: []helpEntry{
		{actions: []string{"scroll_viewer_down", "scroll_viewer_up"}, group: "general", label: "Scroll viewer"},
		{actions: []string{"toggle_resolved"}, group: "general", label: "Toggle resolved view"},
	}},
	{title: "General", entries: []helpEntry{
		{actions: []string{"help"}, group: "general", label: "This help"},
		{actions: []string{"escape"}, group: "general", label: "Close overlay"},
		{actions: []string{"quit"}, group: "general", label: "Quit"},
	}},
}

var variantsHelpSections = []helpSection{
	{title: "Navigation", entries: []helpEntry{
		{actions: []string{"cursor_down", "cursor_up"}, group: "general", label: "Move cursor up/down"},
		{actions: []string{"focus_variants"}, group: "general", label: "Jump to this panel"},
		{actions: []string{"panel_next", "panel_prev"}, group: "general", label: "Switch panels"},
		{actions: []string{"panel_cycle_next", "panel_cycle_prev"}, group: "general", label: "Cycle panels"},
		{actions: []string{"search"}, group: "general", label: "Search/filter items"},
	}},
	{title: "Actions", entries: []helpEntry{
		{actions: []string{"select"}, group: "variants", label: "Select this variant"},
		{actions: []string{"duplicate"}, group: "variants", label: "Duplicate variant"},
		{actions: []string{"rename"}, group: "variants", label: "Rename variant"},
		{actions: []string{"delete"}, group: "variants", label: "Delete variant (confirm)"},
		{actions: []string{"edit"}, group: "variants", label: "Edit in $EDITOR"},
		{actions: []string{"toggle_resolved"}, group: "general", label: "Toggle resolved view"},
		{actions: []string{"diff"}, group: "variants", label: "Diff from this variant"},
		{actions: []string{"references"}, group: "variants", label: "Show experiment references"},
	}},
	{title: "Viewer", entries: []helpEntry{
		{actions: []string{"scroll_viewer_down", "scroll_viewer_up"}, group: "general", label: "Scroll viewer"},
	}},
	{title: "General", entries: []helpEntry{
		{actions: []string{"help"}, group: "general", label: "This help"},
		{actions: []string{"escape"}, group: "general", label: "Exit diff / Close overlay"},
		{actions: []string{"quit"}, group: "general", label: "Quit"},
	}},
}

var builderStatusEntries = []statusEntry{
	{label: "Navigate", actions: []string{"cursor_down", "cursor_up"}, group: "general"},
	{label: "Expand", actions: []string{"expand_collapse"}, group: "builder"},
	{label: "Unassign", actions: []string{"unassign"}, group: "builder"},
	{label: "Panels", actions: []string{"panel_next", "panel_prev"}, group: "general"},
	{label: "Scroll", actions: []string{"scroll_viewer_down", "scroll_viewer_up"}, group: "general"},
	{label: "Resolve", actions: []string{"toggle_resolved"}, group: "general"},
	{label: "Search", actions: []string{"search"}, group: "general"},
	{label: "Help", actions: []string{"help"}, group: "general"},
	{label: "Quit", actions: []string{"quit"}, group: "general"},
}

var variantsStatusEntries = []statusEntry{
	{label: "Navigate", actions: []string{"cursor_down", "cursor_up"}, group: "general"},
	{label: "Refs", actions: []string{"references"}, group: "variants"},
	{label: "Select", actions: []string{"select"}, group: "variants"},
	{label: "Dup", actions: []string{"duplicate"}, group: "variants"},
	{label: "Rename", actions: []string{"rename"}, group: "variants"},
	{label: "Del", actions: []string{"delete"}, group: "variants"},
	{label: "Edit", actions: []string{"edit"}, group: "variants"},
	{label: "Resolve", actions: []string{"toggle_resolved"}, group: "general"},
	{label: "Diff", actions: []string{"diff"}, group: "variants"},
	{label: "Search", actions: []string{"search"}, group: "general"},
	{label: "Help", actions: []string{"help"}, group: "general"},
}

var variantsDiffStatusEntries = []statusEntry{
	{label: "Navigate", actions: []string{"cursor_down", "cursor_up"}, group: "general"},
	{label: "Scroll", actions: []string{"scroll_viewer_down", "scroll_viewer_up"}, group: "general"},
	{label: "Resolve", actions: []string{"toggle_resolved"}, group: "general"},
	{label: "Search", actions: []string{"search"}, group: "general"},
	{label: "Exit diff", actions: []string{"escape"}, group: "general"},
	{label: "Help", actions: []string{"help"}, group: "general"},
	{label: "Quit", actions: []string{"quit"}, group: "general"},
}

// bindingGroup returns the forward map for the given group name.
func bindingGroup(group string, b CompiledBindings) map[string]KeyBinding {
	switch group {
	case "builder":
		return b.Builder
	case "variants":
		return b.Variants
	default:
		return b.General
	}
}

// lookupKeyName returns the human-readable key name for an action in a group.
func lookupKeyName(action, group string, b CompiledBindings) string {
	if binding, ok := bindingGroup(group, b)[action]; ok {
		return binding.Name
	}
	return "?"
}

// generateHelpText builds the help modal text for the given panel from compiled bindings.
func generateHelpText(panelIdx int, bindings CompiledBindings) string {
	var sections []helpSection
	var title string
	switch panelIdx {
	case 0:
		sections = builderHelpSections
		title = "Builder"
	case 1:
		sections = variantsHelpSections
		title = "Variants"
	default:
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[yellow::b]%s — Help[-:-:-]\n", title))

	for _, section := range sections {
		sb.WriteString(fmt.Sprintf("\n[green]%s:[-]\n", section.title))
		for _, entry := range section.entries {
			var keyStr string
			if len(entry.actions) == 2 {
				k1 := lookupKeyName(entry.actions[0], entry.group, bindings)
				k2 := lookupKeyName(entry.actions[1], entry.group, bindings)
				keyStr = k1 + " / " + k2
			} else {
				keyStr = lookupKeyName(entry.actions[0], entry.group, bindings)
			}
			sb.WriteString(fmt.Sprintf("  %-14s%s\n", keyStr, entry.label))
		}
	}

	sb.WriteString("\n[darkgray]Press Escape to close[-]")
	return sb.String()
}

// generateStatusBarText builds the status bar text for the given panel from compiled bindings.
func generateStatusBarText(panelIdx int, diffMode bool, bindings CompiledBindings) string {
	var entries []statusEntry
	switch panelIdx {
	case 0:
		entries = builderStatusEntries
	case 1:
		if diffMode {
			entries = variantsDiffStatusEntries
		} else {
			entries = variantsStatusEntries
		}
	default:
		return fmt.Sprintf(" Panel %d", panelIdx)
	}

	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		var keyStr string
		if len(e.actions) == 2 {
			k1 := lookupKeyName(e.actions[0], e.group, bindings)
			k2 := lookupKeyName(e.actions[1], e.group, bindings)
			keyStr = k1 + "/" + k2
		} else {
			keyStr = lookupKeyName(e.actions[0], e.group, bindings)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", e.label, keyStr))
	}

	return " " + strings.Join(parts, " | ")
}

func compileGroup(descriptors map[string]string, forward map[string]KeyBinding, reverse map[keyID]string) {
	for action, desc := range descriptors {
		key, r, err := parseKeyDescriptor(desc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid key descriptor %q for action %q: %v\n", desc, action, err)
			continue
		}

		binding := KeyBinding{Key: key, Rune: r, Name: desc}
		forward[action] = binding

		id := keyID{Key: key, Rune: r}
		if existing, ok := reverse[id]; ok {
			fmt.Fprintf(os.Stderr, "warning: duplicate key %q: action %q overrides %q\n", desc, action, existing)
		}
		reverse[id] = action
	}
}
