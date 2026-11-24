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

"""Generate golden test vectors by running the *reference* null-context module.

Usage: python3 tools/vectors/gen.py --ref-module ../null-context --out testdata/vectors.json [--terraform terraform]

The reference module is copied into a scratch dir so it is never modified.
Requires: terraform (>= 1.3) on PATH, python3. No cloud credentials or providers needed.
"""
import argparse, glob, json, os, shutil, subprocess, sys, tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cases as C

SCALARS = ["enabled", "namespace", "tenant", "project", "region", "environment", "stage", "name", "delimiter",
           "label_order", "regex_replace_chars", "id_length_limit", "label_key_case", "label_value_case",
           "domain_name", "dns_name_format"]
DEFAULTS = {"attributes": "[]", "tags": "{}", "additional_tag_map": "{}", "descriptor_formats": "{}",
            "labels_as_tags": '["default"]'}

DEFAULT_CONTEXT = """
locals {
  default_context = {
    enabled = true, namespace = null, tenant = null, project = null, region = null, environment = null,
    stage = null, name = null, delimiter = null, attributes = [], tags = {}, additional_tag_map = {},
    regex_replace_chars = null, label_order = [], id_length_limit = null, label_key_case = null,
    label_value_case = null, descriptor_formats = {}, domain_name = null, dns_name_format = null,
    labels_as_tags = ["unset"]
  }
}
"""

def block(name, ctx_expr):
    out = [f'module "{name}" {{', f"  for_each = var.{name}", '  source   = "./ref"']
    for v in SCALARS:
        out.append(f"  {v} = try(each.value.inputs.{v}, null)")
    for v, d in DEFAULTS.items():
        out.append(f"  {v} = try(each.value.inputs.{v}, {d})")
    out.append(f"  context = {ctx_expr}")
    out.append("}")
    return "\n".join(out)

def harness():
    parts = ['variable "base" { type = any }', 'variable "chained" { type = any }',
             'variable "chained2" { type = any }', DEFAULT_CONTEXT,
             block("base", "try(each.value.context, local.default_context)"),
             block("chained", 'module.base[each.value.from][each.value.via]'),
             block("chained2", 'module.chained[each.value.from][each.value.via]'),
             'output "results" {',
             '  value = merge(',
             '    { for k, m in module.base : k => jsonencode(m) },',
             '    { for k, m in module.chained : k => jsonencode(m) },',
             '    { for k, m in module.chained2 : k => jsonencode(m) },',
             '  )', '}']
    return "\n".join(parts) + "\n"

def run(cmd, cwd, check=True):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    if check and r.returncode != 0:
        sys.stderr.write(r.stdout + r.stderr)
        raise SystemExit(f"command failed: {' '.join(cmd)}")
    return r

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--ref-module", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--terraform", default="terraform")
    a = ap.parse_args()

    base, chained, chained2, negative = C.build()
    work = tempfile.mkdtemp(prefix="ctxvec-")
    ref = os.path.join(work, "ref")
    os.makedirs(ref)
    for f in glob.glob(os.path.join(a.ref_module, "*.tf")):
        shutil.copy(f, ref)
    with open(os.path.join(work, "main.tf"), "w") as fh:
        fh.write(harness())
    with open(os.path.join(work, "cases.tfvars.json"), "w") as fh:
        json.dump({"base": base, "chained": chained, "chained2": chained2}, fh)
    tf = a.terraform
    run([tf, "init", "-no-color", "-input=false"], work)
    run([tf, "apply", "-no-color", "-input=false", "-auto-approve", "-var-file=cases.tfvars.json"], work)
    res = json.loads(run([tf, "output", "-json", "results"], work).stdout)
    results = {k: json.loads(v) for k, v in res.items()}

    ver = run([tf, "version", "-json"], work).stdout
    vectors = []
    for group, kind in ((base, "base"), (chained, "chained"), (chained2, "chained2")):
        for name, spec in group.items():
            v = {"name": name, "kind": kind, "inputs": spec["inputs"], "expect": results[name]}
            if "context" in spec:
                v["context"] = spec["context"]
            if "from" in spec:
                v["from"] = spec["from"]
                v["via"] = spec["via"]
            vectors.append(v)

    neg_out = []
    for name, spec in negative.items():
        d = tempfile.mkdtemp(prefix="ctxneg-")
        shutil.copytree(ref, os.path.join(d, "ref"))
        shutil.copy(os.path.join(work, "main.tf"), d)
        with open(os.path.join(d, "c.tfvars.json"), "w") as fh:
            one = {"base": {name: {k: v for k, v in spec.items() if k != "error_contains"}}, "chained": {}, "chained2": {}}
            json.dump(one, fh)
        run([tf, "init", "-no-color", "-input=false"], d)
        r = run([tf, "apply", "-no-color", "-input=false", "-auto-approve", "-var-file=c.tfvars.json"], d, check=False)
        err = r.stdout + r.stderr
        ok = r.returncode != 0 and spec["error_contains"] in err
        neg_out.append({"name": name, "inputs": spec["inputs"], "context": spec.get("context"),
                        "error_contains": spec["error_contains"], "terraform_failed": r.returncode != 0,
                        "message_matched": ok, "terraform_error_excerpt": err.strip().splitlines()[-12:] if r.returncode else []})
        shutil.rmtree(d, ignore_errors=True)

    out = {"reference": {"module_path": os.path.abspath(a.ref_module), "terraform_version": json.loads(ver).get("terraform_version")},
           "vectors": vectors, "negative": neg_out}
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    with open(a.out, "w") as fh:
        json.dump(out, fh, indent=1, sort_keys=True)
    shutil.rmtree(work, ignore_errors=True)
    bad = [n for n in neg_out if not n["message_matched"]]
    print(f"wrote {a.out}: {len(vectors)} vectors, {len(neg_out)} negative ({len(bad)} unexpected)")
    for n in bad:
        print("  UNEXPECTED negative:", n["name"], n["terraform_failed"], n["terraform_error_excerpt"][-4:])

if __name__ == "__main__":
    main()
