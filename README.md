# Claude Session Manager (CSM)

A powerful terminal UI (TUI) application for managing multiple Claude Code instances using tmux. Inspired by [Claude Squad](https://github.com/smtg-ai/claude-squad).

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## Features

- **Multi-Session Management** - Run and manage multiple Claude Code instances simultaneously
- **Live Preview** - Real-time preview of Claude's output with ANSI color support
- **Session Resume** - Resume previous Claude conversations from any project
- **Activity Indicators** - Visual indicators showing active vs idle sessions
- **Custom Colors** - Personalize sessions with foreground colors, background colors, and gradients
- **Prompt Sending** - Send messages to running sessions without attaching
- **Session Reordering** - Organize sessions with keyboard shortcuts
- **Compact Mode** - Toggle spacing between sessions for denser view

## Installation

### Prerequisites

- Go 1.21 or later
- tmux
- [Claude Code CLI](https://github.com/anthropics/claude-code) installed and configured

### Build from Source

```bash
git clone https://github.com/izll/claude-session-manager.git
cd claude-session-manager
go build -o csm .
```

### Install to PATH

```bash
# Linux/macOS
cp csm ~/.local/bin/
# or
sudo cp csm /usr/local/bin/
```

## Usage

Simply run:

```bash
csm
```

### Keyboard Shortcuts

#### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `J` / `Shift+↓` | Move session down (reorder) |
| `K` / `Shift+↑` | Move session up (reorder) |

#### Session Actions
| Key | Action |
|-----|--------|
| `Enter` | Start (if stopped) and attach to session |
| `s` | Start session without attaching |
| `x` | Stop session |
| `n` | Create new session |
| `e` | Rename session |
| `r` | Resume previous Claude conversation |
| `p` | Send prompt/message to running session |
| `d` | Delete session |

#### Customization
| Key | Action |
|-----|--------|
| `c` | Change session color |
| `l` | Toggle compact mode |
| `y` | Toggle auto-yes mode (`--dangerously-skip-permissions`) |

#### Other
| Key | Action |
|-----|--------|
| `?` | Show help |
| `q` | Quit |

### Inside Attached Session
| Key | Action |
|-----|--------|
| `Ctrl+q` | Detach from session (quick) |
| `Ctrl+b d` | Detach from session (tmux default) |

## Color Customization

Press `c` to open the color picker for the selected session:

- **Foreground Colors** - 22 solid colors + 15 gradients
- **Background Colors** - 22 solid colors
- **Auto Mode** - Automatically picks contrasting text color
- **Full Row Mode** - Extend background color to full row width (press `f` to toggle)
- **Gradients** - Rainbow, Sunset, Ocean, Forest, Fire, Ice, Neon, Galaxy, Pastel, and more!

Use `Tab` to switch between foreground and background color selection.

## Session Resume

CSM can resume previous Claude Code conversations:

1. Press `r` on any session
2. Browse through previous conversations (shows last message and timestamp)
3. Select a conversation to resume or start fresh

## Activity Indicators

Sessions show different status indicators:

- `●` Orange - Active (Claude is working)
- `●` Gray - Idle (waiting for input)
- `○` Red outline - Stopped

## Configuration

Sessions are stored in `~/.config/claude-session-manager/sessions.json`.

Each session stores:
- Name and path
- Color settings
- Resume session ID
- Auto-yes preference

## Architecture

```
claude-session-manager/
├── main.go           # Entry point
├── cmd/              # CLI commands
├── config/           # Configuration handling
├── session/          # Session management & tmux integration
│   ├── instance.go   # Instance lifecycle
│   ├── storage.go    # Persistence
│   └── claude_sessions.go  # Claude session discovery
└── ui/               # Bubbletea TUI
    └── model.go      # UI model and views
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Inspired by [Claude Squad](https://github.com/smtg-ai/claude-squad)
- Built with [Charm](https://charm.sh/) libraries
- Powered by [Claude Code](https://github.com/anthropics/claude-code)
