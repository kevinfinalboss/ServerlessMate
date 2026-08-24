resource "aws_resourcegroups_group" "this" {
  name        = var.name
  description = var.description

  resource_query {
    query = jsonencode({
      ResourceTypeFilters = ["AWS::AllSupported"]
      TagFilters = [
        {
          Key    = var.tag_key
          Values = [var.tag_value]
        },
      ]
    })
  }

  tags = var.tags
}
