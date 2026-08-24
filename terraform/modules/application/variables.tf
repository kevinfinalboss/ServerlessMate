variable "name" {
  description = "Name of the Resource Group."
  type        = string
}

variable "description" {
  description = "Description of the group."
  type        = string
  default     = ""
}

variable "tag_key" {
  description = "Tag key the group's resource_query filters on. Every resource in the stack must carry this tag (see root local.common_tags) to show up in the group."
  type        = string
}

variable "tag_value" {
  description = "Tag value the group's resource_query filters on."
  type        = string
}

variable "tags" {
  description = "Tags applied to the group resource itself."
  type        = map(string)
  default     = {}
}
