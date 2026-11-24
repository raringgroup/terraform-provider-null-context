terraform {
  required_version = ">= 1.8.0"
  required_providers {
    null-context = {
      source  = "raringgroup/null-context"
      version = "~> 0.1"
    }
  }
}

# Optional shared defaults, inherited by every module that uses the provider.
provider "null-context" {
  namespace   = "acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform" }
}
