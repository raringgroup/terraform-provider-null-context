// Copyright 2026 Raring Group LLC
// Portions Copyright 2020-2022 Cloud Posse, LLC (derived from terraform-null-label)
// SPDX-License-Identifier: Apache-2.0

package label

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

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
		return re.ReplaceAllString(s, repl), nil
	}
	return strings.ReplaceAll(s, substr, repl), nil
}

// tfTitle mirrors Terraform's title(), which is Go's strings.Title.
func tfTitle(s string) string {
	return strings.Title(s) //nolint:staticcheck // identical to Terraform's implementation; parity matters more
}

func applyCase(s, mode string) string {
	switch mode {
	case "none":
		return s
	case "title":
		return tfTitle(strings.ToLower(s))
	case "upper":
		return strings.ToUpper(s)
	default:
		return strings.ToLower(s)
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

// substrRunes mirrors substr(s, 0, n) for n >= 0.
func substrRunes(s string, n int) string {
	r := []rune(s)
	if n >= len(r) {
		return s
	}
	return string(r[:n])
}

// tfFormat mirrors format() for string arguments. Terraform errors when the
// number of verbs and arguments differ; Go's fmt would print %!s(MISSING).
func tfFormat(format string, args []string) (string, error) {
	verbs := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			i++
			continue
		}
		verbs++
	}
	if verbs != len(args) {
		return "", fmt.Errorf("format %q has %d verbs but %d arguments were given", format, verbs, len(args))
	}
	anyArgs := make([]any, len(args))
	for i, a := range args {
		anyArgs[i] = a
	}
	return fmt.Sprintf(format, anyArgs...), nil
}
