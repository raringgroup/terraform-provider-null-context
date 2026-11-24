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

# Module example: one provider configuration at the root, contexts chained into child modules.
# Run:  TF_CLI_CONFIG_FILE=<your dev_overrides file> terraform apply   (see the repo README)

terraform {
  required_version = ">= 1.8.0"
  required_providers {
    null-context = {
      source  = "raringgroup/null-context"
      version = "~> 0.1"
    }
  }
}

# Provider configuration = the DEFAULT parent context for every data "null-context" that sets no `context`.
provider "null-context" {
  namespace   = "acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform", CostCenter = "1234" }
}

# Root context (no inputs): acme-dev-app
data "null-context" "root" {}

module "network" {
  source     = "./modules/network"
  context    = data.null-context.root.self # chain: pass `self` of the parent
  name       = "vpc"
  cidr_block = "10.0.0.0/16"
}

module "logs" {
  source     = "./modules/storage"
  context    = module.network.context # chain from another module's `context` output
  name       = "logs"
  attributes = ["archive"]
}

# No context passed: the child falls back to the provider defaults above.
module "scratch" {
  source = "./modules/storage"
  name   = "scratch"
}

output "ids" {
  value = {
    root    = data.null-context.root.id        # acme-dev-app
    network = module.network.id                # acme-dev-app-vpc
    private = module.network.private_subnet_id # acme-dev-app-vpc-private
    logs    = module.logs.id                   # acme-dev-app-logs-archive
    scratch = module.scratch.id                # acme-dev-app-scratch
  }
}

output "network_tags" {
  value = module.network.tags
}
