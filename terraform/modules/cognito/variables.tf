variable "name" {
  description = "Name of the Cognito User Pool."
  type        = string
}

variable "region" {
  description = "AWS region the pool lives in, used to build the issuer/JWKS URLs."
  type        = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
