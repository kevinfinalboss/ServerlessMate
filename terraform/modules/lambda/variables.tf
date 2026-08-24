variable "name" {
  description = "Logical Lambda name; the function is named serverlessmate-<name>-lambda."
  type        = string
}

variable "artifact_bucket" {
  description = "S3 bucket holding the built Lambda zip artifacts."
  type        = string
}

variable "artifact_version" {
  description = "Version folder under s3://<artifact_bucket>/lambda/<version>/<name>.zip that this deploy consumes."
  type        = string
}

variable "environment" {
  description = "Environment variables injected into the function."
  type        = map(string)
  default     = {}
}

variable "dynamodb_table_arns" {
  description = "DynamoDB table ARNs this function needs access to (table-level actions, plus their GSIs)."
  type        = list(string)
  default     = []
}

variable "dynamodb_actions" {
  description = "DynamoDB actions granted on dynamodb_table_arns."
  type        = list(string)
  default = [
    "dynamodb:GetItem",
    "dynamodb:PutItem",
    "dynamodb:UpdateItem",
    "dynamodb:DeleteItem",
    "dynamodb:Query",
    "dynamodb:Scan",
    "dynamodb:TransactWriteItems",
    "dynamodb:TransactGetItems",
  ]
}

variable "extra_policy_json" {
  description = "Additional IAM policy document (JSON) to merge in, e.g. Bedrock access for aimove."
  type        = string
  default     = null
}

variable "websocket_execution_arn" {
  description = <<-EOT
    Execution ARN of the WebSocket API, only for functions that call PostToConnection
    without being an API Gateway integration target themselves (matchmaker, triggered by
    DynamoDB Streams — ADR-033). The 9 functions that ARE integration targets get their
    execute-api:ManageConnections + lambda:InvokeFunction permissions from modules/api-gateway-ws
    instead, to avoid a module dependency cycle (that module needs this one's invoke_arn,
    so this one can't also depend on its execution_arn).
  EOT
  type        = string
  default     = null
}

variable "enable_websocket_manage_connections" {
  description = <<-EOT
    Set true together with websocket_execution_arn. Kept as a separate plain
    boolean (rather than checking `websocket_execution_arn != null`) because
    that ARN comes from a not-yet-created API Gateway API and is unknown at
    plan time — a count/for_each can't branch on an unknown value, but this
    flag is a literal the caller always knows statically.
  EOT
  type        = bool
  default     = false
}

variable "stream_arn" {
  description = "DynamoDB Stream ARN to trigger this function from, if any (matchmaker only)."
  type        = string
  default     = null
}

variable "enable_stream_trigger" {
  description = "Set true together with stream_arn, for the same reason as enable_websocket_manage_connections above (a new table's stream_arn is unknown at plan time)."
  type        = bool
  default     = false
}

variable "memory_size" {
  type    = number
  default = 256
}

variable "timeout" {
  type    = number
  default = 15
}

variable "tags" {
  type    = map(string)
  default = {}
}
