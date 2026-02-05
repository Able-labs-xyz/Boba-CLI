#!/bin/bash
#
# Boba CLI Installer
# Installs the Boba CLI and optionally configures Claude Desktop/Code
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/boba-labs/boba-cli/main/install.sh | bash
#
# Or with options:
#   curl -fsSL ... | bash -s -- --skip-claude
#

set -e

# Colors
PURPLE='\033[0;35m'
BRIGHT_PURPLE='\033[1;35m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Spinner frames
SPINNER_FRAMES=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")

# Print functions
print_purple() {
  printf "${PURPLE}%s${NC}\n" "$1"
}

print_bright() {
  printf "${BRIGHT_PURPLE}%s${NC}\n" "$1"
}

print_success() {
  printf "${GREEN}✓${NC} %s\n" "$1"
}

print_error() {
  printf "${RED}✗${NC} %s\n" "$1"
}

print_warning() {
  printf "${YELLOW}!${NC} %s\n" "$1"
}

# Spinner function
spin() {
  local pid=$1
  local message=$2
  local i=0

  while kill -0 "$pid" 2>/dev/null; do
    printf "\r${PURPLE}${SPINNER_FRAMES[$i]}${NC} %s" "$message"
    i=$(( (i + 1) % ${#SPINNER_FRAMES[@]} ))
    sleep 0.1
  done
  printf "\r"
}

# Banner
print_banner() {
  echo ""
  print_bright "  ██████╗  ██████╗ ██████╗  █████╗ "
  print_bright "  ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗"
  print_bright "  ██████╔╝██║   ██║██████╔╝███████║"
  print_bright "  ██╔══██╗██║   ██║██╔══██╗██╔══██║"
  print_bright "  ██████╔╝╚██████╔╝██████╔╝██║  ██║"
  print_bright "  ╚═════╝  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝"
  echo ""
  print_purple "  AI Trading Made Simple"
  echo ""
}

# Check for required tools
check_requirements() {
  local missing=0

  if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed"
    echo "  Install from: https://nodejs.org/ (v18+ required)"
    missing=1
  else
    local node_version=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$node_version" -lt 18 ]; then
      print_error "Node.js v18+ required (found v$node_version)"
      missing=1
    else
      print_success "Node.js $(node -v)"
    fi
  fi

  if ! command -v npm &> /dev/null; then
    print_error "npm is not installed"
    missing=1
  else
    print_success "npm $(npm -v)"
  fi

  if [ $missing -eq 1 ]; then
    echo ""
    print_error "Please install the missing requirements and try again."
    exit 1
  fi
}

# Install boba CLI
install_boba() {
  echo ""
  print_purple "Installing Boba CLI..."
  echo ""

  # Install globally
  npm install -g @boba/cli 2>&1 &
  local pid=$!
  spin $pid "Installing @boba/cli globally..."
  wait $pid
  local exit_code=$?

  if [ $exit_code -eq 0 ]; then
    print_success "Boba CLI installed successfully"
  else
    print_error "Failed to install Boba CLI"
    echo ""
    echo "Try installing manually:"
    echo "  npm install -g @boba/cli"
    exit 1
  fi

  # Verify installation
  if command -v boba &> /dev/null; then
    print_success "boba command available at: $(which boba)"
  else
    print_warning "boba command not in PATH"
    echo "  You may need to add npm global bin to your PATH"
    echo "  Run: npm bin -g"
  fi
}

# Configure Claude (optional)
configure_claude() {
  echo ""
  print_purple "Configuring Claude..."
  echo ""

  if command -v boba &> /dev/null; then
    boba install --desktop --code
  else
    npx -y @boba/cli install --desktop --code
  fi
}

# Main
main() {
  local skip_claude=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      --skip-claude)
        skip_claude=true
        shift
        ;;
      --help|-h)
        echo "Boba CLI Installer"
        echo ""
        echo "Usage: install.sh [options]"
        echo ""
        echo "Options:"
        echo "  --skip-claude  Skip Claude Desktop/Code configuration"
        echo "  --help, -h     Show this help message"
        exit 0
        ;;
      *)
        print_warning "Unknown option: $1"
        shift
        ;;
    esac
  done

  print_banner

  echo "Checking requirements..."
  echo ""
  check_requirements

  install_boba

  if [ "$skip_claude" = false ]; then
    echo ""
    read -p "Configure Claude Desktop & Claude Code? [Y/n] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
      configure_claude
    else
      print_purple "Skipping Claude configuration"
      echo "  Run 'boba install' later to configure Claude"
    fi
  fi

  echo ""
  print_bright "Installation complete!"
  echo ""
  print_purple "Next steps:"
  echo "  1. Run ${BOLD}boba init${NC} to set up your agent credentials"
  echo "  2. Run ${BOLD}boba proxy${NC} to start the MCP proxy"
  echo "  3. Open Claude and start trading!"
  echo ""
  print_purple "For help: boba --help"
  echo ""
}

main "$@"
