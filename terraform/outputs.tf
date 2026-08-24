output "websocket_invoke_url" {
  description = "wss:// URL the frontend (VITE_WS_URL) connects to."
  value       = module.api_gateway_ws.invoke_url
}

output "cloudfront_domain_name" {
  value = module.s3_cloudfront.distribution_domain_name
}

output "cloudfront_distribution_id" {
  value = module.s3_cloudfront.distribution_id
}

output "frontend_bucket_name" {
  value = module.s3_cloudfront.bucket_name
}

output "cognito_user_pool_id" {
  value = module.cognito.user_pool_id
}

output "cognito_client_id" {
  value = module.cognito.client_id
}

output "application_arn" {
  value = module.application.arn
}

output "dynamodb_table_names" {
  value = {
    connections  = module.connections_table.table_name
    games        = module.games_table.table_name
    players      = module.players_table.table_name
    friendships  = module.friendships_table.table_name
    queue        = module.queue_table.table_name
    game_history = module.game_history_table.table_name
    rate_limits  = module.rate_limits_table.table_name
  }
}
