# lazyconfigs

A lazygit-inspired TUI for browsing and managing Hydra configuration hierarchies. Built for the Thousand Brains Project's Monty experiment configs.

```
╭─ [1] Builder ──────────╮╭─ Viewer ────────────────────────────────╮
│ ▼ /monty: informed_5   ││                                         │
│   ▶ motor_system_config ││  sensor_module: camera                  │
│   ▶ sensor_module       ││  resolution: 640                        │
│     learning_module: x  ││  frame_rate: 30                         │
│                         ││  ...                                    │
╰─────────────────────────╯│                                         │
╭─ [2] Variants ──────────╮│                                         │
│ * default               ││                                         │
│   variant_a             ││                                         │
│   variant_b             ││                                         │
╰─────────────────────────╯╰─────────────────────────────────────────╯
 Navigate: j/k | Expand: Enter | Help: ?     Thousand Brains Project 0.0.1
```

## Features

- **Hierarchical tree browser** -- expand/collapse Hydra config nodes
- **Variant management** -- select, rename, duplicate, delete, and edit variants
- **Unified diff** -- compare two variants side-by-side with colored output
- **Resolved view** -- see fully merged Hydra configuration (all defaults applied)
- **Fuzzy search** -- filter builder items or variants with incremental fuzzy matching
- **Reference tracking** -- see which experiments use a given variant before modifying it
- **Confirmation warnings** -- modal warnings before destructive cross-experiment changes
- **Syntax highlighting** -- YAML viewer with configurable Chroma styles
- **Fully configurable** -- colors, keybindings, warnings, and editor via YAML config

## Installation

Requires **Go 1.22+**.

```bash
cd tui
make install    # builds and copies binary to ~/.local/bin/
```

Other make targets:

```bash
make build      # build the binary locally
make run        # build and run
make clean      # remove built binary
```

## Usage

Run `lazyconfigs` from within a git repository that contains Hydra configs:

```bash
cd /path/to/your/project
lazyconfigs
```

By default, lazyconfigs walks up from the current directory to find the git root, then looks for configs at `<git_root>/src/tbp/monty/conf`. This can be overridden in the config file.

## Panels

| Panel | Description |
|-------|-------------|
| **Builder** `[1]` | Hierarchical tree of Hydra config keys and their current values. Expand/collapse nodes to navigate the config structure. |
| **Variants** `[2]` | Lists available variant files for the selected builder node. The active variant is marked with `*`. |
| **Viewer** | Displays the YAML content of the selected item with syntax highlighting. Shows unified diffs in diff mode. |

## Configuration

Create `~/.config/lazyconfigs/config.yaml` to customize behavior. All fields are optional -- omitted values use the defaults shown below.

```yaml
# Path to the Hydra config directory. Supports environment variables.
# If empty, falls back to <git_root>/src/tbp/monty/conf
conf_dir: ""

# Fallback editor when $EDITOR is unset
editor: "vi"

# Chroma syntax highlighting style (e.g. gruvbox, monokai, dracula)
syntax_style: "gruvbox"

# Per-action confirmation modals. Set to false to skip the modal.
warnings:
  delete: true
  rename: true
  reassign: true
  unassign: true
  edit: true

# UI colors. Accepts named colors (red, green) or hex (#rrggbb).
colors:
  border_focused: "green"
  border_unfocused: "default"
  cursor: "#6a9fb5"
  diff_from: "#ff69b4"
  active_variant: "green"
  modal_delete_border: "red"
  modal_warning_border: "yellow"
  modal_help_border: "green"
  modal_refs_border: "green"
  diff_add: "green"
  diff_remove: "red"
  diff_hunk: "yellow"
  error: "red"
  value_ok: "green"
  value_error: "red"

# Keybindings grouped by context. Accepts single chars, special keys
# (Enter, Tab, Esc, Space, Shift-Tab, Backspace), and Ctrl combos (Ctrl-d).
keybindings:
  general:
    quit: "q"
    help: "?"
    focus_builder: "1"
    focus_variants: "2"
    panel_next: "l"
    panel_prev: "h"
    panel_cycle_next: "Tab"
    panel_cycle_prev: "Shift-Tab"
    cursor_down: "j"
    cursor_up: "k"
    scroll_viewer_down: "J"
    scroll_viewer_up: "K"
    toggle_resolved: "v"
    search: "/"
    escape: "Esc"
  builder:
    expand_collapse: "Enter"
    unassign: "d"
  variants:
    select: "Space"
    duplicate: "d"
    rename: "r"
    delete: "D"
    edit: "e"
    diff: "w"
    references: "Enter"
```

### Configuration notes

- **`conf_dir`** supports environment variables (e.g. `$PROJECT_ROOT/src/tbp/monty/conf`). When empty, the app finds the git root and uses the default path.
- **Warnings** use `*bool` semantics -- omitted fields default to `true`. Set explicitly to `false` to disable.
- **Colors** accept any value tview understands: named colors (`red`, `green`, `yellow`, `default`) or hex (`#rrggbb`).
- **Keybindings** are grouped by context. Builder and variant bindings take priority over general bindings when their panel is focused. Modal keybindings (`y`/`n`/`Esc` in confirmations) are not configurable.

## Keybinding Reference

### General (all panels)

| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Show help |
| `1` / `2` | Focus builder / variants panel |
| `h` / `l` | Switch panels |
| `Tab` / `Shift-Tab` | Cycle panels |
| `j` / `k` | Move cursor down / up |
| `J` / `K` | Scroll viewer down / up |
| `v` | Toggle resolved view |
| `/` | Fuzzy search |
| `Esc` | Exit diff mode / close overlay / quit |

### Builder panel

| Key | Action |
|-----|--------|
| `Enter` | Expand / collapse node |
| `d` | Unassign selected node |

### Variants panel

| Key | Action |
|-----|--------|
| `Space` | Select variant (assign to builder node) |
| `d` | Duplicate variant |
| `r` | Rename variant |
| `D` | Delete variant |
| `e` | Edit in `$EDITOR` |
| `w` | Enter diff mode (compare from this variant) |
| `Enter` | Show experiment references |

## Project Structure

```
tui/
  main.go       Core app: layout, panels, keybindings, modal management
  config.go     Config loading, theme compilation, keybinding parsing
  hydra.go      Hydra config tree parsing and variant reference discovery
  resolve.go    Deep YAML merging and @package directive resolution
  tree.go       Tree flattening and item rendering
  viewer.go     Syntax highlighting and file/diff display
  diff.go       Unified diff generation and colorization
  search.go     Fuzzy search matching and filtered list management
  yamlwrite.go  Structure-preserving YAML modification
  Makefile      Build, install, clean, run targets
```

## License

Thousand Brains Project -- internal tooling.
