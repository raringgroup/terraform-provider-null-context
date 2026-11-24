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
