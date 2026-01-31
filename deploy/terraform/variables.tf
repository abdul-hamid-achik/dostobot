variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key"
  type        = string
  default     = "~/.ssh/id_hetzner.pub"
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "ash"  # Ashburn, Virginia, USA (closest to Mexico)
}
