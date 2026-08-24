variable "aws_region" {
  description = "AWS region for every resource except the ACM certificate (always us-east-1, see ADR-018)."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Deployment environment tag."
  type        = string
  default     = "prod"
}

variable "artifact_bucket" {
  description = "S3 bucket holding built Lambda zips, already existing (see ADR-041)."
  type        = string
  default     = "serverlessmate-artificats"
}

variable "artifact_version" {
  description = "Version folder under s3://<artifact_bucket>/lambda/<version>/<name>.zip consumed by every Lambda in this apply. Must match the Makefile's VERSION used to upload."
  type        = string
  default     = "v0.0.8"
}

variable "domain_name" {
  description = "Public domain for the frontend, served via CloudFront (ADR-018/ADR-029)."
  type        = string
  default     = "serverlessmate.kevindev.com.br"
}

variable "ws_domain_name" {
  description = "Custom domain for the WebSocket API Gateway (ADR-042). Regional cert, separate from the CloudFront one which must be us-east-1."
  type        = string
  default     = "ws.serverlessmate.kevindev.com.br"
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone id for kevindev.com.br. (ca0b66cdf24fc62e2e9e25f8a2cd4b16 was wrong — that's the Cloudflare *account* id, not this zone's.)"
  type        = string
  default     = "d23be5f1295b1afaf3da3c3230997041"
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token with DNS edit permission on the zone above."
  type        = string
  sensitive   = true
}

variable "reconnect_grace_ms" {
  description = "RECONNECT_GRACE_MS for connect/disconnect/makemove (ADR-026)."
  type        = string
  default     = "60000"
}

variable "bedrock_model_id" {
  description = "BEDROCK_MODEL_ID env var for aimove — a Bedrock model id or inference profile id used in the Converse API call (ADR-002/ADR-035). No default: the code has no fallback."
  type        = string
}

variable "bedrock_model_arns" {
  description = "ARNs the aimove Lambda is allowed to invoke via Bedrock Converse (model and/or inference profile ARN matching bedrock_model_id)."
  type        = list(string)
}
