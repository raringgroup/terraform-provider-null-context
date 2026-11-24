## ----------------------------------------------------------------------------
##  Copyright 2026 Raring Group LLC
##
##  Licensed under the Apache License, Version 2.0 (the "License");
##  you may not use this file except in compliance with the License.
##  You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
##  Unless required by applicable law or agreed to in writing, software
##  distributed under the License is distributed on an "AS IS" BASIS,
##  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
##  See the License for the specific language governing permissions and
##  limitations under the License.
## ----------------------------------------------------------------------------

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
  value = provider::null-context::label(null, { namespace = "acme", name = "baz", attributes = ["1", "2"] }).id
}
