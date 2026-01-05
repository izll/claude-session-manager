# Agent Session Manager (ASMGR)

A powerful terminal UI (TUI) application for managing multiple AI coding assistant CLI sessions using tmux. Inspired by [Claude Squad](https://github.com/smtg-ai/claude-squad).

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## Supported AI Agents

- **Claude Code** - Anthropic's CLI coding assistant
- **Gemini CLI** - Google's AI assistant
- **Aider** - AI pair programming in your terminal
- **OpenAI Codex** - OpenAI's coding assistant
- **Amazon Q** - AWS AI coding companion
- **OpenCode** - Open-source AI coding assistant
- **Custom** - Any CLI command you want to manage

## Features

- **Projects/Workspaces** - Organize sessions into separate projects with isolated session lists
- **Single Instance Lock** - Only one instance of ASMGR can run per project at a time
- **Multi-Agent Support** - Run Claude, Gemini, Aider, Codex, Amazon Q, OpenCode, or custom commands
- **Multi-Session Management** - Run and manage multiple AI sessions simultaneously (multiple sessions can run in the same directory)
- **Parallel Sessions** - Start multiple instances of the same session with different names for working on multiple tasks
- **Live Preview** - Real-time preview of agent output with ANSI color support and proper wide character handling
- **Session Resume** - Resume previous conversations for Claude, Gemini, Codex, OpenCode, and Amazon Q
- **Activity Indicators** - Visual indicators showing active vs idle sessions
- **Custom Colors** - Personalize sessions with foreground colors, background colors, and gradients
- **Prompt Sending** - Send messages to running sessions without attaching (improved reliability for all agents)
- **Session Reordering** - Organize sessions with keyboard shortcuts
- **Compact Mode** - Toggle spacing between sessions for denser view
- **Smart Resize** - Terminal resize follows when attached, preview size preserved when detached
- **Overlay Dialogs** - Modal dialogs rendered over the main view with proper Unicode character width handling
- **Fancy Status Bar** - Styled bottom bar with highlighted keys, toggle indicators, and separators
- **Scrollable Help View** - Comprehensive help page with keyboard shortcuts, detailed descriptions, and scroll support
- **Session Groups** - Organize sessions into collapsible groups for better organization

## Installation

### Prerequisites

- Go 1.24 or later
- tmux
- At least one AI CLI tool installed:
  - [Claude Code](https://github.com/anthropics/claude-code)
  - [Gemini CLI](https://github.com/google-gemini/gemini-cli)
  - [Aider](https://github.com/paul-gauthier/aider)
  - [OpenAI Codex](https://github.com/openai/codex)
  - [Amazon Q](https://aws.amazon.com/q/)
  - [OpenCode](https://github.com/opencode-ai/opencode)

### Build from Source

```bash
git clone https://github.com/izll/agent-session-manager.git
cd agent-session-manager
go build -o asmgr .
```

### Install to PATH

```bash
# Linux/macOS
cp asmgr ~/.local/bin/
# or
sudo cp asmgr /usr/local/bin/
```

## Usage

Simply run:

```bash
asmgr
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
| `a` | Start session with options: replace current or start parallel instance |
| `x` | Stop session |
| `n` | Create new session instance |
| `e` | Rename session |
| `r` | Resume previous conversation or start new (supports Claude, Gemini, Codex, OpenCode, Amazon Q) |
| `p` | Send prompt/message to running session |
| `d` | Delete session |

#### Groups
| Key | Action |
|-----|--------|
| `g` | Create new group |
| `G` | Assign session to group |
| `→` | Expand group (when group selected) |
| `←` | Collapse group (when group selected) |
| `Tab` | Toggle group collapse (when group selected) |
| `e` | Rename group (when group selected) |
| `d` | Delete group (when group selected) |

#### Customization
| Key | Action |
|-----|--------|
| `c` | Change session color |
| `l` | Toggle compact mode |
| `t` | Toggle status lines (last output under sessions) |
| `y` | Toggle auto-yes mode (`--dangerously-skip-permissions`) |

#### Projects
| Key | Action |
|-----|--------|
| `P` | Return to project selector |
| `n` | Create new project (in project selector) |
| `e` | Rename project (in project selector) |
| `d` | Delete project (in project selector) |
| `i` | Import sessions from default to current project |

#### Other
| Key | Action |
|-----|--------|
| `R` | Force resize preview pane |
| `F1` / `?` | Show help |
| `q` | Quit |

### Inside Attached Session
| Key | Action |
|-----|--------|
| `Ctrl+q` | Detach from session (quick, works in any tmux session) |
| `Ctrl+b d` | Detach from session (tmux default) |

> **Note:** `Ctrl+q` is set as a universal quick-detach for all tmux sessions. ASM sessions get automatic resize before detach to maintain proper preview dimensions.

## Color Customization

Press `c` to open the color picker for the selected session:

- **Foreground Colors** - 22 solid colors + 15 gradients
- **Background Colors** - 22 solid colors
- **Auto Mode** - Automatically picks contrasting text color
- **Full Row Mode** - Extend background color to full row width (press `f` to toggle)
- **Gradients** - Rainbow, Sunset, Ocean, Forest, Fire, Ice, Neon, Galaxy, Pastel, and more!

Use `Tab` to switch between foreground and background color selection.

## Session Resume

ASM can resume previous Claude Code conversations:

1. Press `r` on any session
2. Browse through previous conversations (shows last message and timestamp)
3. Select a conversation to resume or start fresh

## Starting Sessions

Press `a` on any session to see start options:

- **Replace current session** (1/r): Stops the current session (if running) and starts a fresh new one
- **Start parallel session** (2/n): Prompts for a name (defaults to current session name), then creates a new instance with the same settings and starts it right below the current one in the list

This allows you to work on multiple tasks in the same project simultaneously, each with their own AI session.

## Session Groups

Organize your sessions into collapsible groups:

```
📁 Backend ▼ [3]
   ● api-server
   ● database-worker
   ○ cache-service
📁 Frontend ▶ [2]  (collapsed)
   ● misc-session
```

- Press `g` to create a new group
- Press `G` to assign the selected session to a group
- Press `→` to expand a group, `←` to collapse it
- Press `Tab` to toggle collapse/expand
- Press `e` on a group to rename it
- Press `c` on a group to change its color
- Press `d` on a group to delete it (sessions become ungrouped)

Sessions without a group appear at the bottom of the list.

## Projects

Projects allow you to organize your sessions into separate workspaces. Each project has its own isolated session list and groups.

### Project Selector

When you start ASMGR, you'll see the project selector:

```
┌─────────────────────────────────────┐
│  Agent Session Manager              │
│             v0.3.0                  │
├─────────────────────────────────────┤
│  > Backend API         [5 sessions] │
│    Frontend App        [3 sessions] │
│    DevOps Scripts      [2 sessions] │
│  ───────────────────────────────────│
│    [ ] Continue without project     │
│    [+] New Project                  │
└─────────────────────────────────────┘
```

- Select an existing project to work with its sessions
- Choose "Continue without project" for backward-compatible default sessions
- Create a new project to start fresh

### Single Instance Lock

Only one instance of ASMGR can run per project at a time. If you try to open a project that's already open in another terminal, you'll see an error with the PID of the running instance.

### Session Import

Press `i` in the project selector to import sessions from the default (no project) session list into the selected project. This is useful when migrating from the old single-session-list mode to the new project-based organization.

## Activity Indicators

Sessions show different status indicators:

- `●` Orange - Active (agent is working)
- `●` Gray - Idle (waiting for input)
- `○` Red outline - Stopped

## Configuration

Configuration files are stored in `~/.config/agent-session-manager/`:

```
~/.config/agent-session-manager/
├── projects.json              # Project list & metadata
├── sessions.json              # Default (no project) sessions
└── projects/
    ├── backend-api/
    │   └── sessions.json      # Project-specific sessions
    └── frontend-app/
        └── sessions.json
```

### projects.json
Stores the list of projects with their names and creation dates.

### sessions.json
Stores sessions and groups:
- Session: name, path, color settings, resume ID, auto-yes, group, agent type
- Group: name, collapsed state, color settings

### filters.json (optional)
Customize status line filtering for each agent. Default filters are built-in, but you can override them:

```json
{
  "claude": {
    "skip_contains": ["? for", "Context left"],
    "skip_prefixes": ["╭", "╰"],
    "min_separators": 20
  },
  "opencode": {
    "skip_contains": ["ctrl+?", "Context:"],
    "content_prefix": "┃",
    "show_contains": ["Generating"],
    "show_as": ["Generating..."]
  }
}
```

## Architecture

```
agent-session-manager/
├── main.go              # Entry point
├── session/             # Session management & tmux integration
│   ├── instance.go      # Instance lifecycle & PTY handling
│   ├── storage.go       # Persistence & project management
│   ├── project.go       # Project data structures
│   └── claude_sessions.go  # Claude session discovery
└── ui/                  # Bubbletea TUI
    ├── model.go         # Core model, constants, Init, Update
    ├── handlers.go      # Keyboard input handlers
    ├── views.go         # View rendering functions
    ├── colors.go        # Color definitions & gradients
    ├── styles.go        # Lipgloss style definitions
    └── helpers.go       # ANSI utilities & overlay dialog rendering
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [go-runewidth](https://github.com/mattn/go-runewidth) - Unicode character width calculation for overlay dialogs

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Inspired by [Claude Squad](https://github.com/smtg-ai/claude-squad)
- Built with [Charm](https://charm.sh/) libraries
- Powered by [Claude Code](https://github.com/anthropics/claude-code)
