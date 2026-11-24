data "null-context" "vpc" {
  name = "My VPC" # returned normalized: data.null-context.vpc.name == "myvpc"
}

# Chain from a parent: pass `self` (not `normalized_context`).
data "null-context" "sg" {
  context    = data.null-context.vpc.self
  attributes = ["sg"]
}

output "sg_id" {
  value = data.null-context.sg.id # acme-dev-app-myvpc-sg
}

output "sg_tags" {
  value = data.null-context.sg.tags
}
