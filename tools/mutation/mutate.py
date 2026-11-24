#!/usr/bin/env python3
# ----------------------------------------------------------------------------
#  Copyright 2026 Raring Group LLC
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
# ----------------------------------------------------------------------------

"""Mutation check for the label core: each mutation must make `go test ./internal/label/` fail.

A SURVIVED mutation means the golden vectors do not cover that behaviour: add vectors
in tools/vectors/cases.py, regenerate (`make vectors`), and re-run.
"""
import os, shutil, subprocess, sys, tempfile

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
L, T, J = "internal/label/label.go", "internal/label/tfcompat.go", "internal/label/json.go"
MUTATIONS = [
    ("hash_suffix", L, 'hashSuffix     = "qrstuvwxyz"', 'hashSuffix     = "qrstuvwxyA"'),
    ("hash_len", L, "idHashLength   = 5", "idHashLength   = 4"),
    ("default_delim", L, 'defaultDelim   = "-"', 'defaultDelim   = "_"'),
    ("default_order", L, '[]string{"namespace", "environment", "stage", "name", "attributes"}', '[]string{"namespace", "environment", "name", "stage", "attributes"}'),
    ("dns_order_region_twice", L, '{"domain_name", r.DomainName}, {"region", r.Region},', '{"domain_name", r.DomainName},'),
    ("dns_order_id", L, '{"name", r.Name}, {"stage", r.Stage}, {"id", r.ID},', '{"id", r.ID}, {"name", r.Name}, {"stage", r.Stage},'),
    ("dns_order_attributes", L, '{"attributes", idCtx["attributes"]}, {"domain_name", r.DomainName}', '{"domain_name", r.DomainName}, {"attributes", idCtx["attributes"]}'),
    ("dns_order_namespace_tenant", L, '{"namespace", r.Namespace}, {"tenant", r.Tenant}, {"project", r.Project}, {"region", r.Region},', '{"tenant", r.Tenant}, {"namespace", r.Namespace}, {"project", r.Project}, {"region", r.Region},'),
    ("tag_name_is_id", L, '"name": r.ID, "attributes"', '"name": r.Name, "attributes"'),
    ("tag_merge_order", L, "r.Tags = mergeMaps(generated, in.Tags)", "r.Tags = mergeMaps(in.Tags, generated)"),
    ("labels_as_tags_first_wins", L, "in.LabelsAsTags = sortedUnique(parent.LabelsAsTags)", "in.LabelsAsTags = sortedUnique(vars.LabelsAsTags)"),
    ("labels_as_tags_unset_sentinel", L, 'parent.LabelsAsTags == nil || contains(parent.LabelsAsTags, "unset")', "parent.LabelsAsTags == nil"),
    ("labels_as_tags_default_sorted", L, "in.LabelsAsTags = sortedUnique(vl)", "in.LabelsAsTags = vl"),
    ("coalesce_skips_empty_regex", L, 'in.RegexReplaceChars != nil && *in.RegexReplaceChars != ""', "in.RegexReplaceChars != nil"),
    ("coalescelist_label_order", L, "in.LabelOrder != nil && len(*in.LabelOrder) > 0", "in.LabelOrder != nil"),
    ("trim_trailing_delimiter", L, "strings.TrimSuffix(tfSubstr(r.IDFull, truncLimit), r.Delimiter) + r.Delimiter", "tfSubstr(r.IDFull, truncLimit) + r.Delimiter"),
    ("limit_strictly_greater", L, "tfStrlen(r.IDFull) > int(r.IDLengthLimit)", "tfStrlen(r.IDFull) >= int(r.IDLengthLimit)"),
    ("truncate_boundary", L, "if truncLimit > 0 {", "if truncLimit >= 0 {"),
    ("hash_case_upper", L, "hash = strings.ToUpper(hash)", "hash = strings.ToLower(hash)"),
    ("hash_regex_applied", L, 'if hash, err = tfReplace(hash, regex, ""); err != nil {', 'if hash, err = tfReplace(hash, "/x^/", ""); err != nil {'),
    ("attributes_merge_order", L, "append(append([]string{}, parent.Attributes...), vars.Attributes...)", "append(append([]string{}, vars.Attributes...), parent.Attributes...)"),
    ("descriptor_formats_merge", L, "\tfor k, v := range vars.DescriptorFormats {\n\t\tin.DescriptorFormats[k] = v\n\t}", ""),
    ("enabled_var_wins", L, "\tif in.Enabled == nil {\n\t\tin.Enabled = parent.Enabled\n\t}", "\tin.Enabled = parent.Enabled"),
    ("tags_merge_order", L, "Tags:              mergeMaps(parent.Tags, vars.Tags),", "Tags:              mergeMaps(vars.Tags, parent.Tags),"),
    ("additional_tag_map_in_list", L, "\t\tfor ak, av := range r.AdditionalTagMap {\n\t\t\tm[ak] = av\n\t\t}", ""),
    ("key_case_upper", L, "k = strings.ToUpper(l)", "k = strings.ToLower(l)"),
    ("disabled_gating", J, "attrs, tags = []string{}, map[string]string{}", "attrs, tags = r.Attributes, r.Tags"),
    ("legacy_label_order", J, 'm["label_order"] = r.legacyOrder', 'm["label_order"] = r.LabelOrder'),
    ("title_implementation", T, "return nfc(strings.Title(s)) //nolint:staticcheck // identical to Terraform's implementation; parity matters more", "return strings.ToUpper(s[:1]) + s[1:]"),
    ("regex_slash_rule", T, "len(substr) > 1 && substr[0] ==", "len(substr) > 2 && substr[0] =="),
    ("value_case_none", T, 'case "none":\n\t\treturn s', 'case "none":\n\t\treturn strings.ToLower(s)'),
    ("nfc_normalization", T, "func nfc(s string) string { return norm.NFC.String(s) }", "func nfc(s string) string { _ = norm.NFC; return s }"),
    ("grapheme_id_length", L, "tfStrlen(r.IDFull) > int(r.IDLengthLimit)", "len([]rune(r.IDFull)) > int(r.IDLengthLimit)"),
    ("grapheme_delimiter_length", L, "limit - (idHashLength + tfStrlen(r.Delimiter))", "limit - (idHashLength + len(r.Delimiter))"),
    ("quote_is_json_html_escaped", T, "return string(b)\n}", "return strings.ReplaceAll(string(b), \"\\\\u003c\", \"<\")\n}"),
    ("format_width_in_clusters", T, "if pad := width - tfStrlen(arg); pad > 0 {", "if pad := width - len([]rune(arg)); pad > 0 {"),
    ("format_precision_in_clusters", T, "arg = tfSubstr(arg, prec)", "arg = string([]rune(arg)[:min(prec, len([]rune(arg)))])"),
    ("format_precision_before_quote", T, "\t\tif prec >= 0 {\n\t\t\targ = tfSubstr(arg, prec)\n\t\t}\n\t\tif verb == 'q' {\n\t\t\targ = jsonQuote(arg)\n\t\t}", "\t\tif verb == 'q' {\n\t\t\targ = jsonQuote(arg)\n\t\t}\n\t\tif prec >= 0 {\n\t\t\targ = tfSubstr(arg, prec)\n\t\t}"),
    ("format_zero_flag", T, "arg = strings.Repeat(\"0\", pad) + arg", "arg = strings.Repeat(\" \", pad) + arg"),
    ("format_left_zero_fill", T, "case minus && zero:", "case false:"),
    ("format_too_many_args", T, "if ai != len(args) {", "if false {"),
    ("format_unsupported_verb", T, "if verb != 's' && verb != 'v' && verb != 'q' {", "if false {"),
]

def main():
    work = tempfile.mkdtemp(prefix="ctxmut-")
    shutil.copytree(ROOT, work, dirs_exist_ok=True, ignore=shutil.ignore_patterns(".git", "bin", ".terraform"))
    base = subprocess.run(["go", "test", "./internal/label/"], cwd=work, capture_output=True, text=True)
    if base.returncode != 0:
        print(base.stdout[-3000:] + base.stderr[-1000:])
        sys.exit("baseline tests fail: mutation results would be meaningless (every mutation would look killed)")
    survivors = []
    for name, rel, old, new in MUTATIONS:
        path = os.path.join(work, rel)
        orig = open(os.path.join(ROOT, rel)).read()
        if old not in orig:
            print(f"STALE     {name}: pattern not found in {rel} (update tools/mutation/mutate.py)")
            survivors.append(name)
            continue
        open(path, "w").write(orig.replace(old, new, 1))
        r = subprocess.run(["go", "test", "./internal/label/"], cwd=work, capture_output=True, text=True)
        open(path, "w").write(orig)
        if r.returncode == 0:
            print(f"SURVIVED  {name}")
            survivors.append(name)
        else:
            print(f"killed    {name}")
    shutil.rmtree(work, ignore_errors=True)
    print(f"\n{len(MUTATIONS) - len(survivors)}/{len(MUTATIONS)} mutations killed")
    sys.exit(1 if survivors else 0)

if __name__ == "__main__":
    main()
