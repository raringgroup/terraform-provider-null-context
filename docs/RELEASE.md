# Release checklist: public Terraform Registry

Target address: `registry.terraform.io/raringgroup/null-context`. Scaffolding is validated except signing and publishing
(they need your GPG key and GitHub; `goreleaser check` and a snapshot build of the archives, `SHA256SUMS` and manifest pass).

## One-time setup

1. **GitHub**: public repo `raringgroup/terraform-provider-null-context` (the name must be exactly `terraform-provider-<NAME>`).
2. **Before making it public**: confirm no client names, internal hostnames or local paths are in the repo (`CLAUDE.md` says never to add them) and that publishing the Apache-2.0 derived code is approved (see `NOTICE`).
3. **GPG key** for signing: `gpg --full-generate-key` (RSA 4096, no expiry or long expiry). Export:
   - private key -> repo secret `GPG_PRIVATE_KEY` (`gpg --armor --export-secret-keys <ID>`), passphrase -> `GPG_PASSPHRASE`;
   - public key (`gpg --armor --export <ID>`) -> registry.
4. **Registry**: sign in at registry.terraform.io with the GitHub account that owns the `raringgroup` org (needs org owner/admin), *Publish -> Provider*, choose the repo, add the public GPG key.
5. Commit `testdata/vectors.json` (it is the test spec; CI runs `make verify` without the reference module).

## Every release

1. `make verify && make mutation` green; `make docs` run and committed; `SESSION.md` updated.
2. Tag: `git tag v0.1.0 && git push origin v0.1.0` (semver, `v` prefix).
3. `.github/workflows/release.yml` runs goreleaser: zips per OS/arch, `SHA256SUMS`, detached `.sig`, manifest (protocol 6.0).
4. The registry ingests the GitHub release via webhook (a few minutes). Verify with a clean `terraform init` in `examples/basic`.

## Versioning rules

- Any change that alters an ID or tag for the same inputs is a **breaking change** (major once >= 1.0; minor while 0.x, called out loudly).
- Pin consumers with `version = "~> 0.1"` until 1.0.

## Not covered here

OpenTofu registry publication (separate registry, separate sign-up); signing/publishing were not exercised in the sandbox.
