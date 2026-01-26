# Todoist TUI

A terminal-based Todoist client written in Go with Vim-style keybindings.

## Features

- Fast and lightweight terminal interface
- Vim keybindings (j/k, gg/G, dd, etc.)
- Full task management (create, edit, complete, delete)
- Support for projects, sections, labels, and subtasks
- Live search across all tasks
- Smart due date parsing
- Secure OAuth2 or API token authentication

## Installation

### Prerequisites

- Go 1.21 or higher
- A Todoist account

### From Source

```bash
git clone https://github.com/hy4ri/todoist-tui.git
cd todoist-tui
make build
# Binary will be in bin/todoist-tui
```

## Configuration

The application looks for a configuration file at `~/.config/todoist-tui/config.yaml`.

```yaml
auth:
  api_token: "your-api-token-here"
```

Alternatively, set the `TODOIST_API_TOKEN` environment variable.

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| j / k | Move up / down |
| gg / G | Go to top / bottom |
| Ctrl+u / Ctrl+d | Half page up / down |
| Tab | Switch between sidebar and tasks |
| Enter | Select project or open task details |
| Esc | Go back / Cancel |

### Views

| Key | Action |
|-----|--------|
| t | Today view |
| u | Upcoming view |
| p | Projects view |
| c | Calendar view |
| L | Labels view |

### Task Actions

| Key | Action |
|-----|--------|
| a | Add new task |
| e | Edit selected task |
| x | Toggle completion |
| dd | Delete task |
| 1-4 | Set priority |
| s | Add subtask |
| m | Move task to section |
| A | Add comment |
| ctrl+z | Undo last action |

### General

| Key | Action |
|-----|--------|
| / | Search |
| r | Refresh |
| ? | Toggle help |
| q | Quit |

## Development

```bash
make build   # Build binary
make run     # Run in development
make test    # Run all tests
make check   # Format, vet, and test
```

## License

MIT
