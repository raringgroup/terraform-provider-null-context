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
//  ./label.go
//  This file contains code ported from terraform-null-context and modified by Raring Group LLC
// ----------------------------------------------------------------------------

// Package label is a faithful Go port of the null-context Terraform module
// (github.com/raringgroup/terraform-null-context). It has no Terraform dependencies so
// it can be tested against golden vectors generated from the real module.
package label

import (
	"crypto/md5" //nolint:gosec // md5 is required for ID-hash parity with the module
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	defaultRegex   = "/[^-a-zA-Z0-9]/"
	defaultDelim   = "-"
	defaultKeyCase = "title"
	defaultValCase = "lower"
	idHashLength   = 5
	hashSuffix     = "qrstuvwxyz"
	defaultDNSFmt  = "${name}.${domain_name}"
)

var defaultOrder = []string{"namespace", "environment", "stage", "name", "attributes"}

// legacyIncompatible are removed from label_order in the `legacy` output.
var legacyIncompatible = []string{"project", "region", "domain_name", "dns_name_format"}

// DescriptorFormat is one entry of descriptor_formats.
type DescriptorFormat struct {
	Format string   `json:"format"`
	Labels []string `json:"labels"`
}

// Context mirrors both var.context and the individual module variables
// (they share the same field names). nil pointers / nil slices mean null.
// For LabelsAsTags, nil (or a list containing "unset") means "not set yet".
type Context struct {
	Enabled           *bool
	Namespace         *string
	Tenant            *string
	Project           *string
	Region            *string
	Environment       *string
	Stage             *string
	Name              *string
	Delimiter         *string
	Attributes        []string
	Tags              map[string]string
	AdditionalTagMap  map[string]string
	LabelOrder        *[]string
	RegexReplaceChars *string
	IDLengthLimit     *int64
	LabelKeyCase      *string
	LabelValueCase    *string
	DescriptorFormats map[string]DescriptorFormat
	LabelsAsTags      []string
	DomainName        *string
	DNSNameFormat     *string
}

// DefaultContext is the default value of var.context in the module.
func DefaultContext() Context {
	t := true
	return Context{
		Enabled:           &t,
		Attributes:        []string{},
		Tags:              map[string]string{},
		AdditionalTagMap:  map[string]string{},
		LabelOrder:        &[]string{},
		DescriptorFormats: map[string]DescriptorFormat{},
		LabelsAsTags:      []string{"unset"},
	}
}

// Result holds every value the module computes.
type Result struct {
	Enabled           bool
	ID, IDFull        string
	DNSName           string
	Namespace         string
	Tenant            string
	Project           string
	Region            string
	Environment       string
	Stage             string
	Name              string
	Delimiter         string
	Attributes        []string
	Tags              map[string]string
	AdditionalTagMap  map[string]string
	LabelOrder        []string
	RegexReplaceChars string
	IDLengthLimit     int64
	LabelKeyCase      string
	LabelValueCase    string
	LabelsAsTags      []string
	TagsAsListOfMaps  []map[string]string
	Descriptors       map[string]string
	DescriptorFormats map[string]DescriptorFormat
	DomainName        string
	DNSNameFormat     string

	input       Context // module `self` output (local.input)
	legacyOrder []string
}

func pickS(v, c *string) *string {
	if v != nil {
		return v
	}
	return c
}

func sval(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Compute evaluates the module for a parent context (var.context) and the
// module's own variables (vars). Pass DefaultContext() when there is no parent.
func Compute(parent, vars Context) (*Result, error) {
	parent, vars = normalizeContext(parent), normalizeContext(vars) // cty normalizes every string to NFC
	// ---- local.input: variables win over the context, except merged fields ----
	in := Context{
		Enabled:           vars.Enabled,
		Namespace:         pickS(vars.Namespace, parent.Namespace),
		Tenant:            pickS(vars.Tenant, parent.Tenant),
		Project:           pickS(vars.Project, parent.Project),
		Region:            pickS(vars.Region, parent.Region),
		Environment:       pickS(vars.Environment, parent.Environment),
		Stage:             pickS(vars.Stage, parent.Stage),
		Name:              pickS(vars.Name, parent.Name),
		Delimiter:         pickS(vars.Delimiter, parent.Delimiter),
		Attributes:        distinct(compact(append(append([]string{}, parent.Attributes...), vars.Attributes...))),
		Tags:              mergeMaps(parent.Tags, vars.Tags),
		AdditionalTagMap:  mergeMaps(parent.AdditionalTagMap, vars.AdditionalTagMap),
		LabelOrder:        parent.LabelOrder,
		RegexReplaceChars: pickS(vars.RegexReplaceChars, parent.RegexReplaceChars),
		IDLengthLimit:     vars.IDLengthLimit,
		LabelKeyCase:      pickS(vars.LabelKeyCase, parent.LabelKeyCase),
		LabelValueCase:    pickS(vars.LabelValueCase, parent.LabelValueCase),
		DescriptorFormats: map[string]DescriptorFormat{},
		DomainName:        pickS(vars.DomainName, parent.DomainName),
		DNSNameFormat:     pickS(vars.DNSNameFormat, parent.DNSNameFormat),
	}
	if in.Enabled == nil {
		in.Enabled = parent.Enabled
	}
	if vars.LabelOrder != nil {
		in.LabelOrder = vars.LabelOrder
	}
	if vars.IDLengthLimit == nil {
		in.IDLengthLimit = parent.IDLengthLimit
	}
	for k, v := range parent.DescriptorFormats {
		in.DescriptorFormats[k] = v
	}
	for k, v := range vars.DescriptorFormats {
		in.DescriptorFormats[k] = v
	}
	// labels_as_tags: the first setting wins and can never be overridden later.
	parentUnset := parent.LabelsAsTags == nil || contains(parent.LabelsAsTags, "unset")
	if parentUnset {
		vl := vars.LabelsAsTags
		if vl == nil {
			vl = []string{"default"}
		}
		in.LabelsAsTags = sortedUnique(vl)
	} else {
		in.LabelsAsTags = sortedUnique(parent.LabelsAsTags)
	}

	r := &Result{input: in}
	r.Enabled = in.Enabled == nil || *in.Enabled

	regex := defaultRegex
	if in.RegexReplaceChars != nil && *in.RegexReplaceChars != "" { // coalesce() skips ""
		regex = *in.RegexReplaceChars
	}
	r.RegexReplaceChars = regex
	r.LabelValueCase = defaultValCase
	if in.LabelValueCase != nil {
		r.LabelValueCase = *in.LabelValueCase
	}
	r.LabelKeyCase = defaultKeyCase
	if in.LabelKeyCase != nil {
		r.LabelKeyCase = *in.LabelKeyCase
	}
	// The module validates var.context and each variable separately (not the merged value).
	for _, c := range []Context{parent, vars} {
		if err := validateCases(c.LabelKeyCase, c.LabelValueCase); err != nil {
			return nil, err
		}
	}
	if err := validateLimit(vars.IDLengthLimit); err != nil {
		return nil, err
	}

	// ---- normalize labels ----
	norm := func(p *string) (string, error) {
		if p == nil {
			return "", nil
		}
		s, err := tfReplace(*p, regex, "")
		if err != nil {
			return "", err
		}
		return applyCase(s, r.LabelValueCase), nil
	}
	var err error
	if r.Namespace, err = norm(in.Namespace); err != nil {
		return nil, err
	}
	if r.Tenant, err = norm(in.Tenant); err != nil {
		return nil, err
	}
	if r.Environment, err = norm(in.Environment); err != nil {
		return nil, err
	}
	if r.Stage, err = norm(in.Stage); err != nil {
		return nil, err
	}
	if r.Project, err = norm(in.Project); err != nil {
		return nil, err
	}
	if r.Region, err = norm(in.Region); err != nil {
		return nil, err
	}
	if r.Name, err = norm(in.Name); err != nil {
		return nil, err
	}
	attrs := []string{}
	for _, a := range in.Attributes {
		s, err := tfReplace(a, regex, "")
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, s)
	}
	attrs = distinct(compact(attrs))
	for i, a := range attrs {
		attrs[i] = applyCase(a, r.LabelValueCase)
	}
	r.Attributes = distinct(compact(attrs))

	r.Delimiter = defaultDelim
	if in.Delimiter != nil {
		r.Delimiter = *in.Delimiter
	}
	r.LabelOrder = append([]string{}, defaultOrder...)
	if in.LabelOrder != nil && len(*in.LabelOrder) > 0 { // coalescelist()
		r.LabelOrder = append([]string{}, *in.LabelOrder...)
	}
	if in.IDLengthLimit != nil {
		r.IDLengthLimit = *in.IDLengthLimit
	}
	r.DomainName = sval(in.DomainName)
	r.DNSNameFormat = defaultDNSFmt
	if in.DNSNameFormat != nil {
		r.DNSNameFormat = *in.DNSNameFormat
	}
	r.AdditionalTagMap = mergeMaps(parent.AdditionalTagMap, vars.AdditionalTagMap)
	r.DescriptorFormats = in.DescriptorFormats

	// ---- id ----
	idCtx := map[string]string{
		"namespace": r.Namespace, "tenant": r.Tenant, "project": r.Project, "region": r.Region,
		"environment": r.Environment, "stage": r.Stage, "name": r.Name,
		"attributes": nfc(strings.Join(r.Attributes, r.Delimiter)),
	}
	labels := []string{}
	for _, l := range r.LabelOrder {
		v, ok := idCtx[l]
		if !ok {
			return nil, fmt.Errorf("Invalid index: label_order contains unknown label %q", l)
		}
		if len(v) > 0 {
			labels = append(labels, v)
		}
	}
	r.IDFull = nfc(strings.Join(labels, r.Delimiter))
	r.ID = r.IDFull
	if r.IDLengthLimit != 0 && tfStrlen(r.IDFull) > int(r.IDLengthLimit) {
		limit := int(r.IDLengthLimit)
		truncLimit := limit - (idHashLength + tfStrlen(r.Delimiter))
		truncated := ""
		if truncLimit > 0 {
			truncated = nfc(strings.TrimSuffix(tfSubstr(r.IDFull, truncLimit), r.Delimiter) + r.Delimiter)
		}
		sum := md5.Sum([]byte(r.IDFull)) //nolint:gosec
		hash := hex.EncodeToString(sum[:]) + hashSuffix
		switch r.LabelValueCase {
		case "title":
			hash = tfTitle(hash)
		case "upper":
			hash = strings.ToUpper(hash)
		case "lower":
			hash = strings.ToLower(hash)
		}
		if hash, err = tfReplace(hash, regex, ""); err != nil {
			return nil, err
		}
		r.ID = tfSubstr(nfc(truncated+hash), limit)
	}

	// ---- tags ----
	tagsCtx := map[string]string{
		"namespace": r.Namespace, "tenant": r.Tenant, "project": r.Project, "region": r.Region,
		"environment": r.Environment, "stage": r.Stage, "name": r.ID, "attributes": idCtx["attributes"],
	}
	lat := in.LabelsAsTags
	if contains(lat, "default") {
		lat = sortedKeys(tagsCtx)
	}
	r.LabelsAsTags = lat
	generated := map[string]string{}
	for _, l := range sortedKeys(tagsCtx) {
		if !contains(lat, l) || len(tagsCtx[l]) == 0 {
			continue
		}
		k := tfTitle(strings.ToLower(l))
		switch r.LabelKeyCase {
		case "upper":
			k = strings.ToUpper(l)
		case "lower":
			k = strings.ToLower(l)
		}
		generated[k] = tagsCtx[l]
	}
	r.Tags = mergeMaps(generated, in.Tags)
	r.TagsAsListOfMaps = []map[string]string{}
	for _, k := range sortedKeys(r.Tags) {
		m := map[string]string{"key": k, "value": r.Tags[k]}
		for ak, av := range r.AdditionalTagMap {
			m[ak] = av
		}
		r.TagsAsListOfMaps = append(r.TagsAsListOfMaps, m)
	}

	// ---- descriptors ----
	r.Descriptors = map[string]string{}
	for name, d := range r.DescriptorFormats {
		args := make([]string, 0, len(d.Labels))
		for _, l := range d.Labels {
			v, ok := idCtx[l]
			if !ok {
				return nil, fmt.Errorf("Invalid index: descriptor %q uses unknown label %q", name, l)
			}
			args = append(args, v)
		}
		s, err := tfFormat(d.Format, args)
		if err != nil {
			return nil, err
		}
		r.Descriptors[name] = s
	}

	// ---- dns_name (placeholders are replaced sequentially, in this exact order) ----
	dns := r.DNSNameFormat
	for _, p := range []struct{ k, v string }{
		{"namespace", r.Namespace}, {"tenant", r.Tenant}, {"project", r.Project}, {"region", r.Region},
		{"environment", r.Environment}, {"name", r.Name}, {"stage", r.Stage}, {"id", r.ID},
		{"attributes", idCtx["attributes"]}, {"domain_name", r.DomainName}, {"region", r.Region},
	} {
		dns = nfc(strings.ReplaceAll(dns, "${"+p.k+"}", p.v))
	}
	r.DNSName = dns

	// legacy label_order = setsubtract(label_order, incompatible) as a sorted set
	src := []string{}
	if in.LabelOrder != nil {
		src = *in.LabelOrder
	}
	r.legacyOrder = []string{}
	for _, l := range sortedUnique(src) {
		if !contains(legacyIncompatible, l) {
			r.legacyOrder = append(r.legacyOrder, l)
		}
	}
	return r, nil
}

func validateCases(key, val *string) error {
	if key != nil && !contains([]string{"lower", "title", "upper"}, *key) {
		return fmt.Errorf("label_key_case: Allowed values: `lower`, `title`, `upper`")
	}
	if val != nil && !contains([]string{"lower", "title", "upper", "none"}, *val) {
		return fmt.Errorf("label_value_case: Allowed values: `lower`, `title`, `upper`, `none`")
	}
	return nil
}

func validateLimit(limit *int64) error {
	if limit != nil && !(*limit >= 6 || *limit == 0) {
		return fmt.Errorf("The id_length_limit must be >= 6 if supplied (not null), or 0 for unlimited length")
	}
	return nil
}
