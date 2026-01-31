terraform {
  required_version = ">= 1.0"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.45"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

# Look up existing SSH key or create new one
data "hcloud_ssh_keys" "all" {}

locals {
  ssh_public_key = file(var.ssh_public_key_path)
  # Check if key already exists by fingerprint
  existing_key = [for k in data.hcloud_ssh_keys.all.ssh_keys : k if k.public_key == local.ssh_public_key]
}

resource "hcloud_ssh_key" "dostobot" {
  count      = length(local.existing_key) == 0 ? 1 : 0
  name       = "dostobot-key"
  public_key = local.ssh_public_key
}

locals {
  ssh_key_id = length(local.existing_key) > 0 ? local.existing_key[0].id : hcloud_ssh_key.dostobot[0].id
}

# CPX11: 2 vCPU, 2GB RAM, 40GB SSD - ~$4.50/month (US datacenter)
resource "hcloud_server" "dostobot" {
  name        = "dostobot"
  server_type = "cpx11"
  image       = "ubuntu-24.04"
  location    = var.location
  ssh_keys    = [local.ssh_key_id]

  labels = {
    app = "dostobot"
    env = "production"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
}

# Firewall rules
resource "hcloud_firewall" "dostobot" {
  name = "dostobot-firewall"

  # SSH
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # HTTP
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # HTTPS
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  apply_to {
    server = hcloud_server.dostobot.id
  }
}
