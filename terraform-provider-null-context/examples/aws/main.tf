# Resource example: use the provider for names and tags on real AWS resources.
# `terraform validate` works without credentials; apply needs AWS credentials.

terraform {
  required_version = ">= 1.8.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    null-context = {
      source  = "raringgroup/null-context"
      version = "~> 0.1"
    }
  }
}

# Shared defaults for every data "null-context" below.
provider "null-context" {
  namespace   = "acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform" }
}

provider "aws" {
  region = "us-east-1"
}

# The application's label: acme-dev-app-orders
data "null-context" "app" {
  name = "orders"

  # Optional: extra formatted strings, e.g. a name that must not contain the stage.
  descriptor_formats = {
    account_name = { format = "%v-%v", labels = ["namespace", "name"] } # acme-orders
  }
}

# Child labels derived from it.
data "null-context" "assets" {
  context    = data.null-context.app.self
  attributes = ["assets"]
}

data "null-context" "queue" {
  context    = data.null-context.app.self
  attributes = ["jobs"]
  tags       = { Retention = "7d" } # merged over the parent's tags
}

# Truncated id, e.g. for resources with name length limits.
data "null-context" "short" {
  context         = data.null-context.app.self
  attributes      = ["a-very-long-attribute-name"]
  id_length_limit = 24 # readable prefix + hash, never longer than 24
}

resource "aws_s3_bucket" "assets" {
  bucket = data.null-context.assets.id # acme-dev-app-orders-assets
  tags   = data.null-context.assets.tags
}

resource "aws_sqs_queue" "jobs" {
  name = data.null-context.queue.id
  tags = data.null-context.queue.tags
}

resource "aws_iam_role" "short" {
  name = data.null-context.short.id # <= 24 characters
  tags = data.null-context.short.tags
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "sts:AssumeRole", Principal = { Service = "lambda.amazonaws.com" } }]
  })
}

# Resources whose tags are a list of {key, value, ...} objects (e.g. autoscaling groups):
#   dynamic "tag" {
#     for_each = data.null-context.app.tags_as_list_of_maps
#     content {
#       key                 = tag.value.key
#       value               = tag.value.value
#       propagate_at_launch = true
#     }
#   }

output "bucket" { value = aws_s3_bucket.assets.bucket }
output "short_role" { value = aws_iam_role.short.name }
output "account_name" { value = data.null-context.app.descriptors["account_name"] }
