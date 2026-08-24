variable "model_arns" {
  description = "ARNs of the Bedrock foundation models / inference profiles the aimove Lambda is allowed to invoke via Converse."
  type        = list(string)
}
