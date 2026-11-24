# CLAUDE.md - guidance for Claude working in this repo

## What this is
A Terraform provider (`null-context`, same name as the module it replaces) that must reproduce the `null-context` module's outputs **exactly**.
Correctness == parity with `testdata/vectors.json`. Read `docs/PARITY.md` and `docs/DESIGN.md` first, then `SESSION.md` for current state.

## Golden rules
1. **Parity beats elegance.** Do not "fix" module quirks in `internal/label`; a changed ID replaces real resources. Fixes ship as separate, versioned, opt-in changes.
2. **The module is the spec.** Never hand-edit `testdata/vectors.json`. Change `tools/vectors/cases.py`, run `make vectors`.
3. **Every behaviour change needs a vector** and a mutation in `tools/mutation/mutate.py` that the vectors kill.
4. **Run `make verify` and `make mutation` before declaring anything done.** Report real numbers, never "should pass".
5. Keep `internal/label` free of Terraform imports. Conversion code lives in `internal/provider/convert.go`.
6. Do not weaken, skip or delete failing parity tests to get green. A failing vector is a finding: explain it in `docs/PARITY.md`.

## Commands
```sh
make tidy | fmt | vet | build
make vectors REF_MODULE=../null-context      # needs terraform + python3
make test                                    # TF_ACC_TERRAFORM_PATH=/path/to/terraform if not on PATH
make provider-parity                         # all vectors through built provider (data source + function)
make mutation                                # 32/32 must be killed
make docs                                    # regenerate registry docs after any schema change
make release-check                           # validate .goreleaser.yml (needs goreleaser + a git remote)
```

## Layout
`internal/label` core | `internal/provider` framework wrapper | `tools/vectors` generator + provider checker | `tools/mutation` mutation check | `docs/` | `examples/`

## Gotchas learned the hard way
- Terraform `replace()` treats `/.../` as RE2 regex, otherwise literal; `coalesce` skips `""`; `coalescelist` skips `[]`.
- `labels_as_tags`: first-set-wins; set conversion sorts and de-duplicates in **both** branches.
- `normalized_context` has no `project`/`region`; `legacy` has no `domain_name`/`dns_name_format`; neither is a valid parent for the other's consumers.
- dns placeholders are replaced sequentially in a fixed order (region twice). Only distinguishable with a regex that keeps `${}`.
- `hashSuffix` only shows when the regex strips md5 hex chars and the prefix is empty: needs long delimiter + small limit.
- Mutation scripts must restore **every** source file between mutations (an earlier run left `json.go` mutated and produced false kills).
- `resp.Name = req.ProviderTypeName` makes the data source type exactly `null-context` (`data "null-context" "x" {}`).
- Single-line HCL blocks accept one argument only (test configs).
- Writing to the user's machine goes through the Filesystem MCP tool; it can hang on an approval prompt. Do work in the sandbox and deliver a zip if it does.
- Full goreleaser snapshot builds take minutes: validate with `goreleaser check` and a reduced `goos/goarch` config, run long builds in the background.
- `goreleaser check` needs a git remote (`git remote add origin ...`) or it reports an invalid-config error that is really an environment issue.

## Conventions
- Go files carry `Copyright 2026 Raring Group LLC` + SPDX; `internal/label` also credits Cloud Posse.
- Docs: sentence case, concise, facts with numbers. Update `SESSION.md` at the end of every working session (what changed, results, open items).
- **This repo is destined to be public** (Terraform Registry). Never put client names, internal hostnames, local paths, tenant/account IDs or secrets in any file. Example namespace is `acme`; `7pi`/SevenPico appear only as upstream attribution and fixtures.
- Registry docs (`docs/index.md`, `docs/data-sources/`, `docs/functions/`) are generated; edit the schema descriptions or `examples/`, then `make docs`. `docs/DESIGN.md`, `PARITY.md`, `RELEASE.md` are hand-written.

## Definition of done for a change
vet clean, `make test` green, `make provider-parity` = 0 mismatches, `make mutation` = all killed, docs updated, `SESSION.md` entry added.
