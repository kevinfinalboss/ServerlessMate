data "aws_iam_policy_document" "assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "this" {
  name               = "serverlessmate-${var.name}-lambda-role"
  assume_role_policy = data.aws_iam_policy_document.assume.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "basic_execution" {
  role       = aws_iam_role.this.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

data "aws_iam_policy_document" "dynamodb" {
  count = length(var.dynamodb_table_arns) > 0 ? 1 : 0

  statement {
    sid       = "DynamoDBAccess"
    actions   = var.dynamodb_actions
    resources = distinct(concat(var.dynamodb_table_arns, [for arn in var.dynamodb_table_arns : "${arn}/index/*"]))
  }
}

data "aws_iam_policy_document" "websocket" {
  count = var.enable_websocket_manage_connections ? 1 : 0

  statement {
    sid       = "PostToConnection"
    actions   = ["execute-api:ManageConnections"]
    resources = ["${var.websocket_execution_arn}/*/POST/@connections/*"]
  }
}

data "aws_iam_policy_document" "stream_read" {
  count = var.enable_stream_trigger ? 1 : 0

  statement {
    sid = "DynamoDBStreamRead"
    actions = [
      "dynamodb:GetRecords",
      "dynamodb:GetShardIterator",
      "dynamodb:DescribeStream",
      "dynamodb:ListStreams",
    ]
    resources = [var.stream_arn]
  }
}

data "aws_iam_policy_document" "combined" {
  source_policy_documents = compact([
    length(data.aws_iam_policy_document.dynamodb) > 0 ? data.aws_iam_policy_document.dynamodb[0].json : null,
    length(data.aws_iam_policy_document.websocket) > 0 ? data.aws_iam_policy_document.websocket[0].json : null,
    length(data.aws_iam_policy_document.stream_read) > 0 ? data.aws_iam_policy_document.stream_read[0].json : null,
    var.extra_policy_json,
  ])
}

resource "aws_iam_role_policy" "this" {
  name   = "serverlessmate-${var.name}-lambda-policy"
  role   = aws_iam_role.this.id
  policy = data.aws_iam_policy_document.combined.json
}

resource "aws_lambda_function" "this" {
  function_name = "serverlessmate-${var.name}-lambda"
  role          = aws_iam_role.this.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = var.memory_size
  timeout       = var.timeout

  s3_bucket = var.artifact_bucket
  s3_key    = "lambda/${var.artifact_version}/${var.name}.zip"

  environment {
    variables = var.environment
  }

  tags = var.tags

  depends_on = [
    aws_iam_role_policy.this,
    aws_iam_role_policy_attachment.basic_execution,
  ]
}

resource "aws_lambda_event_source_mapping" "stream" {
  count = var.enable_stream_trigger ? 1 : 0

  event_source_arn  = var.stream_arn
  function_name     = aws_lambda_function.this.arn
  starting_position = "LATEST"

  filter_criteria {
    filter {
      pattern = jsonencode({ eventName = ["INSERT"] })
    }
  }
}
