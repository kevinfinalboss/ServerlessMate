terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

resource "cloudflare_record" "this" {
  zone_id = var.zone_id
  name    = var.name
  type    = var.type
  content = var.value
  ttl     = var.proxied ? 1 : var.ttl
  proxied = var.proxied
}
