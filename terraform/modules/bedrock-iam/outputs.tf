output "policy_json" {
  description = "IAM policy document (JSON) granting Bedrock Converse access, to attach to the aimove Lambda role."
  value       = data.aws_iam_policy_document.bedrock_invoke.json
}
