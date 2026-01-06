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
- **Agent Icons** - Toggle display of agent type icons (🤖💎🔧📦🦜💻⚙️) in session list
- **Custom Colors** - Personalize sessions with foreground colors, background colors, and gradients
- **Prompt Sending** - Send messages to running sessions without attaching (improved reliability for all agents)
- **Session Reordering** - Organize sessions with keyboard shortcuts
- **Compact Mode** - Toggle spacing between sessions for denser view
- **Smart Resize** - Terminal resize follows when attached, preview size preserved when detached
- **Overlay Dialogs** - Modal dialogs rendered over the main view with proper Unicode character width handling
- **Fancy Status Bar** - Styled bottom bar with highlighted keys, toggle indicators, and separators
- **Scrollable Help View** - Comprehensive help page with keyboard shortcuts, detailed descriptions, and scroll support
- **Session Groups** - Organize sessions into collapsible groups for better organization
- **Session Notes** - Add persistent notes/comments to sessions (stays with session, not conversation)
- **Split View** - Compare two sessions side-by-side with pinned preview

## Installation

### Prerequisites

- tmux
- At least one AI CLI tool installed:
  - [Claude Code](https://github.com/anthropics/claude-code)
  - [Gemini CLI](https://github.com/google-gemini/gemini-cli)
  - [Aider](https://github.com/paul-gauthier/aider)
  - [OpenAI Codex](https://github.com/openai/codex)
  - [Amazon Q](https://aws.amazon.com/q/)
  - [OpenCode](https://github.com/opencode-ai/opencode)

### Homebrew (macOS/Linux)

```bash
brew tap izll/tap
brew install asmgr
```

Update:
```bash
brew upgrade asmgr
```

### Quick Install Script (Linux)

Download and install the latest release automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/izll/agent-session-manager/main/install.sh | bash
```

Or download and run locally:

```bash
curl -fsSL https://raw.githubusercontent.com/izll/agent-session-manager/main/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

Install options:
```bash
./install.sh              # Install latest version to ~/.local/bin
./install.sh -v 0.5.2     # Install specific version
./install.sh -d /usr/local/bin  # Install to custom directory
./install.sh -u           # Update existing installation
```

### Package Managers

**Debian/Ubuntu (.deb):**
```bash
# Download from releases
wget https://github.com/izll/agent-session-manager/releases/download/v0.5.2/asmgr_0.5.2_linux_amd64.deb
sudo dpkg -i asmgr_0.5.2_linux_amd64.deb
```

**RedHat/Fedora/Rocky (.rpm):**
```bash
# Download from releases
wget https://github.com/izll/agent-session-manager/releases/download/v0.5.2/asmgr_0.5.2_linux_x86_64.rpm
sudo rpm -i asmgr_0.5.2_linux_x86_64.rpm
```

### Build from Source

If you prefer to build from source (requires Go 1.24+):

```bash
git clone https://github.com/izll/agent-session-manager.git
cd agent-session-manager
go build -o asmgr .
cp asmgr ~/.local/bin/
```

## Updating

### Built-in Self-Update

ASMGR includes a built-in self-update feature. Simply press `U` (Shift+U) while running the application:

1. A gold `↑` arrow appears in the top-right corner when an update is available
2. Press `U` to check for updates
3. Confirm the update with `Y`
4. The update is downloaded and installed automatically
5. Restart ASMGR to use the new version

**Supported installation methods:**
- ✅ **Homebrew** - Updates via `brew upgrade asmgr`
- ✅ **Debian/Ubuntu (.deb)** - Interactive `sudo dpkg -i` update
- ✅ **RedHat/Fedora/Rocky (.rpm)** - Interactive `sudo rpm -Uvh` update
- ✅ **Install script (tar.gz)** - Self-update with automatic binary replacement
- ✅ **Manual install (tar.gz)** - Self-update if installed to user directory

### Manual Update

**Homebrew:**
```bash
brew upgrade asmgr
```

**Package managers:**
```bash
# Debian/Ubuntu
sudo apt update && sudo apt upgrade asmgr

# RedHat/Fedora
sudo dnf upgrade asmgr
```

**Install script:**
```bash
curl -fsSL https://raw.githubusercontent.com/izll/agent-session-manager/main/install.sh | bash -s -- -u
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
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `Ctrl+↓` | Move session down (reorder) |
| `Ctrl+↑` | Move session up (reorder) |
| `J` / `Shift+↓` / `PgDn` | Scroll preview down |
| `K` / `Shift+↑` / `PgUp` | Scroll preview up |
| `Home` | Scroll preview to top |
| `End` | Scroll preview to bottom |

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
| `N` | Add/edit session notes |
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
| `I` | Toggle agent icons in session list (🤖💎🔧📦🦜💻⚙️) |
| `Ctrl+y` | Toggle auto-yes/yolo mode (restarts session if running) |

#### Split View
| Key | Action |
|-----|--------|
| `v` | Toggle split view |
| `m` | Mark/pin session for top pane |
| `Tab` | Switch focus between split panes |

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
| `U` | Check for updates and install (built-in self-update) |
| `R` | Force resize preview pane |
| `F1` / `?` | Show help |
| `q` | Quit |

### Inside Attached Session
| Key | Action |
|-----|--------|
| `Ctrl+q` | Detach from session (quick, works in any tmux session) |
| `Ctrl+b d` | Detach from session (tmux default) |

> **Note:** `Ctrl+q` is set as a universal quick-detach for all tmux sessions. ASMGR sessions get automatic resize before detach to maintain proper preview dimensions.

## Color Customization

Press `c` to open the color picker for the selected session:

- **Foreground Colors** - 22 solid colors + 15 gradients
- **Background Colors** - 22 solid colors
- **Auto Mode** - Automatically picks contrasting text color
- **Full Row Mode** - Extend background color to full row width (press `f` to toggle)
- **Gradients** - Rainbow, Sunset, Ocean, Forest, Fire, Ice, Neon, Galaxy, Pastel, and more!

Use `Tab` to switch between foreground and background color selection.

## Session Resume

Resume previous conversations for supported agents (Claude, Gemini, Codex, OpenCode, Amazon Q):

1. Press `r` on any session
2. Browse through previous conversations (shows last message and timestamp)
3. Select a conversation to resume or start fresh

Note: Aider and custom commands don't support session resume.

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

## Session Notes

Add persistent notes to any session that stay with the session even when you change the resume conversation:

- Press `N` (Shift+N) on any session to open the notes editor
- Write multi-line notes (Enter for new lines)
- `Ctrl+S` to save, `Esc` to cancel, `Ctrl+D` to clear
- Notes are shown in the preview pane below the session info
- Notes persist across session restarts and conversation changes

Use notes to track:
- Current task/goal for each session
- Important context or decisions
- TODOs and reminders
- Handoff notes when switching between sessions

## Split View

Compare two sessions side-by-side:

- Press `v` to toggle split view
- Press `m` to mark/pin the current session (shown in top pane)
- Press `Tab` to switch focus between panes
- Navigate with arrow keys to change the selected session (bottom pane)

The pinned session stays visible while you browse other sessions, useful for comparing outputs or referencing one session while working in another.

## Projects

Projects allow you to organize your sessions into separate workspaces. Each project has its own isolated session list and groups.

### Project Selector

When you start ASMGR, you'll see the project selector:

```
┌─────────────────────────────────────┐
│  Agent Session Manager              │
│             v0.5.0                  │
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
- `●` Blue - Waiting (waiting for user permission/input)
- `●` Gray - Idle (ready for new prompt)
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
- Session: name, path, color settings, resume ID, auto-yes, group, agent type, notes
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
├── main.go                  # Entry point
├── session/                 # Session management & tmux integration
│   ├── instance.go          # Instance lifecycle & PTY handling
│   ├── storage.go           # Persistence & project management
│   ├── project.go           # Project data structures
│   ├── status_detector.go   # Activity detection (idle/busy/waiting)
│   ├── suggestion.go        # Prompt suggestions from agents
│   ├── agent_session.go     # Agent session interface
│   ├── claude_sessions.go   # Claude session discovery
│   ├── gemini_sessions.go   # Gemini session discovery
│   ├── codex_sessions.go    # Codex session discovery
│   ├── opencode_sessions.go # OpenCode session discovery
│   ├── amazonq_sessions.go  # Amazon Q session discovery
│   └── filters/             # Status line filters per agent
│       ├── config.go        # Filter configuration & loading
│       ├── claude.go        # Claude-specific filters
│       ├── gemini.go        # Gemini-specific filters
│       └── ...              # Other agent filters
├── ui/                      # Bubbletea TUI
│   ├── model.go             # Core model, constants, Init, Update
│   ├── views.go             # Main View() dispatcher
│   ├── views_session_list.go # Session list rendering
│   ├── views_preview.go     # Preview pane & split view
│   ├── views_dialogs.go     # Overlay dialogs (confirm, rename, notes, etc.)
│   ├── views_project.go     # Project selector views
│   ├── views_status.go      # Status bar & session selector
│   ├── views_help.go        # Help screen
│   ├── views_color_picker.go # Color picker view
│   ├── handlers.go          # Handler dispatcher
│   ├── handlers_list.go     # Main list keyboard handlers
│   ├── handlers_dialogs.go  # Dialog keyboard handlers
│   ├── handlers_session.go  # Session action handlers
│   ├── handlers_project.go  # Project management handlers
│   ├── handlers_group.go    # Group management handlers
│   ├── colors.go            # Color definitions & gradients
│   ├── styles.go            # Lipgloss style definitions
│   └── helpers.go           # ANSI utilities & overlay rendering
└── updater/                 # Self-update functionality
    └── updater.go           # Update checker & installer
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
