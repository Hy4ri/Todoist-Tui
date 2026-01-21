# Todoist TUI

A fast, terminal-based Todoist client written in Go with Vim-style keybindings.

![Todoist TUI Demo](https://img.shields.io/badge/status-beta-yellow)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- **⚡ Fast & Lightweight** - Terminal-based UI with minimal resource usage
- **⌨️ Vim Keybindings** - Navigate with j/k, gg/G, dd, and more
- **📋 Full Task Management** - Create, edit, complete, and delete tasks
- **🎨 Terminal Adaptive** - Beautiful colors that work in light and dark themes
- **🔍 Live Search** - Instant search across all your tasks
- **📁 Project Navigation** - Browse and filter tasks by project
- **🏷️ Priority & Labels** - Set priorities (P1-P4) and view labels
- **📅 Smart Due Dates** - Natural language dates like "tomorrow" or "next monday"
- **🔐 OAuth2 Authentication** - Secure authentication with Todoist

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/hy4ri/todoist-tui.git
cd todoist-tui

# Build and install
make install

# Or just build to bin/todoist-tui
make build
```

### Prerequisites

- Go 1.21 or higher
- A Todoist account (free or premium)
- Todoist API credentials (see Configuration)

## Configuration

### 1. Get Your Todoist API Token

**Option A: API Token (Quick Setup)**

1. Go to [Todoist Integrations](https://app.todoist.com/app/settings/integrations/developer)
2. Scroll down to "API token"
3. Copy your token

**Option B: OAuth2 (Recommended for Production)**

1. Go to [Todoist App Console](https://developer.todoist.com/appconsole.html)
2. Create a new app
3. Set callback URL to: `http://localhost:8080/callback`
4. Copy your Client ID and Client Secret

### 2. Configure the Application

Create the config directory:

```bash
mkdir -p ~/.config/todoist-tui
```

Create `~/.config/todoist-tui/config.yaml`:

```yaml
auth:
  # Option 1: Use API token (simple)
  api_token: "your-api-token-here"
  
  # Option 2: Use OAuth2 (recommended)
  # client_id: "your-client-id"
  # client_secret: "your-client-secret"

ui:
  vim_mode: true
```

**Alternative:** Set environment variables:

```bash
export TODOIST_API_TOKEN="your-api-token"
# Or for OAuth2:
export TODOIST_CLIENT_ID="your-client-id"
export TODOIST_CLIENT_SECRET="your-client-secret"
```

## Usage

### Starting the App

```bash
# If installed
todoist-tui

# If built locally
./bin/todoist-tui
```

### Keyboard Shortcuts

#### Navigation
| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `gg` / `G` | Go to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Half page down / up |
| `Tab` | Switch between sidebar and tasks |
| `Enter` | Select project or open task details |
| `Esc` | Go back / Cancel |

#### Task Actions
| Key | Action |
|-----|--------|
| `a` | Add new task |
| `e` | Edit selected task |
| `x` | Complete / uncomplete task |
| `dd` | Delete task |
| `1-4` | Set priority (1=highest, 4=lowest) |

#### Views & Features
| Key | Action |
|-----|--------|
| `/` | Search tasks |
| `r` | Refresh data |
| `?` | Show help |
| `q` | Quit |

### Task Form

When adding or editing a task (`a` or `e`):

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next / previous field |
| `Enter` | Submit form (when on Submit button) |
| `Esc` | Cancel |
| **Priority Field:** | |
| `1-4` | Set priority directly |
| `h` / `l` | Decrease / increase priority |
| **Project Field:** | |
| `Enter` | Open project dropdown |
| `j` / `k` | Navigate projects |
| `Enter` | Select project |

### Search

Press `/` to open search:

- Type to filter tasks in real-time
- Results show across all projects
- `j`/`k` to navigate results
- `Enter` to view task details
- `x` to complete/uncomplete
- `Esc` to close search

## Project Structure

```
todoist-tui/
├── cmd/
│   └── todoist-tui/
│       └── main.go           # Application entry point
├── internal/
│   ├── api/                  # Todoist REST API v2 client
│   │   ├── client.go         # HTTP client
│   │   ├── tasks.go          # Task operations
│   │   ├── projects.go       # Project operations
│   │   ├── sections.go       # Section operations
│   │   ├── labels.go         # Label operations
│   │   ├── comments.go       # Comment operations
│   │   ├── types.go          # Data structures
│   │   └── errors.go         # Error handling
│   ├── auth/
│   │   └── oauth.go          # OAuth2 authentication
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── tui/                  # Terminal UI
│       ├── app.go            # Main app model
│       ├── form.go           # Task form component
│       ├── keymap.go         # Vim keybindings
│       └── styles/
│           └── styles.go     # Lipgloss styles
├── go.mod
├── Makefile
└── README.md
```

## Development

### Build Commands

```bash
make build              # Build binary to bin/todoist-tui
make run                # Run in development
make install            # Install to $GOPATH/bin
make clean              # Remove build artifacts
```

### Testing

```bash
make test               # Run all tests
make test-one TEST=TestGetTasks    # Run specific test
make test-pkg PKG=./internal/api   # Test specific package
make test-cover         # Generate coverage report
```

### Code Quality

```bash
make fmt                # Format code (go fmt + goimports)
make vet                # Run go vet
make lint               # Run golangci-lint
make check              # fmt + vet + test (pre-commit)
```

## Architecture

### Bubble Tea Model

The app uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI:

- **Model**: `App` struct holds all application state
- **Update**: Processes keyboard input and API responses
- **View**: Renders the current view to terminal

### API Client

Direct integration with [Todoist REST API v2](https://developer.todoist.com/rest/v2/):

- Uses standard `net/http` with Bearer token authentication
- All endpoints return proper error types
- Request/response types match API spec

### Views

- **Today View**: Shows overdue and today's tasks (default)
- **Project View**: Tasks filtered by selected project
- **Task Detail**: Full task information
- **Task Form**: Add/edit task with all fields
- **Search**: Real-time task filtering
- **Help**: Keyboard shortcut reference

## Troubleshooting

### Authentication Issues

**Problem:** "Invalid token" or "Unauthorized"

```bash
# Verify your token works
curl -X GET https://api.todoist.com/rest/v2/tasks \
  -H "Authorization: Bearer YOUR_TOKEN"

# Check config file exists
cat ~/.config/todoist-tui/config.yaml
```

### Build Errors

**Problem:** `package not found`

```bash
# Update dependencies
go mod tidy
go mod download
```

### Display Issues

**Problem:** Weird characters or broken borders

- Ensure your terminal supports Unicode
- Try a different terminal (alacritty, kitty, iTerm2)
- Check `$TERM` environment variable:

```bash
echo $TERM
# Should be something like: xterm-256color
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Guidelines

- Follow Go best practices (see `AGENTS.md`)
- Add tests for new features
- Run `make check` before committing
- Keep commits atomic and descriptive

## Roadmap

- [ ] Subtasks support
- [ ] Comments view
- [ ] Recurring tasks
- [ ] Filters (custom Todoist filters)
- [ ] Collaboration features
- [ ] Task duration tracking
- [ ] Offline mode with sync

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework
- Styled with [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Powered by [Todoist API](https://developer.todoist.com/)

## Links

- [Todoist API Documentation](https://developer.todoist.com/rest/v2/)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Report Issues](https://github.com/hy4ri/todoist-tui/issues)

---

**Made with ❤️ by developers who love terminals**
