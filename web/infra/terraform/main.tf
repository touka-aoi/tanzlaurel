terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "blog" {
  account_id = var.cloudflare_account_id
  name       = var.tunnel_name
  tunnel_secret     = var.tunnel_secret
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "blog" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.blog.id

  config = {
    ingress = [
      {
        hostname = var.domain
        service  = "http://nginx:80"
      },
      {
        service = "http_status:404"
      },
    ]
  }
}

resource "cloudflare_dns_record" "blog" {
  zone_id = var.cloudflare_zone_id
  name    = var.domain
  type    = "CNAME"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.blog.id}.cfargotunnel.com"
  proxied = true
  ttl     = 1
}
