variable "name" {
  description = "Logical table name; the actual table is named serverlessmate-<name>."
  type        = string
}

variable "hash_key" {
  description = "Partition key attribute name."
  type        = string
}

variable "hash_key_type" {
  description = "Partition key attribute type (S, N or B)."
  type        = string
  default     = "S"
}

variable "range_key" {
  description = "Sort key attribute name, if any."
  type        = string
  default     = null
}

variable "range_key_type" {
  description = "Sort key attribute type (S, N or B)."
  type        = string
  default     = "S"
}

variable "global_secondary_indexes" {
  description = "List of GSIs: name, hash_key, hash_key_type, range_key (optional), range_key_type (optional), projection_type."
  type = list(object({
    name            = string
    hash_key        = string
    hash_key_type   = optional(string, "S")
    range_key       = optional(string)
    range_key_type  = optional(string, "S")
    projection_type = optional(string, "ALL")
  }))
  default = []
}

variable "stream_enabled" {
  description = "Whether to enable a DynamoDB Stream on this table."
  type        = bool
  default     = false
}

variable "stream_view_type" {
  description = "Stream view type when stream_enabled is true."
  type        = string
  default     = "NEW_AND_OLD_IMAGES"
}

variable "ttl_attribute_name" {
  description = "Attribute name used for item TTL, if any."
  type        = string
  default     = null
}

variable "tags" {
  description = "Tags applied to the table."
  type        = map(string)
  default     = {}
}
