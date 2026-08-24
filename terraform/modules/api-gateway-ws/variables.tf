variable "name" {
  description = "Name of the WebSocket API."
  type        = string
}

variable "routes" {
  description = <<-EOT
    One entry per WS route. Multiple routes can point at the same Lambda
    (e.g. move/resign/offerDraw/acceptDraw/chat all go to makemove) — permissions
    and the ManageConnections policy are deduplicated per function_name.
  EOT
  type = list(object({
    route_key     = string
    invoke_arn    = string
    function_name = string
    role_name     = string
  }))
}

variable "stage_name" {
  type    = string
  default = "prod"
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "custom_domain_name" {
  description = "Custom domain to map onto the API (e.g. ws.serverlessmate.kevindev.com.br). Null skips the custom domain entirely."
  type        = string
  default     = null
}

variable "validated_certificate_arn" {
  description = "ARN of an already-validated REGIONAL ACM certificate for custom_domain_name (must live in the same region as this API — WebSocket APIs don't support edge-optimized custom domains). Required when custom_domain_name is set."
  type        = string
  default     = null
}

variable "tags" {
  type    = map(string)
  default = {}
}
