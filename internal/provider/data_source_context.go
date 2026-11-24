// Copyright 2026 Raring Group LLC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/raringgroup/terraform-provider-null-context/internal/label"
)

var (
	_ datasource.DataSource              = (*contextDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*contextDataSource)(nil)
)

// NewContextDataSource returns the `null-context` data source.
func NewContextDataSource() datasource.DataSource { return &contextDataSource{} }

type contextDataSource struct{ defaults label.Context }

// dynamicOutputs are computed attributes typed `any`.
var dynamicOutputs = map[string]bool{"normalized_context": true, "legacy": true, "self": true}

func (d *contextDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	// The data source type is exactly the provider name: `data "null-context" "x" {}`.
	resp.TypeName = req.ProviderTypeName
}

func (d *contextDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	// Optional+Computed: input name == module output name; the computed value is the normalized/merged result.
	optStr := func(desc string) dschema.Attribute {
		return dschema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: desc}
	}
	optList := func(desc string) dschema.Attribute {
		return dschema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, MarkdownDescription: desc}
	}
	optMap := func(desc string) dschema.Attribute {
		return dschema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType, MarkdownDescription: desc}
	}
	inStr := func(desc string) dschema.Attribute {
		return dschema.StringAttribute{Optional: true, MarkdownDescription: desc}
	}
	outStr := func(desc string) dschema.Attribute {
		return dschema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	outDyn := func(desc string) dschema.Attribute {
		return dschema.DynamicAttribute{Computed: true, MarkdownDescription: desc}
	}
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Computes `id` and `tags` for one resource from the provider defaults and/or a parent `context`. " +
			"Inputs that share a name with an output (for example `name` or `tags`) are merged with the parent and returned normalized.",
		Attributes: map[string]dschema.Attribute{
			"context": dschema.DynamicAttribute{Optional: true, MarkdownDescription: "Parent context: the `self` or `normalized_context` of another `context` data source (or the old module). When set, provider-level defaults are ignored."},
			// inputs that are also outputs
			"enabled":             dschema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "False blanks `id`, `tags` and the normalized strings."},
			"namespace":           optStr("ID element; returned normalized."),
			"tenant":              optStr("ID element; returned normalized."),
			"environment":         optStr("ID element; returned normalized."),
			"stage":               optStr("ID element; returned normalized."),
			"name":                optStr("ID element; returned normalized."),
			"delimiter":           optStr("Delimiter; returned with the default applied."),
			"attributes":          optList("Attributes appended to the parent's; returned normalized."),
			"tags":                optMap("Extra tags merged over the parent's; returned as the final tag map."),
			"additional_tag_map":  optMap("Merged with the parent's; returned merged."),
			"label_order":         optList("Returned with the default applied."),
			"regex_replace_chars": optStr("Returned with the default applied."),
			"id_length_limit":     dschema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "Returned with the default applied."},
			"domain_name":         optStr("Domain used by `dns_name`."),
			"dns_name_format":     optStr("Format of `dns_name`."),
			// input-only
			"project":          inStr("ID element (not in the default `label_order`)."),
			"region":           inStr("ID element (not in the default `label_order`)."),
			"label_key_case":   inStr("`lower`, `title` or `upper`."),
			"label_value_case": inStr("`lower`, `title`, `upper` or `none`."),
			"descriptor_formats": dschema.MapAttribute{Optional: true, ElementType: descriptorObjType,
				MarkdownDescription: "Map of `{format, labels}` objects rendered into `descriptors`."},
			"labels_as_tags": dschema.SetAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "ID elements to include as tags. First setting in a chain wins."},
			// outputs
			"id":                   outStr("Disambiguated ID, limited to `id_length_limit`."),
			"id_full":              outStr("ID without the length limit."),
			"dns_name":             outStr("DNS name built from `dns_name_format`."),
			"tags_as_list_of_maps": dschema.ListAttribute{Computed: true, ElementType: types.MapType{ElemType: types.StringType}, MarkdownDescription: "One map per tag with `key`, `value` and `additional_tag_map`."},
			"descriptors":          dschema.MapAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Rendered `descriptor_formats`."},
			"normalized_context":   outDyn("Normalized context to pass to child contexts (drops `project` and `region`, like the module)."),
			"legacy":               outDyn("Cloud Posse compatible context."),
			"self":                 outDyn("Merged raw inputs; the recommended value to pass as a child's `context`."),
		},
	}
}

func (d *contextDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		d.defaults = label.DefaultContext()
		return
	}
	c, ok := req.ProviderData.(*label.Context)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.defaults = *c
}

func (d *contextDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	g, err := tfToGo(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid configuration", err.Error())
		return
	}
	cfg := g.(map[string]any)
	parentRaw := cfg["context"]
	delete(cfg, "context")
	vars, err := label.ContextFromMap(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Invalid configuration", err.Error())
		return
	}
	parent := d.defaults
	if parentRaw != nil {
		if parent, err = contextFromAny(parentRaw); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("context"), "Invalid context", err.Error())
			return
		}
	}
	res, err := label.Compute(parent, vars)
	if err != nil {
		resp.Diagnostics.AddError("Cannot compute context", err.Error())
		return
	}
	for name, v := range res.Outputs() {
		av, err := toAttr(ctx, v)
		if err != nil {
			resp.Diagnostics.AddError("Cannot convert output "+name, err.Error())
			return
		}
		if dynamicOutputs[name] {
			av = types.DynamicValue(av)
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), av)...)
	}
	// Pass input-only attributes through unchanged.
	var s types.String
	for _, n := range []string{"project", "region", "label_key_case", "label_value_case"} {
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(n), &s)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(n), s)...)
	}
	var df types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("descriptor_formats"), &df)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("descriptor_formats"), df)...)
	var lat types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("labels_as_tags"), &lat)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("labels_as_tags"), lat)...)
	var pc types.Dynamic
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("context"), &pc)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("context"), pc)...)
}
