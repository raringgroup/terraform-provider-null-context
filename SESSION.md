# SESSION.md - progress log

Newest entry first. Update at the end of every working session. Keep this file free of client names, local paths and IDs (the repo will be public).

## 2026-11-23 - session 3: provider renamed to `null-context`

**Decision:** the provider is named `null-context` (same name as the module it replaces).
- Address `registry.terraform.io/raringgroup/null-context`; repo and binary `terraform-provider-null-context`; Go module path `github.com/raringgroup/terraform-provider-null-context`.
- HCL: `required_providers { null-context = { source = "raringgroup/null-context" } }`, `data "null-context" "x" {}`, references `data.null-context.x.id`, function `provider::null-context::label(...)`.
- Hyphen is deliberate and verified: the framework accepts it and Terraform resolves the data source and function by that name. (An underscore name would make Terraform infer provider `null` from the type prefix.)
- Re-verified after the rename: `go vet`, Go tests, provider parity (**294 vectors x 2 entry points x 23 outputs, 0 mismatches**) on Terraform **1.17.2 and 1.8.0**, mutation **32/32**, `examples/basic` applies, docs regenerated with `--provider-name null-context`, no stale references to the old name.
- `.goreleaser.yml` now sets `project_name: terraform-provider-null-context` (`goreleaser check` re-run and passes; the multi-platform snapshot build was not repeated, only the name changed).

## 2026-11-23 - session 2: decisions, project-in-label_order, release scaffolding

**Answers received:** (1) minimum Terraform version is the maintainer's call; (2) `project` is in `label_order`, `region` is not; (3) public Terraform Registry.

### Decisions made
- **Baseline: Terraform >= 1.8.0, OpenTofu >= 1.7.0** (first versions with provider-defined functions; one rule for every module). Verified: provider acceptance tests and all vectors pass on Terraform **1.8.0** and **1.17.2**. OpenTofu **not tested**.
- **Registry: public Terraform Registry**, `registry.terraform.io/raringgroup/null-context` (address in `main.go` is now the real one).

### Finding that matters for the migration
With `project` in `label_order`, a child chained via `normalized_context` loses `project` in its ID (`7pi-dev-...` instead of `7pi-project1-dev-...`) and keeps the parent's `Name` tag; via `self` it is correct. The Go core and provider reproduce both exactly (vectors `proj_order*`, `chain_proj_order_*`, `chain_projorder_*`), so the provider itself changes no IDs. **Action for the maintainer:** audit modules for `normalized_context` chaining before migrating (see `docs/PARITY.md`, `docs/DESIGN.md` migration step 0).

### Work done
- Vectors: **294** (was 284) + 9 negative, regenerated with Terraform 1.17.2 from the reference module.
- Verification (all re-run after the changes): `go vet` clean; `go test` pass; provider parity **294 vectors x 2 entry points x 23 outputs, 0 mismatches**; mutation **32/32 killed**.
- Release scaffolding for the public registry: `terraform-registry-manifest.json` (protocol 6.0), `.goreleaser.yml`, `.github/workflows/{test,release}.yml`, `docs/RELEASE.md`.
  - Validated: `goreleaser check` passes; a snapshot release (linux/darwin x amd64/arm64) produced zips, `SHA256SUMS` and the manifest with registry-compliant names.
  - **Not exercised:** GPG signing, GitHub release, registry ingestion (need keys/accounts). The full 8-OS matrix snapshot was not completed in the sandbox (too slow); the config is the standard scaffold.
- Registry docs generated with `tfplugindocs` (`docs/index.md`, `docs/data-sources/context.md`, `docs/functions/label.md`) with examples under `examples/`; `make docs` regenerates them.
- Example namespace changed from a client-derived name to `acme` everywhere (code, tests, docs, vectors) because the repo will be public. One test hash was recomputed and independently checked with Python's md5 (`acme-dev-app-a-c1b3a`).
- Makefile: added `docs`, `release-check`.

### Open items
1. Audit modules for `normalized_context` chaining (live vs latent bug); decide per module: keep behaviour or switch to `self` (changes IDs, needs a plan review).
2. Provider defaults at the root: change the vendored `variable "context"` default to `null`, or keep passing an explicit context (see `docs/DESIGN.md`).
3. Before going public: confirm approval to publish the derived code; create the GitHub repo, GPG key and registry listing (`docs/RELEASE.md`).
4. Optional: OpenTofu registry + OpenTofu test run; later opt-in fix for the `normalized_context` quirks.
5. Phase 4 (plan diffs on real stacks) is still pending and is the real proof for production.

### Next steps (suggested)
1. Unzip, `make tidy`, `make verify`, `make mutation` locally.
2. Run the `normalized_context` audit.
3. Phase 4 on 2-3 representative stacks; then phase 5 (shim + leaf-module migration) and the first `v0.1.0` release.

## 2026-11-23 - session 1: phases 0-3 (reference freeze -> provider wrapper)

**Scope:** phases 0-3 (freeze, vectors, Go core, provider wrapper); phase 4 explicitly out of scope; document everything.
**Note:** Haiku sub-agents were not available in this chat; inputs were read directly (filesystem MCP + shell).

### Environment used (sandbox)
Go 1.27.4 (`go.mod` says 1.25.8), Terraform 1.17.2, terraform-plugin-framework v1.19.0, terraform-plugin-go v0.31.0, terraform-plugin-testing v1.16.0.

### Phase 0 - freeze the reference: DONE
- Reference = `null-context` repo, branch `develop` @ `c9a24810` (code identical to release `v0.1.0`; later commits are merges). Tag `v0.1.0` -> `90b05f6`, `main` -> `b85ce46`.
- The repo's only tag is `v0.1.0`. `2.0.0` is the **registry** version `SevenPico/context/null` that the vendored `exports/_context.tf` points at.
- Cross-check: registry `SevenPico/context/null` `2.0.0` (git `a753bc8`) vs the local reference: all five `.tf` files functionally identical; one whitespace difference in `main.tf`.
- Consumers chain through `module.context.self` (keeps `project`/`region`).

### Phase 1 - test vectors: DONE
- `tools/vectors/cases.py` defines cases; `gen.py` copies the reference module to a scratch dir, runs one `terraform apply` over a `for_each` harness (base, chained, chained2 groups) and runs negative cases one by one.
- Findings (all in `docs/PARITY.md`): `legacy` cannot be a parent for null-context; `normalized_context` chaining carries the parent's `Name` tag and drops `project`/`region`; `labels_as_tags` is always sorted and unique; disabled-gating applies to some outputs only; Terraform errors on `format()` argument-count mismatches.

### Phase 2 - Go core: DONE
- `internal/label` (`Compute`, `Context`, `Result`, `Outputs/Self/Legacy/NormalizedContext`, JSON parsing with Terraform null semantics).
- **Mutation testing** (`tools/mutation/mutate.py`): the first vector set let 3 mutants survive (hash suffix beyond 5 chars, dns placeholder order, `//` regex and format-arg count); vectors were added until none survived. An early mutation script did not restore `json.go` between runs and produced false kills; fixed (all files are restored).

### Phase 3 - provider wrapper: DONE
- Provider `null-context`: shared defaults, `data "null-context"` (type name equals provider name), `provider::null-context::label(context, vars)`.
- Terraform accepted Optional+Computed attributes whose returned value differs from config.
- `tools/vectors/check_provider.py` runs every vector through the built provider under real Terraform via dev_overrides. A tampered-vector control produced the expected mismatches.
- Go acceptance tests (`internal/provider/provider_test.go`) drive a real Terraform CLI and skip when none is found.

### Known limits
- Grapheme-cluster semantics for `length`/`substr` (the port uses runes); `format()` limited to string args (see `docs/PARITY.md`).
