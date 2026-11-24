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
