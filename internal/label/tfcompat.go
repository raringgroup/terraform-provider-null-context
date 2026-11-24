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
//  ./tfcompat.go
//  This file contains code ported from terraform-null-context and modified by Raring Group LLC
// ----------------------------------------------------------------------------

package label

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/apparentlymart/go-textseg/v15/textseg"
	"golang.org/x/text/unicode/norm"
)

// nfc mirrors cty.StringVal: every string value in Terraform is NFC-normalized, including the
// result of every string function. Without this, decomposed input ("a" + U+0308) behaves
// differently from the module (it would survive or be stripped by regexes differently).
func nfc(s string) string { return norm.NFC.String(s) }

// This file ports the handful of Terraform built-in functions that the
// null-context module relies on. Each helper mirrors the Terraform semantics
// (see tfcompat notes in docs/PARITY.md), not Go's idea of the same operation.

// tfReplace mirrors Terraform's replace(): a substring wrapped in forward
// slashes is a regular expression (RE2), otherwise it is a literal substring.
func tfReplace(s, substr, repl string) (string, error) {
	if len(substr) > 1 && substr[0] == '/' && substr[len(substr)-1] == '/' {
		re, err := regexp.Compile(substr[1 : len(substr)-1])
		if err != nil {
			return "", fmt.Errorf("error parsing regexp: %w", err)
		}
		return nfc(re.ReplaceAllString(s, repl)), nil
	}
	return nfc(strings.ReplaceAll(s, substr, repl)), nil
}

// tfTitle mirrors Terraform's title(), which is Go's strings.Title.
func tfTitle(s string) string {
	return nfc(strings.Title(s)) //nolint:staticcheck // identical to Terraform's implementation; parity matters more
}

func applyCase(s, mode string) string {
	switch mode {
	case "none":
		return s
	case "title":
		return tfTitle(nfc(strings.ToLower(s)))
	case "upper":
		return nfc(strings.ToUpper(s))
	default:
		return nfc(strings.ToLower(s))
	}
}

// compact mirrors compact(): drops empty strings.
func compact(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// distinct mirrors distinct(): keeps first occurrence, preserves order.
func distinct(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// sortedUnique is what Terraform does when a list is converted to set(string).
func sortedUnique(in []string) []string {
	out := distinct(in)
	sort.Strings(out)
	return out
}

func contains(in []string, v string) bool {
	for _, x := range in {
		if x == v {
			return true
		}
	}
	return false
}

func mergeMaps(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// tfStrlen mirrors length() on strings: it counts extended grapheme clusters, not runes.
func tfStrlen(s string) int {
	b, n := []byte(s), 0
	for len(b) > 0 {
		adv, _, err := textseg.ScanGraphemeClusters(b, true)
		if err != nil || adv == 0 {
			return n + len([]rune(string(b)))
		}
		b = b[adv:]
		n++
	}
	return n
}

// tfSubstr mirrors substr(s, 0, n) for n >= 0: it keeps the first n grapheme clusters.
func tfSubstr(s string, n int) string {
	b, pos := []byte(s), 0
	for i := 0; i < n && pos < len(b); i++ {
		adv, _, err := textseg.ScanGraphemeClusters(b[pos:], true)
		if err != nil || adv == 0 {
			break
		}
		pos += adv
	}
	return nfc(string(b[:pos]))
}

// normalizeContext applies cty's NFC normalization to every string, including map keys.
func normalizeContext(c Context) Context {
	ps := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := nfc(*p)
		return &v
	}
	list := func(in []string) []string {
		if in == nil {
			return nil
		}
		out := make([]string, len(in))
		for i, v := range in {
			out[i] = nfc(v)
		}
		return out
	}
	mp := func(in map[string]string) map[string]string {
		if in == nil {
			return nil
		}
		out := make(map[string]string, len(in))
		for k, v := range in {
			out[nfc(k)] = nfc(v)
		}
		return out
	}
	out := c
	out.Namespace, out.Tenant, out.Project, out.Region = ps(c.Namespace), ps(c.Tenant), ps(c.Project), ps(c.Region)
	out.Environment, out.Stage, out.Name, out.Delimiter = ps(c.Environment), ps(c.Stage), ps(c.Name), ps(c.Delimiter)
	out.RegexReplaceChars, out.LabelKeyCase, out.LabelValueCase = ps(c.RegexReplaceChars), ps(c.LabelKeyCase), ps(c.LabelValueCase)
	out.DomainName, out.DNSNameFormat = ps(c.DomainName), ps(c.DNSNameFormat)
	out.Attributes, out.LabelsAsTags = list(c.Attributes), list(c.LabelsAsTags)
	out.Tags, out.AdditionalTagMap = mp(c.Tags), mp(c.AdditionalTagMap)
	if c.LabelOrder != nil {
		lo := list(*c.LabelOrder)
		out.LabelOrder = &lo
	}
	if c.DescriptorFormats != nil {
		out.DescriptorFormats = make(map[string]DescriptorFormat, len(c.DescriptorFormats))
		for k, d := range c.DescriptorFormats {
			out.DescriptorFormats[nfc(k)] = DescriptorFormat{Format: nfc(d.Format), Labels: list(d.Labels)}
		}
	}
	return out
}

// jsonQuote is what Terraform's format("%q", s) does: JSON string quoting (encoding/json with HTML
// escaping), NOT strconv.Quote. Verified empirically: `"`, `\`, control characters, `<`, `>`, `&` and
// U+2028/U+2029 are escaped; every other character (zero-width joiner, NBSP, private use...) stays raw.
func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// tfFormat mirrors format() for string arguments: %s and %v (raw), %q (JSON quoting), %% and the
// flags '-' and '0' with width and precision. Verified against Terraform: width and precision count
// grapheme clusters (not runes), precision truncates BEFORE %q quotes, width pads the quoted text,
// '+', ' ' and '#' are ignored for strings, and '-' with '0' zero-fills on the right. Argument and verb counts must match; other verbs and
// `*` widths are errors (Terraform errors too).
func tfFormat(format string, args []string) (string, error) {
	var b strings.Builder
	ai := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			b.WriteByte(format[i])
			continue
		}
		j := i + 1
		if j < len(format) && format[j] == '%' {
			b.WriteByte('%')
			i = j
			continue
		}
		minus, zero := false, false
		for j < len(format) && strings.IndexByte("+-# 0", format[j]) >= 0 {
			switch format[j] {
			case '-':
				minus = true
			case '0':
				zero = true
			}
			j++
		}
		width, prec := 0, -1
		for j < len(format) && format[j] >= '0' && format[j] <= '9' {
			width = width*10 + int(format[j]-'0')
			j++
		}
		if j < len(format) && format[j] == '.' {
			j++
			// "%.s" (no digits) means no precision in Terraform, unlike Go where it means 0.
			for j < len(format) && format[j] >= '0' && format[j] <= '9' {
				if prec < 0 {
					prec = 0
				}
				prec = prec*10 + int(format[j]-'0')
				j++
			}
		}
		if j >= len(format) {
			return "", fmt.Errorf("format %q: incomplete verb at the end", format)
		}
		verb := format[j]
		if ai >= len(args) {
			return "", fmt.Errorf("not enough arguments for %q", format)
		}
		arg := args[ai]
		ai++
		if verb != 's' && verb != 'v' && verb != 'q' {
			return "", fmt.Errorf("unsupported format verb %%%c for a string argument", verb)
		}
		if prec >= 0 {
			arg = tfSubstr(arg, prec)
		}
		if verb == 'q' {
			arg = jsonQuote(arg)
		}
		if pad := width - tfStrlen(arg); pad > 0 {
			switch {
			case minus && zero: // Terraform: "%-08s" pads with zeros on the right (Go would use spaces)
				arg += strings.Repeat("0", pad)
			case minus:
				arg += strings.Repeat(" ", pad)
			case zero:
				arg = strings.Repeat("0", pad) + arg
			default:
				arg = strings.Repeat(" ", pad) + arg
			}
		}
		b.WriteString(arg)
		i = j
	}
	if ai != len(args) {
		return "", fmt.Errorf("too many arguments; only %d used by format string", ai)
	}
	return nfc(b.String()), nil
}
