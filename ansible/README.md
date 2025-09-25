# VPS3 Ansible Integration

This directory contains Ansible playbooks and configurations for automatically setting up VPS instances with WireGuard VPN after provisioning.

## Overview

When you create a VPS using `vps3 create`, the system will automatically:

1. **Check OS Compatibility**: Verify the VPS is running Ubuntu or Debian
2. **Run Ansible Playbook**: Execute the setup playbook if Ansible is installed
3. **Install WireGuard**: Set up WireGuard VPN server on the VPS
4. **Download Client Config**: Retrieve the client configuration for local use
5. **Setup Network Namespace**: Optionally create a local network namespace for routing traffic through the VPS

## Prerequisites

### On Your Local Machine

1. **Ansible** (optional, but recommended for automatic setup):
   ```bash
   # Ubuntu/Debian
   sudo apt update && sudo apt install ansible
   
   # macOS
   brew install ansible
   
   # Or via pip
   pip install ansible
   ```

2. **WireGuard Tools** (for network namespace setup):
   ```bash
   # Ubuntu/Debian
   sudo apt install wireguard wireguard-tools
   
   # macOS
   brew install wireguard-tools
   ```

### On the VPS

The playbook will automatically install all required packages on Ubuntu/Debian systems.

## Files Structure

```
ansible/
├── README.md                 # This file
├── playbook.yml             # Main Ansible playbook
├── inventory.j2             # Inventory template
├── templates/
│   ├── wg0.conf.j2         # WireGuard server configuration template
│   └── client.conf.j2      # WireGuard client configuration template
```

## What Gets Installed

The Ansible playbook performs the following tasks:

### System Updates
- Updates package cache
- Upgrades all system packages
- Installs essential tools (curl, wget, git, etc.)

### WireGuard Setup
- Installs WireGuard and related tools
- Generates server and client key pairs
- Configures WireGuard server (`wg0` interface)
- Sets up IP forwarding and iptables rules
- Starts and enables WireGuard service

### Security Configuration
- Configures UFW firewall
- Hardens SSH configuration
- Disables password authentication
- Enables key-based authentication only

### Network Configuration
- Configures IP forwarding for VPN routing
- Sets up NAT rules for client traffic
- Opens necessary firewall ports

## Generated Configurations

### Server Configuration (`/etc/wireguard/wg0.conf`)
```ini
[Interface]
PrivateKey = <server-private-key>
Address = 10.0.0.1/24
ListenPort = 51820
SaveConfig = true

PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = <client-public-key>
AllowedIPs = 10.0.0.2/32
```

### Client Configuration (`/root/wireguard-client/client.conf`)
```ini
[Interface]
PrivateKey = <client-private-key>
Address = 10.0.0.2/32
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = <server-public-key>
Endpoint = <vps-ip>:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

## Usage

### Automatic Setup (Recommended)

When you run `vps3 create`, the system will automatically:

1. Create the VPS
2. Wait for SSH to be available
3. Check if the OS is compatible (Ubuntu/Debian)
4. Run the Ansible playbook
5. Download the client configuration

### Manual Setup

If automatic setup fails or you want to run it manually:

1. **Generate inventory file** (replace with your VPS details):
   ```bash
   cat > ansible/inventory << EOF
   [vps]
   YOUR_VPS_IP ansible_host=YOUR_VPS_IP ansible_user=root ansible_ssh_private_key_file=PATH_TO_PRIVATE_KEY ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'

   [vps:vars]
   ansible_python_interpreter=/usr/bin/python3
   EOF
   ```

2. **Run the playbook**:
   ```bash
   ansible-playbook -i ansible/inventory ansible/playbook.yml
   ```

3. **Download client configuration**:
   ```bash
   scp -i YOUR_PRIVATE_KEY root@YOUR_VPS_IP:/root/wireguard-client/client.conf ./wireguard-clients/client-YOUR_VPS_IP.conf
   ```

## Network Namespace Setup

After the VPS is configured, you can set up a network namespace to route traffic through the VPN:

### Using vps3 Command
```bash
# Set up namespace (requires root)
sudo vps3 namespace setup

# Set up with custom name
sudo vps3 namespace setup myvps

# Execute commands in namespace
sudo vps3 namespace exec vps curl ipinfo.io

# Check namespace status
sudo vps3 namespace status vps

# List all namespaces
vps3 namespace list

# Delete namespace
sudo vps3 namespace delete vps
```

### Using Setup Script Directly
```bash
# Run the setup script
sudo scripts/setup-namespace.sh

# With custom namespace name
sudo scripts/setup-namespace.sh myvps

# With specific config file
sudo scripts/setup-namespace.sh myvps /path/to/client.conf
```

### Manual Setup
```bash
# Create namespace
sudo ip netns add vps

# Set up WireGuard in namespace
sudo ip netns exec vps wg-quick up /path/to/client.conf

# Test connectivity
sudo ip netns exec vps curl ipinfo.io

# Run commands in namespace
sudo ip netns exec vps bash
```

## Configuration Variables

You can customize the playbook by modifying variables in `playbook.yml`:

- `wireguard_port`: WireGuard listen port (default: 51820)
- `wireguard_interface`: Interface name (default: wg0)

## Troubleshooting

### Ansible Not Found
If you don't have Ansible installed, the system will skip automatic setup and provide manual instructions.

### OS Not Supported
The playbook only supports Ubuntu and Debian. If you select a different OS, Ansible setup will be skipped.

### SSH Connection Issues
- Make sure the VPS is fully booted (can take 1-2 minutes)
- Verify the private key permissions are correct (600)
- Check that SSH key was properly uploaded to the VPS

### WireGuard Issues
- Ensure UDP port 51820 is not blocked by your ISP
- Check VPS firewall settings
- Verify client configuration is correct

### Network Namespace Issues
- Network namespace operations require root privileges
- Make sure WireGuard tools are installed locally
- Check that the client configuration is valid

## Security Considerations

1. **Private Keys**: All private keys are stored securely with 600 permissions
2. **SSH Access**: Password authentication is disabled
3. **Firewall**: UFW is configured to only allow necessary ports
4. **Traffic Routing**: All VPN traffic is properly NAT'd through the VPS

## Advanced Usage

### Multiple Clients
To add more clients, you can modify the server configuration and add additional `[Peer]` sections.

### Custom DNS
Modify the `DNS` setting in the client configuration template to use different DNS servers.

### Split Tunneling
Change `AllowedIPs` in the client configuration to only route specific traffic through the VPN.

## Files Generated

After successful setup, you'll find:

- `wireguard-clients/client-<vps-ip>.conf` - Local client configuration
- `ansible/inventory` - Generated Ansible inventory file
- WireGuard keys and configurations on the VPS

## Support

If you encounter issues with the Ansible integration:

1. Check the Ansible output for specific error messages
2. Verify all prerequisites are installed
3. Ensure the VPS is running a supported OS (Ubuntu/Debian)
4. Test SSH connectivity manually before running Ansible