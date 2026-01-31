# DostoBot Deployment

Deploy DostoBot to a $6/month DigitalOcean droplet using Terraform and Ansible.

## Prerequisites

- [Terraform](https://terraform.io) >= 1.0
- [Ansible](https://ansible.com) >= 2.10
- [Task](https://taskfile.dev) (optional, for convenience)
- DigitalOcean account with API token
- SSH key pair

## Quick Start

### 1. Configure Terraform

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:
```hcl
do_token            = "dop_v1_your_token"
ssh_public_key_path = "~/.ssh/id_rsa.pub"
region              = "nyc1"
```

### 2. Create Infrastructure

```bash
task infra:init    # Initialize Terraform
task infra:plan    # Preview changes
task infra:apply   # Create droplet
```

Or manually:
```bash
cd deploy/terraform
terraform init
terraform apply
```

Note the droplet IP from the output.

### 3. Configure Ansible

```bash
cd deploy/ansible
cp inventory.ini.example inventory.ini
```

Edit `inventory.ini` with your droplet IP:
```ini
[dostobot]
YOUR_DROPLET_IP ansible_user=root
```

### 4. Configure Production Environment

```bash
cp .env.production.example .env.production
```

Edit `.env.production` with your API keys and credentials.

### 5. Deploy

```bash
task deploy:setup   # First deployment (installs everything)
```

Or manually:
```bash
task build:linux
cd deploy/ansible
ansible-playbook -i inventory.ini playbook.yml
```

## Deployment Commands

| Command | Description |
|---------|-------------|
| `task infra:init` | Initialize Terraform |
| `task infra:plan` | Preview infrastructure changes |
| `task infra:apply` | Create/update infrastructure |
| `task infra:destroy` | Destroy droplet (DESTRUCTIVE) |
| `task deploy:setup` | Full server setup with Ansible |
| `task deploy` | Deploy binary only |
| `task deploy:quick DROPLET_IP=x.x.x.x` | Quick SCP deploy |

## Management Commands

| Command | Description |
|---------|-------------|
| `task logs DROPLET_IP=x.x.x.x` | Tail production logs |
| `task status DROPLET_IP=x.x.x.x` | Check service status |
| `task ssh DROPLET_IP=x.x.x.x` | SSH into server |
| `task backup DROPLET_IP=x.x.x.x` | Backup databases |

## Infrastructure Details

### Droplet Specs ($6/month)
- 1 vCPU
- 1 GB RAM
- 25 GB SSD
- 1 TB transfer

### Optional Add-ons
- Backups: +$1.20/month (set `enable_backups = true`)
- Reserved IP: +$4/month (set `enable_reserved_ip = true`)

### Firewall Rules
- SSH (22): Open
- HTTP (80): Open (for future webhooks)
- HTTPS (443): Open
- Outbound: All allowed

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

The service runs as `dostobot` user with security hardening:
- Memory limit: 512MB
- CPU quota: 80%
- Read-only system (except /opt/dostobot/data)

### Service Management

```bash
# On the server
sudo systemctl status dostobot
sudo systemctl restart dostobot
sudo journalctl -u dostobot -f
```

## Troubleshooting

### Check service status
```bash
task status DROPLET_IP=x.x.x.x
```

### View logs
```bash
task logs DROPLET_IP=x.x.x.x
```

### SSH and debug
```bash
task ssh DROPLET_IP=x.x.x.x
sudo -u dostobot /opt/dostobot/bin/dostobot stats
```

### Verify environment
```bash
sudo -u dostobot cat /opt/dostobot/.env
```
