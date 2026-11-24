#!/usr/bin/env python3
"""Run every golden vector through the *built provider* under real Terraform.

Both entry points are exercised: the `null-context` data source and
`provider::null-context::label()`. Compares all 23 module outputs per vector.

Usage: python3 tools/vectors/check_provider.py --vectors testdata/vectors.json \
          --provider-dir ./bin [--terraform terraform]
`--provider-dir` must contain the built `terraform-provider-null-context` binary; it is
wired in with a throw-away dev_overrides CLI config (nothing global is touched).
"""
import argparse, json, os, shutil, subprocess, sys, tempfile

ADDR = "registry.terraform.io/raringgroup/null-context"
SCALARS = ["enabled", "namespace", "tenant", "project", "region", "environment", "stage", "name", "delimiter",
           "label_order", "regex_replace_chars", "id_length_limit", "label_key_case", "label_value_case",
           "domain_name", "dns_name_format", "attributes", "tags", "additional_tag_map", "descriptor_formats",
           "labels_as_tags"]
OUTPUTS = ["id", "id_full", "dns_name", "enabled", "namespace", "tenant", "environment", "name", "stage", "delimiter",
           "attributes", "tags", "additional_tag_map", "label_order", "regex_replace_chars", "id_length_limit",
           "tags_as_list_of_maps", "descriptors", "normalized_context", "legacy", "self", "dns_name_format", "domain_name"]

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--vectors", required=True)
    ap.add_argument("--provider-dir", required=True)
    ap.add_argument("--terraform", default="terraform")
    a = ap.parse_args()
    vf = json.load(open(a.vectors))
    work = tempfile.mkdtemp(prefix="ctxprov-")
    os.makedirs(os.path.join(work, "j"))
    rc = os.path.join(work, "tfrc")
    open(rc, "w").write('provider_installation {\n  dev_overrides { "%s" = "%s" }\n  direct {}\n}\n' % (ADDR, os.path.abspath(a.provider_dir)))
    env = dict(os.environ, TF_CLI_CONFIG_FILE=rc, TF_IN_AUTOMATION="1")

    hcl = ['terraform {\n  required_providers { null-context = { source = "raringgroup/null-context" } }\n}\n', 'provider "null-context" {}\n']
    outs_ds, outs_fn = [], []
    for v in vf["vectors"]:
        n = v["name"]
        json.dump(v["inputs"], open(os.path.join(work, "j", n + ".vars.json"), "w"))
        has_ctx = "context" in v
        if has_ctx:
            json.dump(v["context"], open(os.path.join(work, "j", n + ".ctx.json"), "w"))
        if v["kind"] == "base":
            fn_ctx = f'jsondecode(file("j/{n}.ctx.json"))' if has_ctx else "null"
            ds_ctx = f'  context = jsondecode(file("j/{n}.ctx.json"))\n' if has_ctx else ""
        else:
            fn_ctx = f'local.f_{v["from"]}.{v["via"]}'
            ds_ctx = f'  context = data.null-context.{v["from"]}.{v["via"]}\n'
        hcl.append(f'locals {{\n  v_{n} = jsondecode(file("j/{n}.vars.json"))\n'
                   f'  f_{n} = provider::null-context::label({fn_ctx}, local.v_{n})\n}}\n')
        body = ds_ctx
        for k in SCALARS:
            if v["inputs"].get(k) is not None:
                body += f"  {k} = local.v_{n}.{k}\n"
        hcl.append(f'data "null-context" "{n}" {{\n{body}}}\n')
        outs_fn.append(f'    {n} = jsonencode(local.f_{n})')
        outs_ds.append(f'    {n} = jsonencode({{ ' + ", ".join(f"{o} = data.null-context.{n}.{o}" for o in OUTPUTS) + ' })')
    hcl.append('output "fn" {\n  value = {\n' + "\n".join(outs_fn) + '\n  }\n}\n')
    hcl.append('output "ds" {\n  value = {\n' + "\n".join(outs_ds) + '\n  }\n}\n')
    open(os.path.join(work, "main.tf"), "w").write("\n".join(hcl))

    r = subprocess.run([a.terraform, "apply", "-auto-approve", "-no-color", "-input=false"], cwd=work, env=env, capture_output=True, text=True)
    if r.returncode != 0:
        sys.stderr.write((r.stdout + r.stderr)[-6000:])
        raise SystemExit("terraform apply failed")
    bad = 0
    for kind in ("fn", "ds"):
        res = json.loads(subprocess.run([a.terraform, "output", "-json", kind], cwd=work, env=env, capture_output=True, text=True).stdout)
        for v in vf["vectors"]:
            got = json.loads(res[v["name"]])
            for k in OUTPUTS:
                if got.get(k) != v["expect"][k]:
                    bad += 1
                    print(f"MISMATCH [{kind}] {v['name']} .{k}\n   got : {json.dumps(got.get(k), sort_keys=True)[:300]}\n   want: {json.dumps(v['expect'][k], sort_keys=True)[:300]}")
    shutil.rmtree(work, ignore_errors=True)
    print(f"provider parity: {len(vf['vectors'])} vectors x 2 entry points x {len(OUTPUTS)} outputs, {bad} mismatches")
    sys.exit(1 if bad else 0)

if __name__ == "__main__":
    main()
