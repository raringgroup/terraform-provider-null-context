# terraform-provider-null-context

A Terraform provider that replaces the [`null-context`](https://github.com/raringgroup/terraform-null-context)
module (a fork of Cloud Posse's `terraform-null-label`). Instead of every module downloading and
instantiating `module "context" { source = ".../null-context" }`, Terraform downloads one cached
provider and each module calls `data "null-context" "this" {}` or `provider::null-context::label()`.

> **Status:** experimental. Phases 0-3 (reference freeze, golden vectors, Go core, provider wrapper)
> are done and verified. Nothing is published, migrated or released yet. See [SESSION.md](SESSION.md).

## Requirements

| | Supported |
|---|---|
| Terraform | **>= 1.8.0** (tested on 1.8.0 and 1.17.2) |
| OpenTofu | >= 1.7.0 (same feature set; **not yet tested**, and published separately to the OpenTofu registry) |
| Registry | Public Terraform Registry: `registry.terraform.io/raringgroup/null-context` |

Modules that use the provider should declare `required_version = ">= 1.8.0"` and `required_providers { null-context = { source = "raringgroup/null-context", version = "~> 0.1" } }`.

## Why

- `terraform init` fetches null-context once per module instance today; a provider is fetched once per version.
- Shared defaults (`namespace`, `environment`, `stage`, base `tags`) can live in one `provider "null-context" {}` block.
- The naming logic lives in one tested Go library instead of HCL copied through every module.

It does **not** remove the need to pass a context from a parent module to its children: per-instance values
(`name`, `attributes`) still travel as `var.context`. See [docs/DESIGN.md](docs/DESIGN.md).

## Usage

```hcl
terraform {
  required_providers { null-context = { source = "raringgroup/null-context" } }
}

provider "null-context" {            # optional shared defaults
  namespace   = "acme"
  environment = "dev"
  stage       = "app"
  tags        = { Owner = "platform" }
}

data "null-context" "vpc" {
  name = "My VPC"               # returned normalized: data.null-context.vpc.name == "myvpc"
}

data "null-context" "sg" {
  context    = data.null-context.vpc.self   # chain from a parent
  attributes = ["sg"]
}

resource "aws_security_group" "this" {
  name = data.null-context.sg.id            # acme-dev-app-myvpc-sg
  tags = data.null-context.sg.tags
}
```

Function form (Terraform >= 1.8, OpenTofu >= 1.7), pure and usable where a data source's deferred read is a problem:

```hcl
locals {
  root  = provider::null-context::label(null, { namespace = "acme", name = "a" })
  child = provider::null-context::label(local.root.self, { attributes = ["x"] })   # acme-a-x
}
```

### Examples

| Example | Shows | Verified |
|---|---|---|
| [`examples/basic`](examples/basic) | Provider defaults, chaining, function | applied |
| [`examples/module`](examples/module) | A root with provider config and a `data "null-context"` calling child modules that take a `context` variable, chain it onward (`output "context"`), derive sub-labels, and fall back to provider defaults when `context` is null | applied (`terraform_data` stand-ins, no cloud needed) |
| [`examples/aws`](examples/aws) | Real AWS resources (`aws_s3_bucket`, `aws_sqs_queue`, `aws_iam_role`) named and tagged from the provider; `descriptor_formats`, `id_length_limit`, `tags_as_list_of_maps` | `terraform validate` passes; label values evaluated; not applied (needs credentials) |

The module pattern in one picture: a module takes `variable "context" { type = any, default = null }`, calls
`data "null-context" "this" { context = var.context  name = var.name }`, uses `.id`/`.tags` on its resources and exposes
`output "context" { value = data.null-context.this.self }` for its own children.

### Rules worth knowing

- A data source's own `context` **replaces** the provider-level defaults entirely. Provider defaults apply only when `context` is not set (or null).
- Inputs that share a name with an output (`name`, `tags`, `attributes`, ...) are merged with the parent and returned normalized, exactly like the module's variables and outputs.
- **If `project` is in your `label_order`, always pass `self`** (never `normalized_context`) to children, otherwise the child's ID silently loses `project` (see [docs/PARITY.md](docs/PARITY.md)).
- Pass `self` (not `normalized_context`) to children: `normalized_context` drops `project`/`region` and carries the parent's generated `Name` tag downward (identical to the module; see [docs/PARITY.md](docs/PARITY.md)).
- `legacy` is for Cloud Posse modules; it cannot be fed back into this provider.
- `labels_as_tags` is first-set-wins along a chain.

### Output mapping (module -> provider)

| Module output | Data source / function |
|---|---|
| `id`, `id_full`, `dns_name`, `tags_as_list_of_maps`, `descriptors` | same names |
| `enabled`, `namespace`, `tenant`, `environment`, `stage`, `name`, `delimiter`, `attributes`, `tags`, `additional_tag_map`, `label_order`, `regex_replace_chars`, `id_length_limit`, `dns_name_format`, `domain_name` | same names (Optional+Computed in the data source: input is merged, output is normalized) |
| `normalized_context`, `legacy`, `self` | same names (dynamic objects) |

All 23 module outputs are produced.

## Develop

Requirements: Go (see `go.mod`), Terraform >= 1.8 on `PATH`, Python 3, and a checkout of the reference module
(default `../null-context`, override with `REF_MODULE=`). The reference checkout is only needed to regenerate vectors; `testdata/vectors.json` is committed.

```sh
make tidy              # resolve modules, writes go.sum
make vectors           # regenerate golden vectors by running the REFERENCE MODULE through Terraform
make test              # Go tests: core parity vs vectors + provider tests driving a real terraform CLI
make provider-parity   # every vector through the built provider (data source + function)
make mutation          # 42 mutations of the core; every one must make tests fail
make fuzz              # differential fuzzing vs the reference module (random inputs incl. Unicode)
make fuzz-provider     # replay the fuzz files through the built provider
make verify            # vet + test + provider-parity
```

Try the provider locally without installing: build with `make build`, then point a CLI config at `./bin`:

```hcl
# ~/.terraformrc (or TF_CLI_CONFIG_FILE)
provider_installation {
  dev_overrides { "registry.terraform.io/raringgroup/null-context" = "/absolute/path/to/this/repo/bin" }
  direct {}
}
```

## Layout

```
internal/label/       Pure-Go port of the module (no Terraform deps). The part that must stay byte-for-byte compatible.
internal/provider/    terraform-plugin-framework wrapper: provider config, data source, function, value converters.
tools/vectors/        cases.py (inputs), gen.py (runs the reference module -> testdata/vectors.json), check_provider.py.
tools/mutation/       mutate.py - proves the vectors actually guard each behaviour.
testdata/             vectors.json (generated; 361 vectors + 12 error cases).
docs/                 PARITY.md (behaviours + quirks), DESIGN.md (decisions, migration sketch, open questions).
examples/basic/       Minimal usage.
examples/module/      Root + child modules chaining a context (runs without cloud credentials).
examples/aws/         AWS resources named and tagged from the provider.
```

## Parity guarantee

`vectors.json` is produced by running the real module, so it is the specification. The reference is
`null-context` `develop` @ `c9a2481` (code identical to release `v0.1.0`) and was cross-checked against the registry
module `SevenPico/context/null` `2.0.0` (commit `a753bc8`), which differs by one whitespace character.

## Known limits

- Behaviour was verified on Terraform 1.8.0 and 1.17.2 only. Unicode tables come from the Go toolchain used to build the provider; a Terraform built with a different Unicode version could differ for characters added in the newest Unicode release.
- `format()` supports string arguments with `%s %v %q %%`, the `-` and `0` flags, width and precision (counted in grapheme clusters, like Terraform); other verbs and `*` widths are errors, as in Terraform.
- Not yet published: see [docs/RELEASE.md](docs/RELEASE.md) for the checklist (GPG key, public GitHub repo, registry sign-up).

## License

Apache-2.0. Derived from Cloud Posse's terraform-null-label; see [NOTICE](NOTICE).
