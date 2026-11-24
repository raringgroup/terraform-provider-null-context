// Copyright 2026 Raring Group LLC
// SPDX-License-Identifier: Apache-2.0

// Package provider exposes the label engine as a Terraform provider named
// "null-context": provider-level defaults, a `null-context` data source and a
// `provider::null-context::label()` function.
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/raringgroup/terraform-provider-null-context/internal/label"
)

var (
	_ provider.Provider              = (*contextProvider)(nil)
	_ provider.ProviderWithFunctions = (*contextProvider)(nil)
)

type contextProvider struct{ version string }

// New returns the provider factory used by main and by tests.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &contextProvider{version: version} }
}

func diagErr(d diag.Diagnostics) error {
	if d.HasError() {
		return fmt.Errorf("%s: %s", d.Errors()[0].Summary(), d.Errors()[0].Detail())
	}
	return nil
}

func (p *contextProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "null-context"
	resp.Version = p.version
}

func (p *contextProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	str := func(desc string) pschema.Attribute {
		return pschema.StringAttribute{Optional: true, MarkdownDescription: desc}
	}
	list := func(desc string) pschema.Attribute {
		return pschema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: desc}
	}
	mp := func(desc string) pschema.Attribute {
		return pschema.MapAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: desc}
	}
	resp.Schema = pschema.Schema{
		MarkdownDescription: "Computes consistent resource names (`id`) and tags from a shared context. " +
			"Settings here are the *default parent context* of every `null-context` data source that does not set its own `context`.",
		Attributes: map[string]pschema.Attribute{
			"enabled":             pschema.BoolAttribute{Optional: true, MarkdownDescription: "Set to false to blank the generated id and tags."},
			"namespace":           str("ID element: organisation abbreviation."),
			"tenant":              str("ID element (not in the default `label_order`)."),
			"project":             str("ID element (not in the default `label_order`)."),
			"region":              str("ID element (not in the default `label_order`)."),
			"environment":         str("ID element: e.g. `dev`, `prod`."),
			"stage":               str("ID element: e.g. `app`, `build`."),
			"name":                str("ID element: component or solution name."),
			"delimiter":           str("Delimiter between ID elements. Default `-`."),
			"attributes":          list("Additional ID elements appended after `name`."),
			"tags":                mp("Additional tags."),
			"additional_tag_map":  mp("Extra key/values added to every entry of `tags_as_list_of_maps`."),
			"label_order":         list("Order of ID elements in `id`."),
			"regex_replace_chars": str("Regex (wrapped in `/`) of characters removed from ID elements."),
			"id_length_limit":     pschema.Int64Attribute{Optional: true, MarkdownDescription: "Limit `id` to this many characters (>= 6) or 0 for unlimited."},
			"label_key_case":      str("Case of generated tag keys: `lower`, `title` or `upper`."),
			"label_value_case":    str("Case of ID elements: `lower`, `title`, `upper` or `none`."),
			"descriptor_formats": pschema.MapAttribute{Optional: true, ElementType: descriptorObjType,
				MarkdownDescription: "Map of `{format, labels}` objects rendered into `descriptors`."},
			"labels_as_tags":  pschema.SetAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "ID elements to include as tags. Default: all."},
			"domain_name":     str("Domain used by `dns_name`."),
			"dns_name_format": str("Format of `dns_name`, e.g. `${name}.${domain_name}`."),
		},
	}
}

func (p *contextProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	g, err := tfToGo(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	cfg, err := contextFromAny(g)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	defaults := label.DefaultContext().Overlay(cfg)
	resp.DataSourceData = &defaults
}

func (p *contextProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{NewContextDataSource}
}

func (p *contextProvider) Resources(context.Context) []func() resource.Resource { return nil }

func (p *contextProvider) Functions(context.Context) []func() function.Function {
	return []func() function.Function{NewLabelFunction}
}

var _ = attr.Value(nil)
