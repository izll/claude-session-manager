#!/bin/bash
#
# ASMGR - Agent Session Manager Installer
# https://github.com/izll/agent-session-manager
#
# Quick install:
#   curl -fsSL https://raw.githubusercontent.com/izll/agent-session-manager/main/install.sh | bash
#

set -e

# Configuration
REPO="izll/agent-session-manager"
BINARY="asmgr"
INSTALL_PATH="$HOME/.local/bin"
VERSION=""
UPDATE_MODE=false

# Terminal colors
c_red="\033[31m"
c_green="\033[32m"
c_yellow="\033[33m"
c_blue="\033[34m"
c_magenta="\033[35m"
c_reset="\033[0m"

print_banner() {
    echo -e "${c_magenta}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║       ASMGR - Agent Session Manager       ║"
    echo "║               Installer                   ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${c_reset}"
}

info() { echo -e "${c_blue}[INFO]${c_reset} $1"; }
success() { echo -e "${c_green}[OK]${c_reset} $1"; }
warn() { echo -e "${c_yellow}[WARN]${c_reset} $1"; }
error() { echo -e "${c_red}[ERROR]${c_reset} $1"; exit 1; }

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -v|--version) VERSION="$2"; shift 2 ;;
            -d|--dir) INSTALL_PATH="$2"; shift 2 ;;
            -u|--update) UPDATE_MODE=true; shift ;;
            -h|--help) show_help; exit 0 ;;
            *) warn "Unknown option: $1"; shift ;;
        esac
    done
}

show_help() {
    echo "ASMGR Installer"
    echo ""
    echo "Usage: install.sh [options]"
    echo ""
    echo "Options:"
    echo "  -v, --version <ver>  Install specific version (default: latest)"
    echo "  -d, --dir <path>     Install directory (default: ~/.local/bin)"
    echo "  -u, --update         Update to latest version"
    echo "  -h, --help           Show this help"
}

# Get currently installed version
get_installed_version() {
    if [[ -x "$INSTALL_PATH/$BINARY" ]]; then
        "$INSTALL_PATH/$BINARY" --version 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1
    else
        echo ""
    fi
}

# Check for updates
check_update() {
    local current latest

    current=$(get_installed_version)
    if [[ -z "$current" ]]; then
        error "$BINARY is not installed. Run without --update to install."
    fi

    info "Current version: $current"
    info "Checking for updates..."

    latest=$(fetch_latest_version)

    # Normalize versions (remove 'v' prefix for comparison)
    local cur_norm="${current#v}"
    local lat_norm="${latest#v}"

    if [[ "$cur_norm" == "$lat_norm" ]]; then
        success "Already up to date ($current)"
        exit 0
    fi

    info "New version available: $latest"
    read -rp "Update now? [Y/n] " answer
    if [[ "$answer" =~ ^[Nn] ]]; then
        info "Update cancelled"
        exit 0
    fi

    VERSION="$latest"
}

# Detect operating system
detect_os() {
    local os_name
    os_name=$(uname -s | tr '[:upper:]' '[:lower:]')

    case "$os_name" in
        linux*)
            if grep -qiE "(microsoft|wsl)" /proc/version 2>/dev/null; then
                echo "linux-wsl"
            else
                echo "linux"
            fi
            ;;
        darwin*) echo "darwin" ;;
        *) error "Unsupported OS: $os_name" ;;
    esac
}

# Detect CPU architecture
detect_arch() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Get the latest release version from GitHub
fetch_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    version=$(curl -fsSL "$api_url" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version. Check your internet connection or specify version with -v"
    fi

    echo "$version"
}

# Check if tmux is installed
check_tmux() {
    if command -v tmux &>/dev/null; then
        success "tmux found: $(tmux -V 2>/dev/null || echo 'version unknown')"
        return 0
    fi

    warn "tmux is required but not installed"
    echo ""

    local pkg_manager=""
    local install_cmd=""

    if command -v apt-get &>/dev/null; then
        pkg_manager="apt"
        install_cmd="sudo apt-get update && sudo apt-get install -y tmux"
    elif command -v dnf &>/dev/null; then
        pkg_manager="dnf"
        install_cmd="sudo dnf install -y tmux"
    elif command -v pacman &>/dev/null; then
        pkg_manager="pacman"
        install_cmd="sudo pacman -S --noconfirm tmux"
    elif command -v brew &>/dev/null; then
        pkg_manager="brew"
        install_cmd="brew install tmux"
    elif command -v zypper &>/dev/null; then
        pkg_manager="zypper"
        install_cmd="sudo zypper install -y tmux"
    fi

    if [[ -n "$pkg_manager" ]]; then
        read -rp "Install tmux using $pkg_manager? [Y/n] " answer
        if [[ ! "$answer" =~ ^[Nn] ]]; then
            info "Installing tmux..."
            eval "$install_cmd"
            success "tmux installed"
        fi
    else
        echo "Please install tmux manually:"
        echo "  Debian/Ubuntu: sudo apt install tmux"
        echo "  Fedora:        sudo dnf install tmux"
        echo "  Arch:          sudo pacman -S tmux"
        echo "  macOS:         brew install tmux"
    fi

    if ! command -v tmux &>/dev/null; then
        read -rp "Continue without tmux? [y/N] " answer
        [[ "$answer" =~ ^[Yy] ]] || exit 1
    fi
}

# Configure tmux for better experience
configure_tmux() {
    local tmux_conf="$HOME/.tmux.conf"
    local marker="# asmgr-config"

    if [[ -f "$tmux_conf" ]] && grep -q "$marker" "$tmux_conf" 2>/dev/null; then
        info "tmux already configured for asmgr"
        return 0
    fi

    echo ""
    info "Recommended tmux settings:"
    echo "  - Mouse support for scrolling"
    echo "  - Clipboard integration"
    echo "  - Extended scrollback history"
    echo ""

    read -rp "Add recommended tmux configuration? [Y/n] " answer
    if [[ "$answer" =~ ^[Nn] ]]; then
        info "Skipping tmux configuration"
        return 0
    fi

    # Detect clipboard tool
    local clip_cmd="xclip -selection clipboard"
    local os_type
    os_type=$(detect_os)

    if [[ "$os_type" == "darwin" ]]; then
        clip_cmd="pbcopy"
    elif [[ "$os_type" == "linux-wsl" ]]; then
        clip_cmd="clip.exe"
    elif [[ -n "$WAYLAND_DISPLAY" ]] && command -v wl-copy &>/dev/null; then
        clip_cmd="wl-copy"
    elif command -v xsel &>/dev/null; then
        clip_cmd="xsel --clipboard --input"
    fi

    # Append configuration
    cat >> "$tmux_conf" << EOF

$marker
# Added by asmgr installer - $(date +%Y-%m-%d)
set -g mouse on
set -g history-limit 50000
set -g default-terminal "screen-256color"
set -ga terminal-overrides ",*256col*:Tc"
set -sg escape-time 10
bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "$clip_cmd"
bind-key -T copy-mode MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel "$clip_cmd"
# end asmgr-config
EOF

    success "tmux configuration added"

    # Reload if tmux is running
    if tmux list-sessions &>/dev/null; then
        tmux source-file "$tmux_conf" 2>/dev/null && info "tmux config reloaded"
    fi
}

# Download and install the binary
install_binary() {
    local os_type arch version download_url tmp_dir

    os_type=$(detect_os)
    [[ "$os_type" == "linux-wsl" ]] && os_type="linux"
    arch=$(detect_arch)

    if [[ -z "$VERSION" ]]; then
        info "Fetching latest version..."
        VERSION=$(fetch_latest_version)
    fi

    success "Installing version: $VERSION"

    # Strip 'v' prefix for filename
    local ver_num="${VERSION#v}"
    download_url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}_${ver_num}_${os_type}_${arch}.tar.gz"

    info "Downloading from: $download_url"

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    if ! curl -fsSL "$download_url" -o "$tmp_dir/archive.tar.gz"; then
        error "Download failed. The release may not exist yet."
    fi

    info "Extracting..."
    tar -xzf "$tmp_dir/archive.tar.gz" -C "$tmp_dir"

    mkdir -p "$INSTALL_PATH"
    mv "$tmp_dir/$BINARY" "$INSTALL_PATH/$BINARY"
    chmod +x "$INSTALL_PATH/$BINARY"

    success "Installed to: $INSTALL_PATH/$BINARY"
}

# Check if install path is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_PATH:"* ]]; then
        warn "$INSTALL_PATH is not in your PATH"
        echo ""
        echo "Add to your shell config:"

        local shell_rc=""
        if [[ -n "$ZSH_VERSION" ]] || [[ -f "$HOME/.zshrc" ]]; then
            shell_rc="$HOME/.zshrc"
        elif [[ -f "$HOME/.bashrc" ]]; then
            shell_rc="$HOME/.bashrc"
        fi

        if [[ -n "$shell_rc" ]]; then
            echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> $shell_rc"
            echo "  source $shell_rc"
        else
            echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        fi
        echo ""
    fi
}

# Print final summary
print_summary() {
    local os_type
    os_type=$(detect_os)

    echo ""
    echo -e "${c_green}╔═══════════════════════════════════════════╗${c_reset}"
    echo -e "${c_green}║          Installation Complete!           ║${c_reset}"
    echo -e "${c_green}╚═══════════════════════════════════════════╝${c_reset}"
    echo ""
    echo -e "Binary:   ${c_green}$INSTALL_PATH/$BINARY${c_reset}"
    echo -e "Version:  ${c_green}$VERSION${c_reset}"
    echo -e "Platform: ${c_green}$(detect_os)/$(detect_arch)${c_reset}"
    echo ""
    echo "Quick start:"
    echo "  $BINARY              # Launch the TUI"
    echo "  $BINARY --help       # Show help"
    echo ""

    if [[ "$os_type" == "linux-wsl" ]]; then
        info "WSL detected - clipboard integration uses Windows clipboard"
    fi
}

# Main installation flow
main() {
    parse_args "$@"
    print_banner

    info "Detected: $(detect_os)/$(detect_arch)"
    echo ""

    if [[ "$UPDATE_MODE" == "true" ]]; then
        check_update
        install_binary
        print_summary
    else
        check_tmux
        install_binary
        configure_tmux
        check_path
        print_summary
    fi
}

main "$@"
