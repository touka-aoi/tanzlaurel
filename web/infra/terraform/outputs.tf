output "tunnel_id" {
  description = "Cloudflare Tunnel ID"
  value       = cloudflare_zero_trust_tunnel_cloudflared.blog.id
}

output "tunnel_token" {
  description = "Tunnel token for cloudflared"
  value       = cloudflare_zero_trust_tunnel_cloudflared.blog.tunnel_secret
  sensitive   = true
}

output "cname_target" {
  description = "CNAME target for the tunnel"
  value       = "${cloudflare_zero_trust_tunnel_cloudflared.blog.id}.cfargotunnel.com"
}

output "access_application_aud" {
  description = "CF Access Application Audience Tag (CF_ACCESS_AUDIENCE env var)"
  value       = cloudflare_zero_trust_access_application.blog_admin.aud
}
