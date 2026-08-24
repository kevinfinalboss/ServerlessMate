locals {
  common_tags = {
    Project     = "ServerlessMate"
    ManagedBy   = "terraform"
    Environment = var.environment
  }

  frontend_bucket_name = "serverlessmate-frontend"
}

module "application" {
  source      = "./modules/application"
  name        = "ServerlessMate"
  description = "Xadrez multiplayer serverless na AWS - WebSocket API Gateway e Lambda em Go com DynamoDB Cognito e Bedrock"
  tag_key     = "Project"
  tag_value   = local.common_tags.Project
  tags        = local.common_tags
}

module "connections_table" {
  source   = "./modules/dynamodb"
  name     = "connections"
  hash_key = "connectionId"

  global_secondary_indexes = [
    {
      name      = "GameConnectionsIndex"
      hash_key  = "gameId"
      range_key = "connectionId"
    },
  ]

  tags = local.common_tags
}

module "games_table" {
  source   = "./modules/dynamodb"
  name     = "games"
  hash_key = "gameId"

  global_secondary_indexes = [
    {
      name           = "PlayerHistoryIndex"
      hash_key       = "playerId"
      range_key      = "endedAt"
      range_key_type = "N"
    },
  ]

  tags = local.common_tags
}

module "players_table" {
  source   = "./modules/dynamodb"
  name     = "players"
  hash_key = "playerId"

  global_secondary_indexes = [
    {
      name           = "LeaderboardIndex"
      hash_key       = "leaderboardPK"
      range_key      = "rating"
      range_key_type = "N"
    },
  ]

  tags = local.common_tags
}

module "friendships_table" {
  source    = "./modules/dynamodb"
  name      = "friendships"
  hash_key  = "playerId"
  range_key = "friendId"

  global_secondary_indexes = [
    {
      name      = "FriendIDIndex"
      hash_key  = "friendId"
      range_key = "playerId"
    },
  ]

  tags = local.common_tags
}

module "queue_table" {
  source           = "./modules/dynamodb"
  name             = "queue"
  hash_key         = "matchmakingKey"
  range_key        = "sortKey"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  tags = local.common_tags
}

module "game_history_table" {
  source         = "./modules/dynamodb"
  name           = "game-history"
  hash_key       = "playerId"
  range_key      = "endedAt"
  range_key_type = "N"

  tags = local.common_tags
}

module "rate_limits_table" {
  source             = "./modules/dynamodb"
  name               = "rate-limits"
  hash_key           = "playerDate"
  ttl_attribute_name = "ttl"

  tags = local.common_tags
}

module "cognito" {
  source = "./modules/cognito"
  name   = "serverlessmate"
  region = var.aws_region
  tags   = local.common_tags
}

module "bedrock_iam" {
  source     = "./modules/bedrock-iam"
  model_arns = var.bedrock_model_arns
}

module "lambda_connect" {
  source           = "./modules/lambda"
  name             = "connect"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    COGNITO_JWKS_URL   = module.cognito.jwks_url
    COGNITO_ISSUER     = module.cognito.issuer_url
    RECONNECT_GRACE_MS = var.reconnect_grace_ms
    GAMES_TABLE        = module.games_table.table_name
    CONNECTIONS_TABLE  = module.connections_table.table_name
  }

  dynamodb_table_arns = [
    module.games_table.table_arn,
    module.connections_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_disconnect" {
  source           = "./modules/lambda"
  name             = "disconnect"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    GAMES_TABLE       = module.games_table.table_name
    CONNECTIONS_TABLE = module.connections_table.table_name
  }

  dynamodb_table_arns = [
    module.games_table.table_arn,
    module.connections_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_makemove" {
  source           = "./modules/lambda"
  name             = "makemove"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    RECONNECT_GRACE_MS = var.reconnect_grace_ms
    GAMES_TABLE        = module.games_table.table_name
    CONNECTIONS_TABLE  = module.connections_table.table_name
    PLAYERS_TABLE      = module.players_table.table_name
    GAME_HISTORY_TABLE = module.game_history_table.table_name
  }

  dynamodb_table_arns = [
    module.games_table.table_arn,
    module.connections_table.table_arn,
    module.players_table.table_arn,
    module.game_history_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_aimove" {
  source           = "./modules/lambda"
  name             = "aimove"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    GAMES_TABLE        = module.games_table.table_name
    CONNECTIONS_TABLE  = module.connections_table.table_name
    RATE_LIMITS_TABLE  = module.rate_limits_table.table_name
    GAME_HISTORY_TABLE = module.game_history_table.table_name
    BEDROCK_MODEL_ID   = var.bedrock_model_id
  }

  dynamodb_table_arns = [
    module.games_table.table_arn,
    module.connections_table.table_arn,
    module.rate_limits_table.table_arn,
    module.game_history_table.table_arn,
  ]

  extra_policy_json = module.bedrock_iam.policy_json

  tags = local.common_tags
}

module "lambda_getprofile" {
  source           = "./modules/lambda"
  name             = "getprofile"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    CONNECTIONS_TABLE = module.connections_table.table_name
    PLAYERS_TABLE     = module.players_table.table_name
    FRIENDSHIPS_TABLE = module.friendships_table.table_name
  }

  dynamodb_table_arns = [
    module.connections_table.table_arn,
    module.players_table.table_arn,
    module.friendships_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_friends" {
  source           = "./modules/lambda"
  name             = "friends"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    CONNECTIONS_TABLE = module.connections_table.table_name
    FRIENDSHIPS_TABLE = module.friendships_table.table_name
    PLAYERS_TABLE     = module.players_table.table_name
  }

  dynamodb_table_arns = [
    module.connections_table.table_arn,
    module.friendships_table.table_arn,
    module.players_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_leaderboard" {
  source           = "./modules/lambda"
  name             = "leaderboard"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    PLAYERS_TABLE = module.players_table.table_name
  }

  dynamodb_table_arns = [module.players_table.table_arn]

  tags = local.common_tags
}

module "lambda_joinqueue" {
  source           = "./modules/lambda"
  name             = "joinqueue"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    CONNECTIONS_TABLE = module.connections_table.table_name
    PLAYERS_TABLE     = module.players_table.table_name
    QUEUE_TABLE       = module.queue_table.table_name
  }

  dynamodb_table_arns = [
    module.connections_table.table_arn,
    module.players_table.table_arn,
    module.queue_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_history" {
  source           = "./modules/lambda"
  name             = "history"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    CONNECTIONS_TABLE  = module.connections_table.table_name
    GAMES_TABLE        = module.games_table.table_name
    GAME_HISTORY_TABLE = module.game_history_table.table_name
    PLAYERS_TABLE      = module.players_table.table_name
  }

  dynamodb_table_arns = [
    module.connections_table.table_arn,
    module.games_table.table_arn,
    module.game_history_table.table_arn,
    module.players_table.table_arn,
  ]

  tags = local.common_tags
}

module "lambda_matchmaker" {
  source           = "./modules/lambda"
  name             = "matchmaker"
  artifact_bucket  = var.artifact_bucket
  artifact_version = var.artifact_version

  environment = {
    WEBSOCKET_API_ENDPOINT = module.api_gateway_ws.management_endpoint
    GAMES_TABLE            = module.games_table.table_name
    QUEUE_TABLE            = module.queue_table.table_name
    CONNECTIONS_TABLE      = module.connections_table.table_name
  }

  dynamodb_table_arns = [
    module.games_table.table_arn,
    module.queue_table.table_arn,
    module.connections_table.table_arn,
  ]

  stream_arn            = module.queue_table.stream_arn
  enable_stream_trigger = true

  websocket_execution_arn             = module.api_gateway_ws.execution_arn
  enable_websocket_manage_connections = true

  tags = local.common_tags
}

locals {
  ws_routes = concat(
    [for rk in ["move", "resign", "offerDraw", "acceptDraw", "chat"] : {
      route_key     = rk
      invoke_arn    = module.lambda_makemove.invoke_arn
      function_name = module.lambda_makemove.function_name
      role_name     = module.lambda_makemove.role_name
    }],
    [for rk in ["start", "aiMove"] : {
      route_key     = rk
      invoke_arn    = module.lambda_aimove.invoke_arn
      function_name = module.lambda_aimove.function_name
      role_name     = module.lambda_aimove.role_name
    }],
    [for rk in ["sendRequest", "acceptRequest", "block", "listFriends", "cancelRequest"] : {
      route_key     = rk
      invoke_arn    = module.lambda_friends.invoke_arn
      function_name = module.lambda_friends.function_name
      role_name     = module.lambda_friends.role_name
    }],
    [for rk in ["joinQueue", "leaveQueue"] : {
      route_key     = rk
      invoke_arn    = module.lambda_joinqueue.invoke_arn
      function_name = module.lambda_joinqueue.function_name
      role_name     = module.lambda_joinqueue.role_name
    }],
    [for rk in ["getProfile", "updateProfile"] : {
      route_key     = rk
      invoke_arn    = module.lambda_getprofile.invoke_arn
      function_name = module.lambda_getprofile.function_name
      role_name     = module.lambda_getprofile.role_name
    }],
    [
      {
        route_key     = "$connect"
        invoke_arn    = module.lambda_connect.invoke_arn
        function_name = module.lambda_connect.function_name
        role_name     = module.lambda_connect.role_name
      },
      {
        route_key     = "$disconnect"
        invoke_arn    = module.lambda_disconnect.invoke_arn
        function_name = module.lambda_disconnect.function_name
        role_name     = module.lambda_disconnect.role_name
      },
      {
        route_key     = "leaderboard"
        invoke_arn    = module.lambda_leaderboard.invoke_arn
        function_name = module.lambda_leaderboard.function_name
        role_name     = module.lambda_leaderboard.role_name
      },
      {
        route_key     = "history"
        invoke_arn    = module.lambda_history.invoke_arn
        function_name = module.lambda_history.function_name
        role_name     = module.lambda_history.role_name
      },
    ],
  )
}

data "aws_iam_policy_document" "apigw_cloudwatch_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "apigw_cloudwatch" {
  name               = "serverlessmate-apigateway-cloudwatch-role"
  assume_role_policy = data.aws_iam_policy_document.apigw_cloudwatch_assume.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "apigw_cloudwatch" {
  role       = aws_iam_role.apigw_cloudwatch.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
}

resource "aws_api_gateway_account" "this" {
  cloudwatch_role_arn = aws_iam_role.apigw_cloudwatch.arn
}

module "api_gateway_ws" {
  source = "./modules/api-gateway-ws"
  name   = "serverlessmate"
  routes = local.ws_routes
  tags   = local.common_tags

  custom_domain_name        = var.ws_domain_name
  validated_certificate_arn = aws_acm_certificate_validation.ws_api.certificate_arn

  depends_on = [aws_api_gateway_account.this]
}

resource "aws_acm_certificate" "ws_api" {
  domain_name       = var.ws_domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = local.common_tags
}

module "dns_validation_ws" {
  source  = "./modules/cloudflare-dns"
  zone_id = var.cloudflare_zone_id

  name  = trimsuffix(one(aws_acm_certificate.ws_api.domain_validation_options).resource_record_name, ".")
  type  = one(aws_acm_certificate.ws_api.domain_validation_options).resource_record_type
  value = trimsuffix(one(aws_acm_certificate.ws_api.domain_validation_options).resource_record_value, ".")
}

resource "aws_acm_certificate_validation" "ws_api" {
  certificate_arn         = aws_acm_certificate.ws_api.arn
  validation_record_fqdns = [module.dns_validation_ws.fqdn]
}

module "dns_ws_app" {
  source  = "./modules/cloudflare-dns"
  zone_id = var.cloudflare_zone_id

  name  = var.ws_domain_name
  type  = "CNAME"
  value = module.api_gateway_ws.custom_domain_target
}

module "s3_cloudfront" {
  source = "./modules/s3-cloudfront"
  providers = {
    aws.us_east_1 = aws.us_east_1
  }

  domain_name               = var.domain_name
  bucket_name               = local.frontend_bucket_name
  validated_certificate_arn = aws_acm_certificate_validation.frontend.certificate_arn

  tags = local.common_tags
}

module "dns_validation" {
  source  = "./modules/cloudflare-dns"
  zone_id = var.cloudflare_zone_id

  name  = trimsuffix(one(module.s3_cloudfront.domain_validation_options).resource_record_name, ".")
  type  = one(module.s3_cloudfront.domain_validation_options).resource_record_type
  value = trimsuffix(one(module.s3_cloudfront.domain_validation_options).resource_record_value, ".")
}

resource "aws_acm_certificate_validation" "frontend" {
  provider = aws.us_east_1

  certificate_arn         = module.s3_cloudfront.certificate_arn
  validation_record_fqdns = [module.dns_validation.fqdn]
}

module "dns_app" {
  source  = "./modules/cloudflare-dns"
  zone_id = var.cloudflare_zone_id

  name  = var.domain_name
  type  = "CNAME"
  value = module.s3_cloudfront.distribution_domain_name
}
