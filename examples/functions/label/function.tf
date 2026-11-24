locals {
  root  = provider::null-context::label(null, { namespace = "acme", name = "a" })
  child = provider::null-context::label(local.root.self, { attributes = ["x"] })
}

output "child_id" {
  value = local.child.id # acme-a-x
}
