variable "domain_name" {
  description = "Public domain served by CloudFront, e.g. serverlessmate.kevindev.com.br."
  type        = string
}

variable "bucket_name" {
  description = "S3 bucket name that stores the built frontend."
  type        = string
}

variable "validated_certificate_arn" {
  description = "ARN of the already-validated ACM certificate used by the distribution's viewer_certificate."
  type        = string
}

variable "price_class" {
  type    = string
  default = "PriceClass_100"
}

variable "tags" {
  type    = map(string)
  default = {}
}
