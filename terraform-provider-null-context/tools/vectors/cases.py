"""Deterministic test-case definitions for the null-context reference module.

Produces three groups:
  base      - cases that call the reference module directly (optionally with a raw `context`)
  chained   - cases whose context is an output (normalized_context/self/legacy) of a base case
  chained2  - cases whose context is an output of a chained case
  negative  - cases that must fail; run one by one, expected error substring recorded
"""
import copy

FULL_ORDER = ["namespace", "tenant", "project", "region", "environment", "stage", "name", "attributes"]

RICH = {
    "enabled": True, "namespace": "7pi", "tenant": "Foo", "project": "Project 1",
    "region": "us-east-1", "environment": "dev", "stage": "sandbox-01", "name": "baz",
    "delimiter": "-", "attributes": ["1", "2"], "tags": {"Compliance": "none"},
    "domain_name": "sevenpico.com", "dns_name_format": "${name}.${domain_name}",
}
MESSY = {
    "namespace": "Acme Corp_X", "tenant": "Ten.ant", "project": "Proj/ect 9", "region": "US East 1",
    "environment": "De v", "stage": "Stage_01", "name": "My-App.v2", "attributes": ["Work ers", "c_d"],
}
BASIC = {"namespace": "acme", "environment": "dev", "stage": "app", "name": "vpc"}

def ctx(**kw):
    c = {
        "enabled": True, "namespace": None, "tenant": None, "project": None, "region": None,
        "environment": None, "stage": None, "name": None, "delimiter": None, "attributes": [],
        "tags": {}, "additional_tag_map": {}, "regex_replace_chars": None, "label_order": [],
        "id_length_limit": None, "label_key_case": None, "label_value_case": None,
        "descriptor_formats": {}, "domain_name": None, "dns_name_format": None,
        "labels_as_tags": ["unset"],
    }
    c.update(kw)
    return c

def build():
    base, chained, chained2, negative = {}, {}, {}, {}
    def add(name, inputs=None, context=None):
        assert name not in base, name
        base[name] = {"inputs": inputs or {}}
        if context is not None:
            base[name]["context"] = context

    add("basic", BASIC)
    add("rich", RICH)
    add("rich_full_order", {**RICH, "label_order": FULL_ORDER})
    PROJ_ORDER = ["namespace", "project", "environment", "stage", "name", "attributes"]  # project yes, region no
    add("proj_order", {**RICH, "label_order": PROJ_ORDER})
    add("proj_order_limit", {**RICH, "label_order": PROJ_ORDER, "id_length_limit": 24})
    add("empty_everything", {})
    add("name_only", {"name": "solo"})

    for i, d in enumerate(["-", "_", ".", "", "--", "/", "a", " "]):
        add(f"delim_{i}", {**RICH, "delimiter": d, "label_order": FULL_ORDER})

    for vc in ["lower", "upper", "title", "none", "weird"]:
        for kc in ["lower", "upper", "title"]:
            if vc == "weird":
                continue
            add(f"case_{vc}_{kc}", {**MESSY, "label_value_case": vc, "label_key_case": kc, "label_order": FULL_ORDER,
                                    "tags": {"Owner": "Me"}})
    add("case_default_messy", {**MESSY, "label_order": FULL_ORDER})
    add("case_title_nodelim", {**MESSY, "label_value_case": "title", "delimiter": "", "label_order": FULL_ORDER})

    orders = {
        "default": None, "empty": [], "name": ["name"], "name_ns": ["name", "namespace"], "full": FULL_ORDER,
        "attr_name": ["attributes", "name"], "dup": ["environment", "environment", "name"],
        "proj_region": ["project", "region", "name"], "tenant_first": ["tenant", "namespace", "name"],
        "reverse": list(reversed(FULL_ORDER)),
    }
    for k, o in orders.items():
        add(f"order_{k}", {**RICH, "label_order": o})

    regexes = {
        "null": None, "empty": "", "az": "/[^a-z]/", "default": "/[^-a-zA-Z0-9]/", "literal": "x",
        "vowels": "/[aeiou]/", "digits": "/[0-9]/", "space": "/\\s/", "all": "/.*/", "dot_literal": ".",
        "slash_only": "/", "dollar": "/\\$\\{[a-z]+\\}/",
    }
    for k, r in regexes.items():
        add(f"regex_{k}", {**MESSY, "regex_replace_chars": r, "label_order": FULL_ORDER})
    add("regex_unicode", {"namespace": "caf\u00e9", "name": "\u00fcber \u4e2d\u6587", "environment": "dev"})
    add("regex_unicode_keep", {"namespace": "caf\u00e9", "name": "\u00fcber", "regex_replace_chars": "/[ ]/"})

    for lim in [0, 6, 7, 8, 9, 10, 11, 12, 15, 16, 17, 20, 25, 30, 50, 100]:
        add(f"limit_{lim}", {**RICH, "label_order": FULL_ORDER, "id_length_limit": lim})
    for lim in [16, 15, 17]:
        add(f"limit_edge_{lim}", {"namespace": "abc", "environment": "dev", "stage": "x", "name": "defghi", "id_length_limit": lim})
    for vc in ["upper", "title", "none", "lower"]:
        for lim in [6, 14, 20]:
            add(f"limit_{vc}_{lim}", {**MESSY, "label_value_case": vc, "id_length_limit": lim, "label_order": FULL_ORDER})
    for rk, rx in {"digits": "/[0-9]/", "hex": "/[0-9a-f]/", "letters_only": "/[^a-z]/"}.items():
        for lim in [8, 12, 14, 20, 30]:
            for vc in ["lower", "upper", "title", "none"]:
                add(f"limit_rx_{rk}_{vc}_{lim}", {**MESSY, "regex_replace_chars": rx, "label_value_case": vc,
                                                  "id_length_limit": lim, "label_order": FULL_ORDER})
    for vc in ["lower", "upper", "title", "none"]:
        for lim in [12, 15]:
            add(f"limit_suffix_{vc}_{lim}", {**MESSY, "regex_replace_chars": "/[0-9a-f]/", "label_value_case": vc,
                                              "delimiter": "~" * 10, "id_length_limit": lim, "label_order": FULL_ORDER})
    PLACE = "/[^-a-zA-Z0-9${}_]/"
    add("dns_seq_ns_name", {"regex_replace_chars": PLACE, "namespace": "${name}", "name": "${region}", "region": "usa",
                            "domain_name": "d.io", "dns_name_format": "${namespace}.${name}.${region}"})
    add("dns_seq_id_late", {"regex_replace_chars": PLACE, "namespace": "ns", "name": "${id}", "stage": "${domain_name}",
                            "domain_name": "d.io", "dns_name_format": "${name}|${stage}|${id}"})
    add("dns_seq_region_twice", {"regex_replace_chars": PLACE, "environment": "${region}", "region": "usa", "name": "n",
                                 "domain_name": "d.io", "dns_name_format": "${environment}-${name}"})
    add("dns_seq_attrs", {"regex_replace_chars": PLACE, "name": "n", "attributes": ["${domain_name}", "${id}"],
                          "domain_name": "d.io", "dns_name_format": "${attributes}"})
    add("dns_seq_tenant_project", {"regex_replace_chars": PLACE, "tenant": "${project}", "project": "${region}", "region": "r1",
                                   "name": "n", "dns_name_format": "${tenant}/${project}/${region}"})
    add("dns_seq_ns_tenant", {"regex_replace_chars": PLACE, "namespace": "${tenant}", "tenant": "t1", "name": "n",
                              "dns_name_format": "${namespace}"})
    add("dns_seq_tenant_ns", {"regex_replace_chars": PLACE, "tenant": "${namespace}", "namespace": "n1", "name": "n",
                              "dns_name_format": "${tenant}"})
    add("regex_double_slash", {"name": "a//b", "namespace": "x//y", "regex_replace_chars": "//"})
    add("regex_single_slash_literal", {"name": "a/b/c", "regex_replace_chars": "/"})
    # Terraform NFC-normalizes every string and measures length by grapheme cluster.
    add("nfc_decomposed_default", {"namespace": "e\u0301cole", "name": "o\u0323\u0302x", "environment": "a\u0308", "stage": "n\u0303"})
    for vc in ["lower", "upper", "title", "none"]:
        add(f"nfc_keep_{vc}", {"namespace": "e\u0301cole", "name": "o\u0323\u0302X\u0131\u0307", "attributes": ["a\u0308", "\u00e4"],
                               "regex_replace_chars": "/x^/", "label_value_case": vc})
    GR = "a\u0323\u0301bcdefghij\U0001F468\u200D\U0001F469\u200D\U0001F467klmnop"
    for dl in ["-", "\U0001F600", "e\u0301", "~~~"]:
        for lim in [8, 10, 12, 14, 20]:
            add(f"limit_grapheme_{abs(hash(dl)) % 997}_{lim}".replace("-", "m"), {"name": GR, "namespace": "ns", "regex_replace_chars": "/x^/",
                                                    "delimiter": dl, "id_length_limit": lim})
    add("limit_nodelim", {**RICH, "delimiter": "", "id_length_limit": 12, "label_order": FULL_ORDER})
    add("limit_bigdelim", {**RICH, "delimiter": "____", "id_length_limit": 14, "label_order": FULL_ORDER})
    add("limit_trailing_delim", {"namespace": "ab", "environment": "cd", "stage": "ef", "name": "gh", "id_length_limit": 9})

    for e in [True, False, None]:
        add(f"enabled_{e}", {**RICH, "enabled": e})

    atts = {
        "simple": ["a", "b"], "dup": ["a", "a", "b"], "empty_item": ["", "a"], "messy": ["A B", "c_d"], "none": [],
        "all_empty": ["", ""], "case_dup": ["A", "a"],
    }
    for k, a in atts.items():
        add(f"attrs_{k}", {**BASIC, "attributes": a})
    add("attrs_ctx_and_var", {**BASIC, "attributes": ["dup", "var1"]}, ctx(attributes=["ctx1", "dup"], namespace="ctxns"))
    add("attrs_ctx_only", {"name": "n"}, ctx(attributes=["only"], namespace="ns", environment="e", stage="s"))

    add("tags_var", {**BASIC, "tags": {"A": "1", "b": "2"}})
    add("tags_ctx", BASIC, ctx(tags={"C": "ctx"}))
    add("tags_overlap", {**BASIC, "tags": {"K": "var"}}, ctx(tags={"K": "ctx", "Z": "z"}))
    add("tags_override_generated", {**RICH, "tags": {"Name": "custom", "Namespace": "ovr"}})
    add("tags_none", BASIC)

    for k, l in {"default": ["default"], "empty": [], "ns": ["namespace"], "two": ["name", "environment"],
                 "attr": ["attributes"], "bogus": ["bogus"], "dup": ["name", "name"], "all": FULL_ORDER}.items():
        add(f"lat_{k}", {**RICH, "labels_as_tags": l})
    add("lat_ctx_set", {**RICH, "labels_as_tags": ["name"]}, ctx(labels_as_tags=["namespace"]))
    add("lat_ctx_unset_var_empty", {**RICH, "labels_as_tags": []}, ctx(labels_as_tags=["unset"]))
    add("lat_ctx_default_var_set", {**RICH, "labels_as_tags": ["name"]}, ctx(labels_as_tags=["default"]))
    add("lat_ctx_empty", RICH, ctx(labels_as_tags=[]))
    add("lat_var_unsorted", {**RICH, "labels_as_tags": ["stage", "name", "environment"]})
    add("lat_ctx_unsorted", RICH, ctx(labels_as_tags=["stage", "name", "environment"]))
    add("lat_ctx_dups", RICH, ctx(labels_as_tags=["name", "name", "stage"]))
    add("lat_ctx_unset_among", RICH, ctx(labels_as_tags=["name", "unset"]))
    add("lat_key_lower", {**RICH, "label_key_case": "lower"})
    add("lat_key_upper", {**RICH, "label_key_case": "upper"})

    add("atm_basic", {**BASIC, "tags": {"a": "1"}, "additional_tag_map": {"propagate_at_launch": "true"}})
    add("atm_multi", {**RICH, "additional_tag_map": {"p": "1", "q": "2"}})
    add("atm_override_key", {**BASIC, "tags": {"a": "1"}, "additional_tag_map": {"key": "x", "value": "y"}})
    add("atm_ctx_merge", BASIC, ctx(additional_tag_map={"c": "1"}))
    add("atm_ctx_var_overlap", {**BASIC, "additional_tag_map": {"c": "var"}}, ctx(additional_tag_map={"c": "ctx", "d": "2"}))
    add("atm_disabled", {**BASIC, "enabled": False, "tags": {"a": "1"}, "additional_tag_map": {"p": "1"}})

    df = {
        "one": {"acct": {"format": "%v-%v", "labels": ["namespace", "name"]}},
        "s_verb": {"x": {"format": "%s/%s", "labels": ["environment", "stage"]}},
        "two": {"a": {"format": "%v", "labels": ["name"]}, "b": {"format": "%s.%s.%s", "labels": ["namespace", "environment", "name"]}},
        "attr": {"a": {"format": "%v:%v", "labels": ["name", "attributes"]}},
        "literal": {"a": {"format": "prefix-%s-suffix", "labels": ["name"]}},
        "percent": {"a": {"format": "100%%-%s", "labels": ["name"]}},
        "quoted": {"a": {"format": "%q", "labels": ["name"]}},
        "empty_label": {"a": {"format": "[%s][%s]", "labels": ["tenant", "name"]}},
        "pascal": {"a": {"format": "%s%s", "labels": ["namespace", "name"]}},
    }
    for k, d in df.items():
        add(f"desc_{k}", {**RICH, "descriptor_formats": d})
    # %q is JSON-style quoting in Terraform, not Go's strconv.Quote. Needs a regex that keeps odd characters.
    ODD = "a\u200db<c>&d\u2028e\u007ff\u0001g\"h\\i\tj\u00a0k\ue000l\u00adm"
    for k, fm in {"q": "%q", "s": "%s", "mixed": "%q|%s|%v", "width": "%-12q|%5s"}.items():
        labs = ["name"] * (fm.count("%") - fm.count("%%"))
        add(f"desc_odd_{k}", {"name": ODD, "regex_replace_chars": "/x^/", "descriptor_formats": {"a": {"format": fm, "labels": labs}}})
    add("desc_upper", {**MESSY, "label_value_case": "upper", "descriptor_formats": df["two"]})
    # width/precision/flags count grapheme clusters; precision truncates before %q quotes; width pads the quoted text.
    CMB = "a\u0323\u0301x\U0001F468\u200D\U0001F469\u200D\U0001F467yz"
    for k, fm in {"w5": "[%5s]", "w_left": "[%-5s]", "w_zero": "[%05s]", "p2": "[%.2s]", "w6p2": "[%6.2s]", "q8": "[%8q]",
                  "q_left": "[%-8q]", "q_p3": "[%.3q]", "plus": "[%+s]", "space": "[% s]", "hash": "[%#s]",
                  "v4": "[%4v]", "zero8": "[%08s]", "zero_left8": "[%-08s]", "w8": "[%8s]", "left8": "[%-8s]", "zero_left": "[%-05s]", "p0": "[%.s]", "w_big": "[%20s]"}.items():
        add(f"desc_fmt_{k}", {"name": CMB, "regex_replace_chars": "/x^/", "descriptor_formats": {"a": {"format": fm, "labels": ["name"]}}})
        add(f"desc_fmt_ascii_{k}", {"name": "abcdef", "descriptor_formats": {"a": {"format": fm, "labels": ["name"]}}})
    add("desc_ctx_var_merge", {**BASIC, "descriptor_formats": {"v": {"format": "%v", "labels": ["name"]}}},
        ctx(descriptor_formats={"c": {"format": "%v", "labels": ["namespace"]}, "v": {"format": "%v!", "labels": ["name"]}}))

    dns = {
        "default": None, "name_domain": "${name}.${domain_name}", "id": "${id}.${domain_name}",
        "all": "${namespace}|${tenant}|${project}|${region}|${environment}|${stage}|${name}|${attributes}|${id}|${domain_name}",
        "region_twice": "${region}.${region}.${domain_name}", "literal": "static.example.com", "empty": "",
        "unknown_placeholder": "${nope}.${name}",
    }
    for k, f in dns.items():
        add(f"dns_{k}", {**RICH, "dns_name_format": f})
    add("dns_no_domain", {**RICH, "domain_name": None})
    add("dns_domain_only", {"domain_name": "x.com", "name": "svc", "namespace": "ns"})
    add("dns_disabled", {**RICH, "enabled": False})
    add("dns_ctx", {"name": "svc"}, ctx(domain_name="ctx.io", dns_name_format="${name}.${domain_name}", namespace="n"))
    add("dns_var_over_ctx", {"name": "svc", "domain_name": "var.io"}, ctx(domain_name="ctx.io", dns_name_format="${name}.${domain_name}"))

    full_ctx = ctx(namespace="cns", tenant="ct", project="cp", region="cr", environment="ce", stage="cs", name="cn",
                   delimiter="_", attributes=["ca"], tags={"CT": "1"}, label_order=FULL_ORDER, id_length_limit=0,
                   label_key_case="lower", label_value_case="upper", regex_replace_chars="/[^-a-zA-Z0-9]/",
                   labels_as_tags=["default"], domain_name="c.io", dns_name_format="${name}.${domain_name}")
    add("ctx_full_only", {}, full_ctx)
    add("ctx_full_var_override", {"namespace": "vns", "name": "vn", "delimiter": ".", "label_value_case": "lower", "label_key_case": "title"}, full_ctx)
    old_shape = {k: v for k, v in full_ctx.items() if k not in ("project", "region")}
    add("ctx_old_no_project_region", {"name": "x"}, old_shape)
    minimal = {k: v for k, v in full_ctx.items() if k not in ("tenant", "project", "region", "label_key_case", "label_value_case", "descriptor_formats", "labels_as_tags")}
    add("ctx_minimal_keys", {"name": "x"}, minimal)
    add("ctx_minimal_keys_lat", {"name": "x", "labels_as_tags": ["name"]}, minimal)
    add("ctx_null_values_var_fill", {"namespace": "a", "environment": "b", "stage": "c", "name": "d"}, ctx())
    add("ctx_disabled", {"name": "x"}, ctx(enabled=False, namespace="a"))
    add("ctx_disabled_var_enable", {"name": "x", "enabled": True}, ctx(enabled=False, namespace="a"))
    add("ctx_enabled_var_disable", {"name": "x", "enabled": False}, ctx(namespace="a"))

    def ch(name, frm, via, inputs=None, store=chained):
        assert name not in store
        store[name] = {"from": frm, "via": via, "inputs": inputs or {}}

    for src in ["proj_order", "proj_order_limit", "rich_full_order", "basic", "limit_edge_15", "case_title_nodelim", "lat_two", "desc_two", "tags_override_generated"]:
        for via in ["normalized_context", "self"]:
            ch(f"chain_{src}_{via}", src, via, {"name": "child", "attributes": ["x"], "tags": {"K": "v"}})
    for via in ["normalized_context", "self"]:
        ch(f"chain_noop_{via}", "rich_full_order", via)
        ch(f"chain_lat_ignored_{via}", "lat_two", via, {"labels_as_tags": ["namespace"]})
        ch(f"chain_lat_default_{via}", "rich_full_order", via, {"labels_as_tags": ["name"]})
        ch(f"chain_override_all_{via}", "rich_full_order", via, {
            "namespace": "ovr", "environment": "ovrenv", "delimiter": "_", "label_value_case": "upper",
            "label_key_case": "lower", "id_length_limit": 12, "regex_replace_chars": "/[^a-z]/", "domain_name": "o.io"})
        ch(f"chain_order_override_{via}", "rich_full_order", via, {"label_order": ["name", "attributes"]})
        ch(f"chain_disabled_{via}", "enabled_False", via, {"name": "child"})
        ch(f"chain_enable_{via}", "enabled_False", via, {"enabled": True, "name": "child"})
        ch(f"chain_atm_{via}", "atm_multi", via, {"additional_tag_map": {"r": "3"}})
        ch(f"chain_dns_{via}", "dns_all", via, {"name": "child"})
        ch(f"chain_ctx_old_{via}", "ctx_old_no_project_region", via, {"name": "child"})
        ch(f"chain_ctx_minimal_{via}", "ctx_minimal_keys", via, {"name": "child"})
        ch(f"chain_limit_{via}", "limit_20", via, {"name": "a-much-longer-child-name"})
        ch(f"chain_limit_title_{via}", "limit_title_14", via, {"attributes": ["more"]})
    for via in ["normalized_context", "self"]:
        ch(f"chain_projorder_child_{via}", "rich", via, {"label_order": PROJ_ORDER, "name": "child"})
        ch(f"chain_projorder_leaf_{via}", f"chain_proj_order_{via}", via, {"attributes": ["leaf"]}, store=chained2)
    ch("chain_lat_ctx_unsorted_self", "lat_ctx_unsorted", "self", {"name": "c"})
    ch("chain_lat_ctx_unsorted_norm", "lat_ctx_unsorted", "normalized_context", {"name": "c"})
    for via in ["normalized_context", "self"]:
        ch(f"chain2_{via}", "chain_rich_full_order_normalized_context", via, {"attributes": ["y"], "name": "grand"}, store=chained2)
        ch(f"chain2_self_{via}", "chain_rich_full_order_self", via, {"attributes": ["y"]}, store=chained2)
        ch(f"chain2_lat_{via}", "chain_lat_two_self", via, {"labels_as_tags": ["stage"]}, store=chained2)

    neg = {
        "bad_label_key_case": ({"name": "x", "label_key_case": "foo"}, None, "Allowed values"),
        "bad_label_value_case": ({"name": "x", "label_value_case": "bar"}, None, "Allowed values"),
        "bad_id_length_limit_3": ({"name": "x", "id_length_limit": 3}, None, "id_length_limit must be"),
        "bad_label_order_item": ({"name": "x", "label_order": ["bogus"]}, None, "Invalid index"),
        "bad_regex": ({"name": "x", "regex_replace_chars": "/[/"}, None, "error parsing regexp"),
        "bad_ctx_label_key_case": ({"name": "x"}, ctx(label_key_case="foo"), "Allowed values"),
        "bad_descriptor_too_few_args": ({"name": "x", "descriptor_formats": {"a": {"format": "%s-%s", "labels": ["name"]}}}, None, "format"),
        "bad_descriptor_too_many_args": ({"name": "x", "descriptor_formats": {"a": {"format": "%s", "labels": ["name", "namespace"]}}}, None, "format"),
        "bad_descriptor_star_width": ({"name": "x", "descriptor_formats": {"a": {"format": "%*s", "labels": ["name"]}}}, None, "format"),
        "bad_descriptor_verb_d": ({"name": "x", "descriptor_formats": {"a": {"format": "%d", "labels": ["name"]}}}, None, "format"),
        "bad_descriptor_trailing_pct": ({"name": "x", "descriptor_formats": {"a": {"format": "abc%", "labels": ["name"]}}}, None, "format"),
        "bad_descriptor_label": ({"name": "x", "descriptor_formats": {"a": {"format": "%s", "labels": ["bogus"]}}}, None, "Invalid index"),
    }
    for k, (inp, c, err) in neg.items():
        negative[k] = {"inputs": inp, "error_contains": err}
        if c is not None:
            negative[k]["context"] = c
    return base, chained, chained2, negative

if __name__ == "__main__":
    b, c, c2, n = build()
    print(len(b), len(c), len(c2), len(n))
