# Dev-override or install the provider first (see README.md), then: terraform apply
terraform {
  required_providers {
    null-context = { source = "raringgroup/null-context" }
  }
}

# Shared defaults for every `data "null-context"` that does not set its own `context`.
provider "null-context" {
  namespace   = "acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform" }
}

data "null-context" "root" {}

data "null-context" "vpc" {
  context = data.null-context.root.self # chain: pass the parent's `self`
  name    = "vpc"
}

data "null-context" "sg" {
  context    = data.null-context.vpc.self
  attributes = ["sg"]
}

output "ids" {
  value = {
    root = data.null-context.root.id # acme-dev-app
    vpc  = data.null-context.vpc.id  # acme-dev-app-vpc
    sg   = data.null-context.sg.id   # acme-dev-app-vpc-sg
  }
}

output "vpc_tags" { value = data.null-context.vpc.tags }

# Terraform >= 1.8 / OpenTofu >= 1.7: the same engine as a pure function.
output "from_function" {
  value = provider::null-context::label(null, { namespace = "7pi", name = "baz", attributes = ["1", "2"] }).id
}
