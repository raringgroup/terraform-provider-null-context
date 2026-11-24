// Copyright 2026 Raring Group LLC
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/raringgroup/terraform-provider-null-context/internal/provider"
)

var factories = map[string]func() (tfprotov6.ProviderServer, error){
	"null-context": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// These tests drive a real Terraform CLI. They are skipped when none is available.
func requireTerraform(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC_TERRAFORM_PATH") != "" {
		return
	}
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform CLI not found; set TF_ACC_TERRAFORM_PATH")
	}
}

const defaultsConfig = `
provider "null-context" {
  namespace   = "Acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform" }
}

data "null-context" "root" {}

data "null-context" "vpc" {
  context = data.null-context.root.self
  name    = "My VPC"
  tags    = { Tier = "net" }
}

data "null-context" "sg" {
  context    = data.null-context.vpc.self
  attributes = ["sg"]
}

data "null-context" "limited" {
  context         = data.null-context.root.self
  name            = "a-rather-long-component-name"
  id_length_limit = 20
}
`

func TestDataSource_ProviderDefaultsAndChaining(t *testing.T) {
	requireTerraform(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: defaultsConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.null-context.root", "id", "acme-dev-app"),
				resource.TestCheckResourceAttr("data.null-context.vpc", "id", "acme-dev-app-myvpc"),
				resource.TestCheckResourceAttr("data.null-context.vpc", "name", "myvpc"), // normalized, not "My VPC"
				resource.TestCheckResourceAttr("data.null-context.vpc", "tags.Owner", "platform"),
				resource.TestCheckResourceAttr("data.null-context.vpc", "tags.Tier", "net"),
				resource.TestCheckResourceAttr("data.null-context.vpc", "tags.Name", "acme-dev-app-myvpc"),
				resource.TestCheckResourceAttr("data.null-context.sg", "id", "acme-dev-app-myvpc-sg"),
				resource.TestCheckResourceAttr("data.null-context.limited", "id", "acme-dev-app-a-c1b3a"),
				resource.TestCheckResourceAttr("data.null-context.limited", "id_full", "acme-dev-app-a-rather-long-component-name"),
			),
		}},
	})
}

func TestDataSource_ExplicitContextReplacesProviderDefaults(t *testing.T) {
	requireTerraform(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: `
provider "null-context" { namespace = "ignored" }
data "null-context" "x" {
  context = { enabled = true, namespace = "7pi", environment = null, stage = null, name = null, delimiter = null,
              attributes = [], tags = {}, additional_tag_map = {}, regex_replace_chars = null, label_order = [],
              id_length_limit = null, label_key_case = null, label_value_case = null, descriptor_formats = {},
              domain_name = null, dns_name_format = null, labels_as_tags = ["unset"] }
  name = "svc"
}`,
			Check: resource.TestCheckResourceAttr("data.null-context.x", "id", "7pi-svc"),
		}},
	})
}

func TestDataSource_DisabledBlanksIDAndTags(t *testing.T) {
	requireTerraform(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: `
data "null-context" "x" {
  enabled   = false
  namespace = "a"
  name      = "b"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.null-context.x", "id", ""),
				resource.TestCheckResourceAttr("data.null-context.x", "tags.%", "0"),
			),
		}},
	})
}

func TestDataSource_InvalidInputIsAnError(t *testing.T) {
	requireTerraform(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config: `
data "null-context" "x" {
  name           = "b"
  label_key_case = "foo"
}`,
			ExpectError: regexp.MustCompile("Allowed values"),
		}},
	})
}

func TestFunction_Label(t *testing.T) {
	requireTerraform(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps: []resource.TestStep{{
			Config: `
locals {
  root  = provider::null-context::label(null, { namespace = "7pi", name = "a" })
  child = provider::null-context::label(local.root.self, { attributes = ["x"] })
}
output "root_id"  { value = local.root.id }
output "child_id" { value = local.child.id }
output "child_name_tag" { value = local.child.tags["Name"] }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckOutput("root_id", "7pi-a"),
				resource.TestCheckOutput("child_id", "7pi-a-x"),
				resource.TestCheckOutput("child_name_tag", "7pi-a-x"),
			),
		}},
	})
}
