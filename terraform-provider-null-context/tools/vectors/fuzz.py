#!/usr/bin/env python3
"""Differential fuzzing: random (but valid) inputs through the REFERENCE module.

Writes testdata/fuzz-<seed>.json in the same format as vectors.json, so the Go core
(`go test`) and the built provider (`check_provider.py --vectors ...`) can be compared to it.

Usage: python3 tools/vectors/fuzz.py --ref-module ../null-context --seed 1 --base 300 --chained 300 --chained2 60
Only valid inputs are generated (an invalid one aborts the whole terraform run); error behaviour is
covered by the `negative` cases in gen.py.
"""
import argparse, json, os, random, shutil, subprocess, sys, tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gen

VALID = ["namespace", "tenant", "project", "region", "environment", "stage", "name", "attributes"]
WORDS = ["app", "Core", "svc", "API", "data", "web", "Main", "x", "Z9", "a1b2", "prod", "Dev", "Stage", "net", "db", "k8s"]
MESSY = ["My App", "a.b_c", "FOO-bar", "  pad ", "x/y", "a$b", "{q}", "A--B", "tab\tx", "a${id}b", "${name}", "${region}",
         "UPPER lower", "snake_case_name", "a\u200db", "x<y>&z", "p\u2028q", "q\"uote", "back\\slash", "\u00a0nb", "\ue000pua", "ctl\u0001x", "del\u007fx", "dot.name.x", "-lead", "trail-", "a b  c", "%s", "100%"]
UNI = ["caf\u00e9", "na\u00efve", "Stra\u00dfe", "\u0130stanbul", "\u01c6", "\u03a3\u0391\u03a3", "\u4e2d\u6587\u540d", "\u65e5\u672c\u8a9e",
       "\U0001f600", "\U0001f468\u200d\U0001f469\u200d\U0001f467", "\U0001f1ee\U0001f1f3", "e\u0301", "a\u0308b", "\ud55c\uae00",
       "\ufb01ne", "\u01c5ungla", "\u03a9", "\u00ff", "\u00f1and\u00fa", "\u1e9e", "\u0e01\u0e33", "o\u0302\u0323", "a\u0300\u0301"]
REGEXES = [None, None, None, "", "/[^-a-zA-Z0-9]/", "/[^a-z0-9]/", "/[^\\p{L}\\p{N}-]/", "/x^/", "/[aeiou]/", "/\\s+/", "/[0-9]/",
           "/[0-9a-f]/", "/[^-a-zA-Z0-9${}]/", "x", "a", "/[A-Z]/", "/[^\\x00-\\x7F]/", "/\\p{M}/", "/./"]
DELIMS = [None, None, "-", "_", ".", "", "--", "/", "\u00e9", "\U0001f600", "~~~~~~", " ", "ab"]
TOKENS = ["${namespace}", "${tenant}", "${project}", "${region}", "${environment}", "${name}", "${stage}", "${id}",
          "${attributes}", "${domain_name}", "-", ".", "x", "[", "]"]
VERBS = ["%s", "%v", "%q", "%5s", "%-8q", "%.3s", "%08s", "%-08s", "%6.2q", "%.s"]

def rstr(r):
    x = r.random()
    if x < 0.5: s = r.choice(WORDS)
    elif x < 0.65: s = r.choice(MESSY)
    elif x < 0.93: s = r.choice(UNI)
    else: s = ""
    if r.random() < 0.25: s += r.choice(["", "-", "_", ".", " ", "/"]) + (r.choice(WORDS) if r.random() < 0.6 else r.choice(UNI))
    return s

def rinputs(r, density=0.5):
    d = {}
    def maybe(k, v, p=density):
        if r.random() < p: d[k] = v
    for k in ["namespace", "tenant", "project", "region", "environment", "stage", "name"]:
        maybe(k, rstr(r) if r.random() < 0.95 else None)
    maybe("attributes", [rstr(r) for _ in range(r.choice([0, 1, 2, 3, 5]))], 0.5)
    maybe("tags", {rstr(r) or "k": rstr(r) for _ in range(r.choice([0, 1, 2, 3]))}, 0.4)
    maybe("additional_tag_map", {rstr(r) or "k": rstr(r) for _ in range(r.choice([0, 1, 2]))}, 0.2)
    maybe("delimiter", r.choice(DELIMS), 0.4)
    maybe("regex_replace_chars", r.choice(REGEXES), 0.5)
    maybe("label_value_case", r.choice(["lower", "upper", "title", "none"]), 0.5)
    maybe("label_key_case", r.choice(["lower", "upper", "title"]), 0.4)
    maybe("id_length_limit", r.choice([0, 0, 6, 7, 8, 10, 12, 15, 16, 20, 25, 30, 45]), 0.5)
    maybe("label_order", r.sample(VALID, r.randint(0, len(VALID))) if r.random() < 0.8 else
          [r.choice(VALID) for _ in range(r.randint(1, 5))], 0.5)
    maybe("labels_as_tags", r.choice([["default"], [], ["name"], r.sample(VALID, r.randint(1, 4)), ["unset", "name"]]), 0.4)
    maybe("enabled", r.random() > 0.15, 0.3)
    maybe("domain_name", r.choice(["example.com", "x.y", "\u00e9.io", ""]), 0.4)
    maybe("dns_name_format", "".join(r.choice(TOKENS) for _ in range(r.randint(0, 6))), 0.4)
    if r.random() < 0.3:
        df = {}
        for _ in range(r.randint(1, 3)):
            k = r.randint(0, 3); labels = [r.choice(VALID) for _ in range(k)]
            fmt = "".join(r.choice(["pre", "-", "%%", ""]) + v for v in [r.choice(VERBS) for _ in labels])
            df[r.choice(["a", "b", "c", "d\u00e9"])] = {"format": fmt + r.choice(["", "!", "%%"]), "labels": labels}
        d["descriptor_formats"] = df
    return d

def build(seed, nb, nc, nc2):
    r = random.Random(seed)
    base, chained, chained2 = {}, {}, {}
    for i in range(nb):
        base[f"f{seed}_b{i}"] = {"inputs": rinputs(r)}
    bn = list(base)
    for i in range(nc):
        chained[f"f{seed}_c{i}"] = {"from": r.choice(bn), "via": r.choice(["self", "normalized_context"]), "inputs": rinputs(r, 0.25)}
    cn = list(chained)
    for i in range(nc2):
        chained2[f"f{seed}_d{i}"] = {"from": r.choice(cn), "via": r.choice(["self", "normalized_context"]), "inputs": rinputs(r, 0.25)}
    return base, chained, chained2

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--ref-module", required=True)
    ap.add_argument("--seed", type=int, default=1)
    ap.add_argument("--base", type=int, default=300)
    ap.add_argument("--chained", type=int, default=300)
    ap.add_argument("--chained2", type=int, default=60)
    ap.add_argument("--out", default=None)
    ap.add_argument("--terraform", default="terraform")
    a = ap.parse_args()
    out = a.out or os.path.join(os.path.dirname(__file__), "..", "..", "testdata", f"fuzz-{a.seed}.json")
    base, chained, chained2 = build(a.seed, a.base, a.chained, a.chained2)
    work = tempfile.mkdtemp(prefix="ctxfuzz-")
    ref = os.path.join(work, "ref"); os.makedirs(ref)
    for f in os.listdir(a.ref_module):
        if f.endswith(".tf"): shutil.copy(os.path.join(a.ref_module, f), ref)
    open(os.path.join(work, "main.tf"), "w").write(gen.harness())
    json.dump({"base": base, "chained": chained, "chained2": chained2}, open(os.path.join(work, "cases.tfvars.json"), "w"))
    tf = a.terraform
    gen.run([tf, "init", "-no-color", "-input=false"], work)
    gen.run([tf, "apply", "-no-color", "-input=false", "-auto-approve", "-var-file=cases.tfvars.json"], work)
    res = json.loads(gen.run([tf, "output", "-json", "results"], work).stdout)
    vectors = []
    for group, kind in ((base, "base"), (chained, "chained"), (chained2, "chained2")):
        for name, spec in group.items():
            v = {"name": name, "kind": kind, "inputs": spec["inputs"], "expect": json.loads(res[name])}
            if "from" in spec: v["from"], v["via"] = spec["from"], spec["via"]
            vectors.append(v)
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    json.dump({"reference": {"seed": a.seed}, "vectors": vectors, "negative": []}, open(out, "w"), sort_keys=True)
    shutil.rmtree(work, ignore_errors=True)
    print(f"wrote {out}: {len(vectors)} fuzz vectors")

if __name__ == "__main__":
    main()
