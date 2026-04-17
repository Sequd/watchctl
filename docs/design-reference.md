# Design Reference — Shared TUI Design System

Source: [dev-worktree-flow](https://github.com/ekoru/dev-flow-tui) — shared design system across Go TUI apps.

## TUI Stack

Charmbracelet ecosystem: Bubble Tea + Lipgloss + Bubbles.

## Color Palette

| Role       | Color     |
|------------|-----------|
| Accent     | `#5F9FFF` |
| OK/Green   | `#68D391` |
| Warn       | `#F6C950` |
| Error      | `#FC8181` |
| Muted      | `#A0AEC0` |
| Border     | `#4A5568` |
| Select bg  | `#2C5282` |

All style definitions live in `internal/ui/styles.go`.

## Layout Conventions

- Two-column layout for main content where applicable
- Tab navigation with `●` indicator for active tab
- Help bar at the bottom with `│` separators
- Status bar for transient messages (3s auto-dismiss)
- Badge styles (colored background) for status indicators: OK, Warn, Error, Accent

## Keyboard Shortcut Table Format

| Key | Action |
|---|---|
| `↑`/`k` | Move up |
| `↓`/`j` | Move down |
| `Tab` | Switch screen |
| `q` / `Esc` | Quit / back |

## Reference App: dev-flow-tui

Terminal UI for managing Git Worktrees and dev workflows.

### Features
- Worktree management: list, create, delete git worktrees
- IDE integration: open in JetBrains Rider or VS Code
- Codex CLI: launch Codex in any worktree
- Docker Compose: start/stop from TUI
- Process tracking: see which worktrees have running processes

### Project Structure
```
dev-flow-tui/
├── cmd/dev-flow/main.go
├── internal/
│   ├── git/worktree.go
│   ├── process/manager.go
│   ├── ide/launcher.go
│   ├── codex/codex.go
│   ├── docker/compose.go
│   └── ui/
│       ├── model.go
│       ├── view.go
│       ├── keys.go
│       └── styles.go
```
