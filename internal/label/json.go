// ----------------------------------------------------------------------------
//  Copyright 2026 Raring Group LLC
//  Copyright 2020-2022 Cloud Posse, LLC
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

// ----------------------------------------------------------------------------
//  ./json.go
//  This file contains code ported from terraform-null-context and modified by Raring Group LLC
// ----------------------------------------------------------------------------

package label

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON reads a context/variables object the way Terraform treats
// `any`-typed values: a missing key and an explicit null both mean "null".
func (c *Context) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*c = Context{}
	has := func(key string) (json.RawMessage, bool) {
		v, ok := raw[key]
		if !ok || string(v) == "null" {
			return nil, false
		}
		return v, true
	}
	var firstErr error
	decode := func(key string, dst any) bool {
		v, ok := has(key)
		if !ok {
			return false
		}
		if err := json.Unmarshal(v, dst); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", key, err)
		}
		return true
	}
	for key, dst := range map[string]**string{
		"namespace": &c.Namespace, "tenant": &c.Tenant, "project": &c.Project, "region": &c.Region,
		"environment": &c.Environment, "stage": &c.Stage, "name": &c.Name, "delimiter": &c.Delimiter,
		"regex_replace_chars": &c.RegexReplaceChars, "label_key_case": &c.LabelKeyCase,
		"label_value_case": &c.LabelValueCase, "domain_name": &c.DomainName, "dns_name_format": &c.DNSNameFormat,
	} {
		var s string
		if decode(key, &s) {
			v := s
			*dst = &v
		}
	}
	var en bool
	if decode("enabled", &en) {
		c.Enabled = &en
	}
	var lim int64
	if decode("id_length_limit", &lim) {
		c.IDLengthLimit = &lim
	}
	decode("attributes", &c.Attributes)
	decode("tags", &c.Tags)
	decode("additional_tag_map", &c.AdditionalTagMap)
	decode("descriptor_formats", &c.DescriptorFormats)
	decode("labels_as_tags", &c.LabelsAsTags)
	var lo []string
	if decode("label_order", &lo) {
		c.LabelOrder = &lo
	}
	return firstErr
}

// ContextFromMap converts a decoded object (for example a module output or a
// Terraform dynamic value converted to Go) into a Context.
func ContextFromMap(m map[string]any) (Context, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return Context{}, err
	}
	var c Context
	if err := json.Unmarshal(b, &c); err != nil {
		return Context{}, err
	}
	return c, nil
}

func sp(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func labelOrderAny(p *[]string) any {
	if p == nil {
		return nil
	}
	return *p
}

// Self mirrors the module's `self` output (local.input).
func (r *Result) Self() map[string]any {
	in := r.input
	var en any
	if in.Enabled != nil {
		en = *in.Enabled
	}
	var lim any
	if in.IDLengthLimit != nil {
		lim = *in.IDLengthLimit
	}
	return map[string]any{
		"enabled": en, "namespace": sp(in.Namespace), "tenant": sp(in.Tenant), "project": sp(in.Project),
		"region": sp(in.Region), "environment": sp(in.Environment), "stage": sp(in.Stage), "name": sp(in.Name),
		"delimiter": sp(in.Delimiter), "attributes": in.Attributes, "tags": in.Tags,
		"additional_tag_map": in.AdditionalTagMap, "label_order": labelOrderAny(in.LabelOrder),
		"regex_replace_chars": sp(in.RegexReplaceChars), "id_length_limit": lim,
		"label_key_case": sp(in.LabelKeyCase), "label_value_case": sp(in.LabelValueCase),
		"descriptor_formats": in.DescriptorFormats, "labels_as_tags": in.LabelsAsTags,
		"domain_name": sp(in.DomainName), "dns_name_format": sp(in.DNSNameFormat),
	}
}

// Legacy mirrors the `legacy` output (Cloud Posse compatible: no project,
// region, domain_name or dns_name_format; label_order is a sorted set).
func (r *Result) Legacy() map[string]any {
	m := r.Self()
	for _, k := range legacyIncompatible {
		delete(m, k)
	}
	m["label_order"] = r.legacyOrder
	return m
}

// NormalizedContext mirrors the `normalized_context` output.
func (r *Result) NormalizedContext() map[string]any {
	return map[string]any{
		"enabled": r.Enabled, "namespace": r.Namespace, "tenant": r.Tenant, "environment": r.Environment,
		"stage": r.Stage, "name": r.Name, "delimiter": r.Delimiter, "attributes": r.Attributes, "tags": r.Tags,
		"additional_tag_map": r.AdditionalTagMap, "label_order": r.LabelOrder,
		"regex_replace_chars": r.RegexReplaceChars, "id_length_limit": r.IDLengthLimit,
		"label_key_case": r.LabelKeyCase, "label_value_case": r.LabelValueCase,
		"labels_as_tags": r.LabelsAsTags, "descriptor_formats": r.DescriptorFormats,
		"domain_name": r.DomainName, "dns_name_format": r.DNSNameFormat,
	}
}

// Outputs returns every module output keyed exactly like the module's outputs,
// including the enabled=false gating the module applies to some of them.
func (r *Result) Outputs() map[string]any {
	gs := func(s string) string {
		if r.Enabled {
			return s
		}
		return ""
	}
	attrs, tags := r.Attributes, r.Tags
	if !r.Enabled {
		attrs, tags = []string{}, map[string]string{}
	}
	return map[string]any{
		"id": gs(r.ID), "id_full": gs(r.IDFull), "dns_name": gs(r.DNSName), "enabled": r.Enabled,
		"namespace": gs(r.Namespace), "tenant": gs(r.Tenant), "environment": gs(r.Environment),
		"name": gs(r.Name), "stage": gs(r.Stage), "delimiter": gs(r.Delimiter), "attributes": attrs,
		"tags": tags, "additional_tag_map": r.AdditionalTagMap, "label_order": r.LabelOrder,
		"regex_replace_chars": r.RegexReplaceChars, "id_length_limit": r.IDLengthLimit,
		"tags_as_list_of_maps": r.TagsAsListOfMaps, "descriptors": r.Descriptors,
		"normalized_context": r.NormalizedContext(), "legacy": r.Legacy(), "self": r.Self(),
		"dns_name_format": gs(r.DNSNameFormat), "domain_name": gs(r.DomainName),
	}
}
