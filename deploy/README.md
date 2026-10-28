# DostoBot Deployment

Deploy DostoBot to a ~$4.50/month Hetzner Cloud server using Terraform and Ansible.

## Prerequisites

- [Terraform](https://terraform.io) >= 1.0
- [Ansible](https://ansible.com) >= 2.10
- [Task](https://taskfile.dev) (optional, for convenience)
- Hetzner Cloud account with an API token (`HCLOUD_TOKEN`)
- SSH key pair (default: `~/.ssh/id_hetzner` / `~/.ssh/id_hetzner.pub`)

## Quick Start

### 1. Configure Terraform

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

```hcl
hcloud_token         = "hcloud_your_token"
ssh_public_key_path = "~/.ssh/id_hetzner.pub"
location             = "ash"   # Ashburn, Virginia, USA
```

> If `HCLOUD_TOKEN` is already exported in your shell (or defined in the
> project `.env`), the `task deploy:infra` task regenerates
> `terraform.tfvars` from it automatically.

### 2. Create Infrastructure

```bash
task deploy:infra   # terraform init + apply, writes IP to deploy/.server_ip
```

Or manually:

```bash
cd deploy/terraform
terraform init
terraform apply
terraform output -raw server_ip > ../.server_ip
```

### 3. Configure Ansible

```bash
cd deploy/ansible
cp inventory.ini.example inventory.ini
```

Edit `inventory.ini` with your server IP:

```ini
[dostobot]
YOUR_SERVER_IP ansible_user=root ansible_ssh_private_key_file=~/.ssh/id_hetzner ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

> The `task deploy:ansible` task writes this file automatically from
> `deploy/.server_ip`.

### 4. Configure Production Environment

```bash
cp .env.production.example .env.production
```

Edit `.env.production` with your API keys and credentials.

### 5. Deploy

The all-in-one command builds the Linux binary, creates the Hetzner server,
waits for SSH, and runs the Ansible playbook:

```bash
task deploy
```

Or step by step:

```bash
task build:linux
task deploy:infra
task deploy:wait
task deploy:ansible
```

Or run Ansible directly:

```bash
task build:linux
cd deploy/ansible
ansible-playbook -i inventory.ini playbook.yml
```

## Deployment Commands

| Command | Description |
|---------|-------------|
| `task deploy` | Full deploy: build + infra + wait + Ansible |
| `task deploy:infra` | `terraform init` + `terraform apply` (creates server) |
| `task deploy:wait` | Wait for SSH to be reachable |
| `task deploy:ansible` | Configure server with Ansible |
| `task deploy:destroy` | Destroy server + firewall (DESTRUCTIVE, prompts) |
| `task update` | Rebuild and SCP just the binary, then restart service |

## Management Commands

These read the server IP from `deploy/.server_ip` automatically:

| Command | Description |
|---------|-------------|
| `task logs` | Tail production logs (`journalctl -u dostobot -f`) |
| `task status` | `systemctl status dostobot` on the server |
| `task ssh` | SSH into server as root |
| `task restart` | Restart the `dostobot` systemd service |
| `task backup` | SCP `dostobot.db` + `quotes.veclite` into `backups/` |

> If `deploy/.server_ip` is missing (e.g. after `deploy:destroy`), the
> management tasks refuse to run with a hint to run `task deploy` first.

## Infrastructure Details

### Server Specs (Hetzner CPX11, ~$4.50/month)

- 2 vCPU
- 2 GB RAM
- 40 GB SSD
- 20 TB transfer
- Location: `ash` (Ashburn, Virginia, USA) — closest to Mexico

### Firewall Rules

- SSH (22): Open inbound
- HTTP (80): Open inbound (for future webhooks)
- HTTPS (443): Open inbound
- Outbound: All allowed

### Resources Created

| Resource | Name | Notes |
|----------|------|-------|
| `hcloud_server.dostobot` | `dostobot` | CPX11, Ubuntu 24.04 |
| `hcloud_firewall.dostobot` | `dostobot-firewall` | Attached to the server |
| SSH key | `dostobot-key` | Created in Hetzner if not present |

> The SSH key is registered in Hetzner on first apply; `terraform destroy`
> does **not** delete it. Remove it manually (`hcloud ssh-key delete
> dostobot-key`) if you want a full teardown.

## Directory Structure on Server

```
/opt/dostobot/
├── bin/
│   └── dostobot          # Binary
├── data/
│   ├── dostobot.db       # SQLite database
│   └── quotes.veclite    # Vector database
├── veclite.yaml          # Embedder config
└── .env                  # Environment variables
```

## Systemd Service

The service runs as the `dostobot` user with security hardening:

- Memory limit: 512MB
- CPU quota: 80%
- Read-only system (except `/opt/dostobot/data`)

### Service Management (on the server)

```bash
sudo systemctl status dostobot
sudo systemctl restart dostobot
sudo journalctl -u dostobot -f
```

## Troubleshooting

### Check service status

```bash
task status
```

### View logs

```bash
task logs
```

### SSH and debug

```bash
task ssh
sudo -u dostobot /opt/dostobot/bin/dostobot stats
```

### Verify environment

```bash
sudo -u dostobot cat /opt/dostobot/.env
```

### After a destroy + redeploy

The new server gets a fresh IP. `task deploy` regenerates
`deploy/.server_ip` and `deploy/ansible/inventory.ini` automatically, so no
manual edit is needed. If you ran `task deploy:destroy`, the databases on
the old server are gone — re-run `task extract` + `task embed` to rebuild
the vector index before (or after) redeploying.