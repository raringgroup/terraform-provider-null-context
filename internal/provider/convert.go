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
	"fmt"
	"math/big"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/raringgroup/terraform-provider-null-context/internal/label"
)

// tfToGo converts a Terraform value into plain Go values (nil, string, bool,
// int64/float64, []any, map[string]any) so it can be treated like JSON.
func tfToGo(v tftypes.Value) (any, error) {
	if !v.IsKnown() {
		return nil, fmt.Errorf("value is unknown")
	}
	if v.IsNull() {
		return nil, nil
	}
	t := v.Type()
	switch {
	case t.Is(tftypes.String):
		var s string
		if err := v.As(&s); err != nil {
			return nil, err
		}
		return s, nil
	case t.Is(tftypes.Bool):
		var b bool
		if err := v.As(&b); err != nil {
			return nil, err
		}
		return b, nil
	case t.Is(tftypes.Number):
		f := new(big.Float)
		if err := v.As(&f); err != nil {
			return nil, err
		}
		if i, acc := f.Int64(); acc == big.Exact {
			return i, nil
		}
		fl, _ := f.Float64()
		return fl, nil
	case t.Is(tftypes.List{}), t.Is(tftypes.Set{}), t.Is(tftypes.Tuple{}):
		var vs []tftypes.Value
		if err := v.As(&vs); err != nil {
			return nil, err
		}
		out := make([]any, 0, len(vs))
		for _, e := range vs {
			g, err := tfToGo(e)
			if err != nil {
				return nil, err
			}
			out = append(out, g)
		}
		return out, nil
	case t.Is(tftypes.Map{}), t.Is(tftypes.Object{}):
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(m))
		for k, e := range m {
			g, err := tfToGo(e)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = g
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported type %s", t)
}

func dynToGo(ctx context.Context, d types.Dynamic) (any, error) {
	if d.IsNull() || d.IsUnderlyingValueNull() {
		return nil, nil
	}
	if d.IsUnknown() || d.IsUnderlyingValueUnknown() {
		return nil, fmt.Errorf("value is unknown")
	}
	tv, err := d.UnderlyingValue().ToTerraformValue(ctx)
	if err != nil {
		return nil, err
	}
	return tfToGo(tv)
}

func contextFromAny(v any) (label.Context, error) {
	if v == nil {
		return label.Context{}, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return label.Context{}, fmt.Errorf("expected an object, got %T", v)
	}
	return label.ContextFromMap(m)
}

var descriptorObjType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"format": types.StringType,
	"labels": types.ListType{ElemType: types.StringType},
}}

// toAttr converts the Go values produced by label.Result.Outputs() into
// framework attr.Values.
func toAttr(ctx context.Context, v any) (attr.Value, error) {
	switch x := v.(type) {
	case nil:
		return types.StringNull(), nil
	case string:
		return types.StringValue(x), nil
	case bool:
		return types.BoolValue(x), nil
	case int64:
		return types.Int64Value(x), nil
	case []string:
		if x == nil {
			x = []string{}
		}
		l, d := types.ListValueFrom(ctx, types.StringType, x)
		return l, diagErr(d)
	case map[string]string:
		if x == nil {
			x = map[string]string{}
		}
		m, d := types.MapValueFrom(ctx, types.StringType, x)
		return m, diagErr(d)
	case []map[string]string:
		if x == nil {
			x = []map[string]string{}
		}
		l, d := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, x)
		return l, diagErr(d)
	case map[string]label.DescriptorFormat:
		elems := make(map[string]attr.Value, len(x))
		for k, df := range x {
			labels, err := toAttr(ctx, append([]string{}, df.Labels...))
			if err != nil {
				return nil, err
			}
			o, d := types.ObjectValue(descriptorObjType.AttrTypes, map[string]attr.Value{
				"format": types.StringValue(df.Format), "labels": labels})
			if err := diagErr(d); err != nil {
				return nil, err
			}
			elems[k] = o
		}
		m, d := types.MapValue(descriptorObjType, elems)
		return m, diagErr(d)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ts := make(map[string]attr.Type, len(x))
		vs := make(map[string]attr.Value, len(x))
		for _, k := range keys {
			av, err := toAttr(ctx, x[k])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			ts[k], vs[k] = av.Type(ctx), av
		}
		o, d := types.ObjectValue(ts, vs)
		return o, diagErr(d)
	}
	return nil, fmt.Errorf("cannot convert %T", v)
}
