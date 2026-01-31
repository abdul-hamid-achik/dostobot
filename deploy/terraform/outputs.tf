output "server_ip" {
  description = "Public IPv4 address of the server"
  value       = hcloud_server.dostobot.ipv4_address
}

output "server_id" {
  description = "Server ID"
  value       = hcloud_server.dostobot.id
}

output "server_status" {
  description = "Server status"
  value       = hcloud_server.dostobot.status
}

output "ssh_command" {
  description = "SSH command to connect"
  value       = "ssh root@${hcloud_server.dostobot.ipv4_address}"
}
