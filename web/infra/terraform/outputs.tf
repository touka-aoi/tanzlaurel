output "tunnel_id" {
  description = "Cloudflare Tunnel ID"
  value       = cloudflare_zero_trust_tunnel_cloudflared.blog.id
}

output "tunnel_token" {
  description = "Tunnel token for cloudflared"
  value       = cloudflare_zero_trust_tunnel_cloudflared.blog.tunnel_token
  sensitive   = true
}

output "cname_target" {
  description = "CNAME target for the tunnel"
  value       = "${cloudflare_zero_trust_tunnel_cloudflared.blog.id}.cfargotunnel.com"
}
