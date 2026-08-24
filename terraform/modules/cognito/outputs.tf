output "user_pool_id" {
  value = aws_cognito_user_pool.this.id
}

output "client_id" {
  value = aws_cognito_user_pool_client.spa.id
}

output "issuer_url" {
  description = "COGNITO_ISSUER for the connect Lambda."
  value       = "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.this.id}"
}

output "jwks_url" {
  description = "COGNITO_JWKS_URL for the connect Lambda."
  value       = "https://cognito-idp.${var.region}.amazonaws.com/${aws_cognito_user_pool.this.id}/.well-known/jwks.json"
}
