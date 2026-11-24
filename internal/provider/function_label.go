// ----------------------------------------------------------------------------
//  Copyright 2026 Raring Group LLC
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
// ----------------------------------------------------------------------------

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/raringgroup/terraform-provider-null-context/internal/label"
)

var _ function.Function = (*labelFunction)(nil)

// NewLabelFunction returns `provider::null-context::label(context, vars)`.
func NewLabelFunction() function.Function { return &labelFunction{} }

type labelFunction struct{}

func (f *labelFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "label"
}

func (f *labelFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Compute an id, tags and child context",
		MarkdownDescription: "Pure equivalent of the null-context module. `context` is a parent (the `self` or " +
			"`normalized_context` of an earlier call, or `null` for the module defaults) and `vars` holds this call's " +
			"inputs (`name`, `attributes`, `tags`, ...). Returns an object with every module output.",
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "context", AllowNullValue: true, Description: "Parent context or null."},
			function.DynamicParameter{Name: "vars", AllowNullValue: true, Description: "Inputs of this call or null."},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *labelFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var c, v types.Dynamic
	if resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &c, &v)); resp.Error != nil {
		return
	}
	cg, err := dynToGo(ctx, c)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "context: "+err.Error())
		return
	}
	vg, err := dynToGo(ctx, v)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "vars: "+err.Error())
		return
	}
	parent := label.DefaultContext()
	if cg != nil {
		if parent, err = contextFromAny(cg); err != nil {
			resp.Error = function.NewArgumentFuncError(0, err.Error())
			return
		}
	}
	vars, err := contextFromAny(vg)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}
	res, err := label.Compute(parent, vars)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	obj, err := toAttr(ctx, res.Outputs())
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, types.DynamicValue(obj)))
}
