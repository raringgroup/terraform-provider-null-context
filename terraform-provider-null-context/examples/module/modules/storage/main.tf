terraform {
  required_version = ">= 1.8.0"
  required_providers {
    null-context = { source = "raringgroup/null-context" }
  }
}

variable "context" {
  type    = any
  default = null
}
variable "name" {
  type    = string
  default = null
}
variable "attributes" {
  type    = list(string)
  default = []
}

data "null-context" "this" {
  context    = var.context
  name       = var.name
  attributes = var.attributes
}

resource "terraform_data" "bucket" {
  input = {
    name = data.null-context.this.id
    tags = data.null-context.this.tags
  }
}

output "id" { value = data.null-context.this.id }
output "context" { value = data.null-context.this.self }
