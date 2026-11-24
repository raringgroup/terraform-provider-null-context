locals {
  root  = provider::null-context::label(null, { namespace = "7pi", name = "a" })
  child = provider::null-context::label(local.root.self, { attributes = ["x"] })
}

output "child_id" {
  value = local.child.id # 7pi-a-x
}
