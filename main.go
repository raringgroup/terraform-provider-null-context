// Copyright 2026 Raring Group LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/raringgroup/terraform-provider-null-context/internal/provider"
)

// version is set by goreleaser at release time.
var version = "dev"

// address is the registry address; change it here if the provider moves registries.
const address = "registry.terraform.io/raringgroup/null-context"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: address, Debug: debug,
	}); err != nil {
		log.Fatal(err.Error())
	}
}
