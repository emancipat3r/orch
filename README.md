# orch

A CLI tool for provisioning and managing VPS instances across multiple cloud providers with automated WireGuard VPN setup.

## Features

- **Multi-provider support** - Linode, DigitalOcean, and Vultr
- **Interactive wizard** - TUI-based provider, region, image, and plan selection
- **Automated WireGuard VPN** - Generates keypairs, configures server and client, sets up a Linux network namespace for traffic isolation
- **Per-instance SSH keys** - Ed25519 keypairs with passphrases, automatically added to ssh-agent
- **Post-provisioning setup** - Ansible playbook handles package installation, SSH hardening, firewall configuration, and WireGuard deployment
- **Instance tracking** - Local TOML-based instance registry for managing lifecycle
- **Bulk destroy** - Multi-select instances for teardown with automatic cleanup of WireGuard interfaces, network namespaces, and local config

## Requirements

- Go 1.24+
- Ansible
- WireGuard tools (`wg`, `wg-quick`)
- Linux (uses netlink and network namespaces)
- Root/sudo access (for network namespace and WireGuard operations)

## Installation

```bash
git clone https://github.com/emancipat3r/orch.git
cd orch
go build -o orch .
```

## Configuration

On first run, `orch` creates its config directory at `~/.config/orch/` and copies a configuration template. Edit `~/.config/orch/config/configuration.toml` with your provider API keys:

```toml
[digitalocean]
key = "your_digitalocean_api_key"

[linode]
key = "your_linode_api_key"

[vultr]
key = "your_vultr_api_key"
```

## Usage

```bash
# Provision a new VPS with WireGuard
orch create

# Provision with a custom name
orch create --name my-vps

# List running instances
orch list

# Re-run post-provisioning setup on an existing instance
orch setup

# Destroy instance(s)
orch destroy
```

## How it works

1. **Create** - Select a provider, region, image, and size through an interactive wizard. `orch` generates a per-instance SSH keypair, uploads it to the provider, provisions the VPS, runs an Ansible playbook to configure WireGuard and harden SSH, then sets up a local network namespace with the WireGuard tunnel.

2. **Setup** - Re-run the Ansible playbook on an existing instance (useful if the initial setup failed).

3. **List** - Query the provider API and display running instances in a table.

4. **Destroy** - Multi-select instances for deletion. Tears down the remote instance, local WireGuard interface, network namespace, and cleans up config files.

## Project structure

```
cmd/           CLI commands (create, destroy, list, setup)
providers/     Provider API clients (Linode, DigitalOcean, Vultr)
utils/         SSH, networking, Ansible, and instance management utilities
ui/            TUI components (wizard, selection, confirmation, tables)
logger/        Structured logging
ansible/       Post-provisioning playbook
templates/     WireGuard and config templates
```

## License

GPL-3.0
