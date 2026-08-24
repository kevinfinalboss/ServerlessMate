data "aws_iam_policy_document" "bedrock_invoke" {
  statement {
    sid = "BedrockConverse"
    actions = [
      "bedrock:InvokeModel",
      "bedrock:InvokeModelWithResponseStream",
    ]
    resources = var.model_arns
  }
}
