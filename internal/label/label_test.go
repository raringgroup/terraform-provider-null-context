// Copyright 2026 Raring Group LLC
// Portions Copyright 2020-2022 Cloud Posse, LLC (derived from terraform-null-label)
// SPDX-License-Identifier: Apache-2.0

package label

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type vector struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind"`
	Inputs  Context         `json:"inputs"`
	Context json.RawMessage `json:"context"`
	From    string          `json:"from"`
	Via     string          `json:"via"`
	Expect  map[string]any  `json:"expect"`
}

type negative struct {
	Name     string          `json:"name"`
	Inputs   Context         `json:"inputs"`
	Context  json.RawMessage `json:"context"`
	Contains string          `json:"error_contains"`
}

type vectorFile struct {
	Vectors  []vector   `json:"vectors"`
	Negative []negative `json:"negative"`
}

func loadVectors(t *testing.T) vectorFile {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "vectors.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("vectors not generated (%v); run `make vectors`", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(b, &vf); err != nil {
		t.Fatal(err)
	}
	return vf
}

// normalize round-trips through JSON so Go-typed values compare equal to
// values decoded from Terraform's JSON.
func normalize(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestParityWithReferenceModule(t *testing.T) {
	vf := loadVectors(t)
	results := map[string]*Result{}
	for _, v := range vf.Vectors {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			parent := DefaultContext()
			switch {
			case v.From != "":
				src, ok := results[v.From]
				if !ok {
					t.Fatalf("chain source %q not computed yet", v.From)
				}
				var m map[string]any
				switch v.Via {
				case "normalized_context":
					m = src.NormalizedContext()
				case "self":
					m = src.Self()
				default:
					t.Fatalf("unsupported via %q", v.Via)
				}
				var err error
				if parent, err = ContextFromMap(normalize(t, m).(map[string]any)); err != nil {
					t.Fatal(err)
				}
			case len(v.Context) > 0:
				if err := json.Unmarshal(v.Context, &parent); err != nil {
					t.Fatal(err)
				}
			}
			res, err := Compute(parent, v.Inputs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			results[v.Name] = res
			got := normalize(t, res.Outputs()).(map[string]any)
			keys := make([]string, 0, len(v.Expect))
			for k := range v.Expect {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if !reflect.DeepEqual(got[k], v.Expect[k]) {
					t.Errorf("output %q mismatch\n  got:  %v\n  want: %v", k, mustJSON(got[k]), mustJSON(v.Expect[k]))
				}
			}
			for k := range got {
				if _, ok := v.Expect[k]; !ok {
					t.Errorf("unexpected extra output %q", k)
				}
			}
		})
	}
}

func TestNegativeCases(t *testing.T) {
	vf := loadVectors(t)
	for _, n := range vf.Negative {
		n := n
		t.Run(n.Name, func(t *testing.T) {
			parent := DefaultContext()
			if len(n.Context) > 0 && string(n.Context) != "null" {
				if err := json.Unmarshal(n.Context, &parent); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Compute(parent, n.Inputs); err == nil {
				t.Fatalf("expected an error containing %q, got none", n.Contains)
			}
		})
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
