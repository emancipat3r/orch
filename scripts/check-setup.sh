#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

echo "======================================"
echo "         VPS3 Setup Verification"
echo "======================================"
echo

# Check if vps3 binary exists and is executable
log_info "Checking vps3 binary..."
if [[ -x "./vps3" ]]; then
    log_success "vps3 binary found and executable"
    BINARY_PATH="./vps3"
elif [[ -x "vps3" ]]; then
    log_success "vps3 binary found and executable"
    BINARY_PATH="vps3"
elif command -v vps3 &> /dev/null; then
    log_success "vps3 found in PATH"
    BINARY_PATH="vps3"
else
    log_error "vps3 binary not found. Run 'go build .' to build it."
    exit 1
fi

# Check vps3 commands
log_info "Testing vps3 commands..."
if $BINARY_PATH --help &> /dev/null; then
    log_success "vps3 help command works"
else
    log_error "vps3 help command failed"
    exit 1
fi

# Check if config directory structure exists
log_info "Checking configuration directory..."
CONFIG_DIR="$HOME/.config/vps3"
if [[ -d "$CONFIG_DIR" ]]; then
    log_success "Config directory exists: $CONFIG_DIR"
else
    log_warn "Config directory not found: $CONFIG_DIR"
    log_info "Creating config directory..."
    mkdir -p "$CONFIG_DIR"
    log_success "Config directory created"
fi

# Check if config file exists
CONFIG_FILE="$CONFIG_DIR/config.toml"
if [[ -f "$CONFIG_FILE" ]]; then
    log_success "Config file exists: $CONFIG_FILE"
else
    log_warn "Config file not found: $CONFIG_FILE"
    log_info "You can copy config.example.toml to $CONFIG_FILE and add your API keys"
fi

# Check SSH directory
SSH_DIR="$CONFIG_DIR/.ssh"
if [[ -d "$SSH_DIR" ]]; then
    log_success "SSH directory exists: $SSH_DIR"
else
    log_info "SSH directory will be created automatically when needed"
fi

# Check Ansible installation
log_info "Checking Ansible installation..."
if command -v ansible &> /dev/null; then
    ANSIBLE_VERSION=$(ansible --version | head -1)
    log_success "Ansible found: $ANSIBLE_VERSION"

    # Check ansible-playbook
    if command -v ansible-playbook &> /dev/null; then
        log_success "ansible-playbook command available"
    else
        log_warn "ansible-playbook command not found"
    fi
else
    log_warn "Ansible not found. Install with: pip install ansible"
    log_info "Without Ansible, automatic VPS setup will be skipped"
fi

# Check WireGuard tools
log_info "Checking WireGuard tools..."
if command -v wg &> /dev/null; then
    WG_VERSION=$(wg --version 2>&1 | head -1 || echo "Unknown version")
    log_success "WireGuard tools found: $WG_VERSION"

    if command -v wg-quick &> /dev/null; then
        log_success "wg-quick command available"
    else
        log_warn "wg-quick command not found"
    fi
else
    log_warn "WireGuard tools not found"
    log_info "Install with:"
    log_info "  Ubuntu/Debian: sudo apt install wireguard wireguard-tools"
    log_info "  macOS: brew install wireguard-tools"
    log_info "Without WireGuard tools, network namespace setup won't work"
fi

# Check network namespace support (Linux only)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    log_info "Checking network namespace support..."
    if command -v ip &> /dev/null; then
        log_success "iproute2 tools (ip command) found"

        # Check if we can list namespaces (requires some privileges)
        if ip netns list &> /dev/null; then
            log_success "Network namespace support available"
        else
            log_info "Network namespace operations may require root privileges"
        fi
    else
        log_warn "iproute2 tools not found. Network namespaces won't work."
    fi
else
    log_info "Network namespace support is Linux-specific (detected: $OSTYPE)"
fi

# Check required directories and files
log_info "Checking project structure..."

REQUIRED_DIRS=("ansible" "ansible/templates" "scripts" "cmd" "providers" "utils")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
        log_success "Directory exists: $dir"
    else
        log_error "Required directory missing: $dir"
    fi
done

REQUIRED_FILES=(
    "ansible/playbook.yml"
    "ansible/templates/wg0.conf.j2"
    "ansible/templates/client.conf.j2"
    "ansible/inventory.j2"
    "scripts/setup-namespace.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        log_success "File exists: $file"
    else
        log_error "Required file missing: $file"
    fi
done

# Check if setup-namespace.sh is executable
if [[ -x "scripts/setup-namespace.sh" ]]; then
    log_success "setup-namespace.sh is executable"
else
    log_warn "setup-namespace.sh is not executable"
    log_info "Run: chmod +x scripts/setup-namespace.sh"
fi

# Summary
echo
echo "======================================"
echo "           Setup Summary"
echo "======================================"

# Count issues
WARNINGS=0
ERRORS=0

# Basic functionality
if [[ -x "$BINARY_PATH" ]]; then
    log_success "✓ vps3 binary ready"
else
    log_error "✗ vps3 binary not ready"
    ((ERRORS++))
fi

# Configuration
if [[ -f "$CONFIG_FILE" ]]; then
    log_success "✓ Configuration file exists"
else
    log_warn "! Configuration file needs setup"
    WARNINGS=$((WARNINGS + 1))
fi

# Ansible
if command -v ansible &> /dev/null && command -v ansible-playbook &> /dev/null; then
    log_success "✓ Ansible ready for automatic setup"
else
    log_warn "! Ansible not available (manual setup only)"
    WARNINGS=$((WARNINGS + 1))
fi

# WireGuard
if command -v wg &> /dev/null && command -v wg-quick &> /dev/null; then
    log_success "✓ WireGuard tools ready"
else
    log_warn "! WireGuard tools not available (no network namespace support)"
    WARNINGS=$((WARNINGS + 1))
fi

echo
if [[ $ERRORS -eq 0 ]]; then
    if [[ $WARNINGS -eq 0 ]]; then
        log_success "🎉 Setup verification completed successfully!"
        log_info "You can now run: $BINARY_PATH create"
    else
        log_info "✓ Basic setup is ready with $WARNINGS optional warnings"
        log_info "You can run: $BINARY_PATH create"
        echo
        log_info "For full functionality, consider addressing the warnings above."
    fi
else
    log_error "❌ Setup verification failed with $ERRORS errors and $WARNINGS warnings"
    log_info "Please fix the errors above before using vps3"
    exit 1
fi

echo
log_info "For help getting started:"
log_info "  • Copy config.example.toml to $CONFIG_FILE and add your API keys"
log_info "  • Run '$BINARY_PATH create' to provision a VPS"
log_info "  • Run '$BINARY_PATH namespace --help' for network namespace commands"
log_info "  • See ansible/README.md for detailed setup instructions"
