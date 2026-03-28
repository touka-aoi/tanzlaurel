variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for the domain"
  type        = string
}

variable "domain" {
  description = "Domain name"
  type        = string
  default     = "toukaaoi.dev"
}

variable "tunnel_name" {
  description = "Cloudflare Tunnel name"
  type        = string
  default     = "crdt-blog"
}

variable "tunnel_secret" {
  description = "Tunnel secret (base64-encoded, at least 32 bytes)"
  type        = string
  sensitive   = true
}

variable "google_oauth_client_id" {
  description = "Google OAuth client ID for CF Access"
  type        = string
}

variable "google_oauth_client_secret" {
  description = "Google OAuth client secret for CF Access"
  type        = string
  sensitive   = true
}

variable "admin_email" {
  description = "Admin email address allowed to access protected paths"
  type        = string
}
