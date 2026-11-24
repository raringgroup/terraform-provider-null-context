// Copyright 2026 Raring Group LLC
// Portions Copyright 2020-2022 Cloud Posse, LLC (derived from terraform-null-label)
// SPDX-License-Identifier: Apache-2.0

package label

// Overlay returns c with every non-null field of o applied on top. It is used
// to turn the provider-level defaults into the parent context of a data source.
func (c Context) Overlay(o Context) Context {
	out := c
	if o.Enabled != nil {
		out.Enabled = o.Enabled
	}
	for _, f := range []struct {
		dst **string
		src *string
	}{
		{&out.Namespace, o.Namespace}, {&out.Tenant, o.Tenant}, {&out.Project, o.Project}, {&out.Region, o.Region},
		{&out.Environment, o.Environment}, {&out.Stage, o.Stage}, {&out.Name, o.Name}, {&out.Delimiter, o.Delimiter},
		{&out.RegexReplaceChars, o.RegexReplaceChars}, {&out.LabelKeyCase, o.LabelKeyCase},
		{&out.LabelValueCase, o.LabelValueCase}, {&out.DomainName, o.DomainName}, {&out.DNSNameFormat, o.DNSNameFormat},
	} {
		if f.src != nil {
			*f.dst = f.src
		}
	}
	if o.Attributes != nil {
		out.Attributes = o.Attributes
	}
	if o.Tags != nil {
		out.Tags = o.Tags
	}
	if o.AdditionalTagMap != nil {
		out.AdditionalTagMap = o.AdditionalTagMap
	}
	if o.LabelOrder != nil {
		out.LabelOrder = o.LabelOrder
	}
	if o.IDLengthLimit != nil {
		out.IDLengthLimit = o.IDLengthLimit
	}
	if o.DescriptorFormats != nil {
		out.DescriptorFormats = o.DescriptorFormats
	}
	if o.LabelsAsTags != nil {
		out.LabelsAsTags = o.LabelsAsTags
	}
	return out
}
