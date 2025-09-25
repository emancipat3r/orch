#!/bin/bash

set -e

# Configuration
NAMESPACE_NAME="${1:-vps}"
CLIENT_CONFIG_PATH="${2}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(dirname "$SCRIPT_DIR")/wireguard-clients"

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

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        log_info "Please run: sudo $0 $*"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    for cmd in ip wg wg-quick; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Please install WireGuard tools:"
        log_info "  Ubuntu/Debian: apt install wireguard wireguard-tools"
        log_info "  Fedora/RHEL: dnf install wireguard-tools"
        log_info "  Arch: pacman -S wireguard-tools"
        exit 1
    fi
}

# Find client configuration
find_client_config() {
    if [[ -n "$CLIENT_CONFIG_PATH" ]]; then
        if [[ ! -f "$CLIENT_CONFIG_PATH" ]]; then
            log_error "Specified client config file not found: $CLIENT_CONFIG_PATH"
            exit 1
        fi
        echo "$CLIENT_CONFIG_PATH"
        return
    fi

    # Look for client configs in the expected directory
    if [[ -d "$CLIENT_DIR" ]]; then
        local configs=($(find "$CLIENT_DIR" -name "client-*.conf" 2>/dev/null))

        if [[ ${#configs[@]} -eq 0 ]]; then
            log_error "No client configurations found in $CLIENT_DIR"
            log_info "Make sure you've run the VPS provisioning process first"
            exit 1
        elif [[ ${#configs[@]} -eq 1 ]]; then
            echo "${configs[0]}"
            return
        else
            log_info "Multiple client configurations found:"
            for i in "${!configs[@]}"; do
                echo "  $((i+1)). ${configs[i]}"
            done

            read -p "Select configuration (1-${#configs[@]}): " choice
            if [[ "$choice" =~ ^[0-9]+$ ]] && [[ "$choice" -ge 1 ]] && [[ "$choice" -le ${#configs[@]} ]]; then
                echo "${configs[$((choice-1))]}"
                return
            else
                log_error "Invalid selection"
                exit 1
            fi
        fi
    else
        log_error "Client configuration directory not found: $CLIENT_DIR"
        log_info "Make sure you've run the VPS provisioning process first"
        exit 1
    fi
}

# Create network namespace
create_namespace() {
    local ns_name="$1"

    if ip netns list | grep -q "^$ns_name\b"; then
        log_warn "Network namespace '$ns_name' already exists"

        # Check if WireGuard is already running in the namespace
        if ip netns exec "$ns_name" wg show 2>/dev/null | grep -q "interface:"; then
            log_warn "WireGuard is already running in namespace '$ns_name'"
            read -p "Do you want to recreate it? (y/N): " recreate
            if [[ "$recreate" =~ ^[Yy]$ ]]; then
                cleanup_namespace "$ns_name"
            else
                log_info "Using existing namespace"
                return 0
            fi
        fi
    fi

    log_info "Creating network namespace: $ns_name"
    ip netns add "$ns_name"

    # Bring up loopback in namespace
    ip netns exec "$ns_name" ip link set lo up

    log_success "Network namespace '$ns_name' created"
}

# Setup WireGuard in namespace
setup_wireguard() {
    local ns_name="$1"
    local config_path="$2"

    log_info "Setting up WireGuard in namespace: $ns_name"

    # Copy config to a temporary location accessible in the namespace
    local temp_config="/tmp/wg-${ns_name}.conf"
    cp "$config_path" "$temp_config"

    # Start WireGuard in the namespace
    if ip netns exec "$ns_name" wg-quick up "$temp_config"; then
        log_success "WireGuard started successfully in namespace: $ns_name"

        # Show WireGuard status
        log_info "WireGuard status:"
        ip netns exec "$ns_name" wg show

        # Show IP configuration
        log_info "Network configuration in namespace:"
        ip netns exec "$ns_name" ip addr show

        # Test connectivity
        log_info "Testing connectivity..."
        if ip netns exec "$ns_name" ping -c 1 -W 5 8.8.8.8 > /dev/null 2>&1; then
            log_success "Internet connectivity test passed"
        else
            log_warn "Internet connectivity test failed"
        fi
    else
        log_error "Failed to start WireGuard"
        cleanup_namespace "$ns_name"
        exit 1
    fi

    # Clean up temporary config
    rm -f "$temp_config"
}

# Cleanup namespace
cleanup_namespace() {
    local ns_name="$1"

    log_info "Cleaning up namespace: $ns_name"

    # Try to bring down WireGuard first
    local temp_config="/tmp/wg-${ns_name}.conf"
    if [[ -f "$temp_config" ]]; then
        ip netns exec "$ns_name" wg-quick down "$temp_config" 2>/dev/null || true
        rm -f "$temp_config"
    fi

    # Delete the namespace
    if ip netns list | grep -q "^$ns_name\b"; then
        ip netns delete "$ns_name"
        log_info "Namespace '$ns_name' deleted"
    fi
}

# Show usage information
show_usage() {
    local ns_name="$1"

    log_success "Network namespace setup complete!"
    echo
    log_info "Usage examples:"
    echo "  # Run a command in the namespace:"
    echo "    sudo ip netns exec $ns_name <command>"
    echo
    echo "  # Start a shell in the namespace:"
    echo "    sudo ip netns exec $ns_name bash"
    echo
    echo "  # Test connectivity:"
    echo "    sudo ip netns exec $ns_name curl ipinfo.io"
    echo
    echo "  # Run a browser (if you have X11 forwarding):"
    echo "    sudo ip netns exec $ns_name -u \$(id -u) -g \$(id -g) firefox"
    echo
    log_info "To remove the namespace:"
    echo "    sudo ip netns delete $ns_name"
}

# Signal handlers for cleanup
cleanup_on_exit() {
    if [[ -n "${TEMP_CONFIG:-}" ]] && [[ -f "$TEMP_CONFIG" ]]; then
        rm -f "$TEMP_CONFIG"
    fi
}

trap cleanup_on_exit EXIT

# Main function
main() {
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        echo "Usage: $0 [NAMESPACE_NAME] [CLIENT_CONFIG_PATH]"
        echo
        echo "Arguments:"
        echo "  NAMESPACE_NAME      Name of the network namespace (default: vps)"
        echo "  CLIENT_CONFIG_PATH  Path to WireGuard client config (auto-detect if not provided)"
        echo
        echo "This script creates a network namespace and sets up WireGuard client connection."
        echo "All traffic in the namespace will be routed through the VPN."
        exit 0
    fi

    check_root
    check_dependencies

    local config_path
    config_path=$(find_client_config)

    log_info "Using client configuration: $config_path"
    log_info "Setting up namespace: $NAMESPACE_NAME"

    create_namespace "$NAMESPACE_NAME"
    setup_wireguard "$NAMESPACE_NAME" "$config_path"
    show_usage "$NAMESPACE_NAME"
}

# Run main function
main "$@"
