output "certificate_arn" {
  description = "Raw (not-yet-validated) ACM certificate ARN, needed by root's aws_acm_certificate_validation resource."
  value       = aws_acm_certificate.this.arn
}

output "domain_validation_options" {
  value = aws_acm_certificate.this.domain_validation_options
}

output "distribution_id" {
  value = aws_cloudfront_distribution.this.id
}

output "distribution_domain_name" {
  value = aws_cloudfront_distribution.this.domain_name
}

output "bucket_name" {
  value = aws_s3_bucket.frontend.id
}
