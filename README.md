# vps3
## Description
This project is a command-line tool for managing virtual private servers (VPS) using provider APIs. It provides a simple interface for creating, listing, and deleting VPS instances with automatic WireGuard VPN setup and network namespace integration.

## Features

- **Multi-Provider Support**: Create VPS instances on Linode, DigitalOcean, and Vultr
- **Automatic WireGuard Setup**: Automatically installs and configures WireGuard VPN on Ubuntu/Debian VPS instances
- **Network Namespace Integration**: Create isolated network namespaces to route traffic through your VPS
- **SSH Key Management**: Per-instance SSH key generation and management
- **Pure Go Implementation**: Automated post-provisioning setup with no external dependencies
- **Security Hardening**: Automatic SSH and firewall configuration

## Quick Start

1. **Install Prerequisites**:
   ```bash
   # For network namespace support (optional)
   sudo apt install wireguard wireguard-tools  # Ubuntu/Debian
   # or
   brew install wireguard-tools  # macOS
   ```

2. **Create a VPS**:
   ```bash
   ./vps3 create
   ```

3. **Use Network Namespace** (after VPS creation):
   ```bash
   # Set up network namespace (routes all traffic through VPS)
   sudo ./vps3 namespace setup
   
   # Run commands through VPS connection
   sudo ./vps3 namespace exec vps curl ipinfo.io
   ```

## What Happens During VPS Creation

When you create a VPS with `vps3 create`, the system automatically:

1. **Provisions the VPS** using your selected provider
2. **Waits for SSH** to become available using native Go SSH client
3. **Checks OS Compatibility** (Ubuntu/Debian required for auto-setup)
4. **Runs Pure Go Post-Provisioning** to:
   - Update and secure the system
   - Install WireGuard VPN server
   - Configure firewall (UFW)
   - Harden SSH configuration
   - Download client configuration files
   - Configure firewall and SSH settings
   - Generate VPN keys and configurations
5. **Downloads Client Config** to `./wireguard-clients/`
6. **Provides Setup Instructions** for network namespace

## Network Namespace Commands

```bash
# Set up a network namespace with WireGuard client
sudo vps3 namespace setup [namespace-name]

# Execute commands in the namespace (traffic goes through VPS)
sudo vps3 namespace exec [namespace-name] <command>

# Check namespace status and connectivity
sudo vps3 namespace status [namespace-name]

# List all namespaces
vps3 namespace list

# Delete a namespace
sudo vps3 namespace delete [namespace-name]
```

## Additional Commands

### Post-Provisioning Setup
If the initial setup fails or you want to reconfigure an existing instance:

```bash
# Run post-provisioning setup on existing VPS
vps3 setup
```

### Connectivity Testing
Test SSH connectivity and get system information:

```bash
# Test SSH and get system info for existing VPS
vps3 test
```

### Instance Management
```bash
# List all VPS instances
vps3 list

# Destroy VPS instances
vps3 destroy
```

## Troubleshooting

### SSH Connection Issues
If SSH connectivity fails during setup:

1. **Check Instance Status**: Ensure the VPS is fully booted (check provider console)
2. **Verify Key Permissions**: SSH key file should have 600 permissions
3. **Test Connectivity**: Use `vps3 test` to diagnose connection issues
4. **Retry Setup**: Use `vps3 setup` to retry post-provisioning

### Common Issues
- **Timeout during SSH wait**: Instance may still be booting - wait a few minutes and retry
- **Permission denied**: Check SSH key file exists and has correct permissions
- **Port 22 closed**: Some providers may take time to configure security groups

## Pure Go Implementation

This tool uses a pure Go implementation for all post-provisioning tasks, eliminating the need for external dependencies like Ansible. The Go-based system:

- Uses native SSH client for secure connections
- Handles all system configuration remotely
- Provides better error handling and retry logic
- Works without any external tools or dependencies
- Offers improved performance and reliability
