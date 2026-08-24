locals {
  routes_by_key = { for r in var.routes : r.route_key => r }
  functions_by_name = { for r in var.routes : r.function_name => r... }
}

resource "aws_apigatewayv2_api" "this" {
  name                       = var.name
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
  tags                       = var.tags
}

resource "aws_apigatewayv2_integration" "this" {
  for_each = local.routes_by_key

  api_id             = aws_apigatewayv2_api.this.id
  integration_type   = "AWS_PROXY"
  integration_uri    = each.value.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_route" "this" {
  for_each = local.routes_by_key

  api_id    = aws_apigatewayv2_api.this.id
  route_key = each.key
  target    = "integrations/${aws_apigatewayv2_integration.this[each.key].id}"
}

resource "aws_lambda_permission" "invoke" {
  for_each = local.functions_by_name

  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = each.value[0].function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.this.execution_arn}/*/*"
}

data "aws_iam_policy_document" "manage_connections" {
  statement {
    sid       = "PostToConnection"
    actions   = ["execute-api:ManageConnections"]
    resources = ["${aws_apigatewayv2_api.this.execution_arn}/*/POST/@connections/*"]
  }
}

resource "aws_iam_role_policy" "manage_connections" {
  for_each = local.functions_by_name

  name   = "manage-connections"
  role   = each.value[0].role_name
  policy = data.aws_iam_policy_document.manage_connections.json
}

resource "aws_cloudwatch_log_group" "access" {
  name              = "/aws/apigateway/${var.name}"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_apigatewayv2_stage" "this" {
  api_id      = aws_apigatewayv2_api.this.id
  name        = var.stage_name
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.access.arn
    format = jsonencode({
      requestId    = "$context.requestId"
      connectionId = "$context.connectionId"
      eventType    = "$context.eventType"
      routeKey     = "$context.routeKey"
      status       = "$context.status"
      errorMessage = "$context.error.message"
    })
  }

  tags = var.tags
}

resource "aws_apigatewayv2_domain_name" "this" {
  count = var.custom_domain_name != null ? 1 : 0

  domain_name = var.custom_domain_name

  domain_name_configuration {
    certificate_arn = var.validated_certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }

  tags = var.tags
}

resource "aws_apigatewayv2_api_mapping" "this" {
  count = var.custom_domain_name != null ? 1 : 0

  api_id          = aws_apigatewayv2_api.this.id
  domain_name     = aws_apigatewayv2_domain_name.this[0].id
  stage           = aws_apigatewayv2_stage.this.id
  api_mapping_key = var.stage_name
}
