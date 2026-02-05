# Todoist TUI

A terminal-based Todoist client written in Go with Vim-style keybindings.

## Features

- Fast and lightweight terminal interface
- Vim keybindings (j/k, gg/G, dd, etc.)
- Full task management (create, edit, complete, delete)
- Support for projects, sections, labels, and subtasks
- Live search across all tasks
- Smart due date parsing
- Highly customizable color themes
- Secure token storage (system keyring or encrypted local data)
- Calendar view

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

## Getting Started

Simply run the application:

```bash
./bin/todoist-tui
```

On first launch, the app will:

1. Automatically create a default configuration file at `~/.config/todoist-tui/config.yaml`.
2. Prompt you to enter your **Todoist API Token** (find it [here](https://app.todoist.com/app/settings/integrations/developer)).

Your token is stored securely in your system keyring (or `~/.local/share/todoist-tui/.credentials`) and is **never** saved in the plain-text config file.

## Configuration

Customized settings and themes are managed in `~/.config/todoist-tui/config.yaml`.

### Themes

You can customize almost every color in the UI:

```yaml
ui:
  theme:
    highlight: "#FF6B6B"
    subtle: "#888888"
    priority_1: "#FF0000"
    status_bar_bg: "#1A1A2E"
```

### Startup

Set your preferred startup view:

```yaml
ui:
  default_view: "today"
  calendar_default_view: "compact"
```

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
| Shift + D | Set current view as default on startup |

### Views

| Key | Action |
|-----|--------|
| t | Today view |
| u | Upcoming view |
| p | Projects view |
| c | Calendar view |
| L | Labels view |
| i | Inbox view |

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
