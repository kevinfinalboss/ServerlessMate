locals {
  base_attributes = concat(
    [{ name = var.hash_key, type = var.hash_key_type }],
    var.range_key != null ? [{ name = var.range_key, type = var.range_key_type }] : []
  )

  gsi_attributes = flatten([
    for gsi in var.global_secondary_indexes : concat(
      [{ name = gsi.hash_key, type = gsi.hash_key_type }],
      gsi.range_key != null ? [{ name = gsi.range_key, type = gsi.range_key_type }] : []
    )
  ])

  all_attributes = distinct(concat(local.base_attributes, local.gsi_attributes))
}

resource "aws_dynamodb_table" "this" {
  name         = "serverlessmate-${var.name}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = var.hash_key
  range_key    = var.range_key

  dynamic "attribute" {
    for_each = local.all_attributes
    content {
      name = attribute.value.name
      type = attribute.value.type
    }
  }

  dynamic "global_secondary_index" {
    for_each = var.global_secondary_indexes
    content {
      name            = global_secondary_index.value.name
      hash_key        = global_secondary_index.value.hash_key
      range_key       = global_secondary_index.value.range_key
      projection_type = global_secondary_index.value.projection_type
    }
  }

  stream_enabled   = var.stream_enabled
  stream_view_type = var.stream_enabled ? var.stream_view_type : null

  dynamic "ttl" {
    for_each = var.ttl_attribute_name != null ? [var.ttl_attribute_name] : []
    content {
      attribute_name = ttl.value
      enabled        = true
    }
  }

  tags = var.tags
}
