variable "zone_id" {
  description = "Cloudflare zone id for kevindev.com.br."
  type        = string
}

variable "name" {
  type = string
}

variable "type" {
  type = string
}

variable "value" {
  type = string
}

variable "ttl" {
  type    = number
  default = 1
}

variable "proxied" {
  type    = bool
  default = false
}
