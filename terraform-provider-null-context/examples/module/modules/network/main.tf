terraform {
  required_version = ">= 1.8.0"
  required_providers {
    null-context = { source = "raringgroup/null-context" }
  }
}

variable "context" {
  type        = any
  default     = null # null => provider-level defaults apply
  description = "Parent context: the `self` of a data \"null-context\" or another module's `context` output."
}
variable "name" {
  type    = string
  default = null
}
variable "attributes" {
  type    = list(string)
  default = []
}
variable "tags" {
  type    = map(string)
  default = {}
}
variable "cidr_block" {
  type = string
}

# This module's own label. Inputs are merged with the parent and returned normalized.
data "null-context" "this" {
  context    = var.context
  name       = var.name
  attributes = var.attributes
  tags       = var.tags
}

# A sub-label derived from this one.
data "null-context" "private" {
  context    = data.null-context.this.self
  attributes = ["private"]
}

# terraform_data stands in for any resource (aws_vpc, azurerm_virtual_network, ...).
resource "terraform_data" "vpc" {
  input = {
    name = data.null-context.this.id
    tags = data.null-context.this.tags
    cidr = var.cidr_block
  }
}

resource "terraform_data" "private_subnet" {
  input = {
    name = data.null-context.private.id
    tags = data.null-context.private.tags
  }
}

output "id" { value = data.null-context.this.id }
output "private_subnet_id" { value = data.null-context.private.id }
output "tags" { value = data.null-context.this.tags }

# Hand this to children so they inherit namespace/environment/stage/tags.
output "context" { value = data.null-context.this.self }
