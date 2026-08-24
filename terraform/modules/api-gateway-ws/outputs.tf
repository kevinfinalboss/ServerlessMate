output "api_id" {
  value = aws_apigatewayv2_api.this.id
}

output "execution_arn" {
  description = "Used as source_arn in aws_lambda_permission and as the resource prefix for execute-api:ManageConnections."
  value       = aws_apigatewayv2_api.this.execution_arn
}

output "invoke_url" {
  description = "wss:// URL the frontend (VITE_WS_URL) connects to — uses the custom domain when set."
  value = var.custom_domain_name != null ? (
    "wss://${var.custom_domain_name}/${var.stage_name}"
    ) : (
    aws_apigatewayv2_stage.this.invoke_url
  )
}

output "management_endpoint" {
  description = "https:// endpoint for ApiGatewayManagementApi PostToConnection calls (WEBSOCKET_API_ENDPOINT for the matchmaker Lambda) — uses the custom domain when set."
  value = var.custom_domain_name != null ? (
    "https://${var.custom_domain_name}/${var.stage_name}"
    ) : (
    "${replace(aws_apigatewayv2_api.this.api_endpoint, "wss://", "https://")}/${aws_apigatewayv2_stage.this.name}"
  )
}

output "custom_domain_target" {
  description = "Regional target domain to point a CNAME at (null when custom_domain_name is not set)."
  value       = var.custom_domain_name != null ? aws_apigatewayv2_domain_name.this[0].domain_name_configuration[0].target_domain_name : null
}
