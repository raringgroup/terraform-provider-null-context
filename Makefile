.PHONY: build test vectors provider-parity mutation verify fmt vet tidy docs release-check fuzz fuzz-provider

# Path to the reference module (a checkout of null-context). Override with REF_MODULE=...
REF_MODULE ?= ../null-context
TERRAFORM   ?= terraform

build:
	go build -o bin/terraform-provider-null-context .

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

# Regenerate golden vectors by running the REFERENCE MODULE (needs terraform + python3).
vectors:
	python3 tools/vectors/gen.py --ref-module $(REF_MODULE) --out testdata/vectors.json --terraform $(TERRAFORM)

# Go unit tests: core parity vs vectors + provider tests that drive a real terraform CLI.
test:
	go test ./...

# Run every vector through the built provider under real Terraform (data source + function).
provider-parity: build
	python3 tools/vectors/check_provider.py --vectors testdata/vectors.json --provider-dir ./bin --terraform $(TERRAFORM)

# Prove the tests catch regressions: every mutation of the core must make tests fail.
mutation:
	python3 tools/mutation/mutate.py

verify: vet test provider-parity

# Regenerate registry docs (docs/index.md, docs/data-sources, docs/functions) from the schemas.
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate --provider-name null-context

# Validate .goreleaser.yml (needs goreleaser and a git remote).
release-check:
	goreleaser check

# Differential fuzzing: random valid inputs (incl. Unicode) through the REFERENCE module, replayed against the Go core.
# Generated files (testdata/fuzz-*.json) are git-ignored. Needs the reference module checkout (REF_MODULE) and terraform.
FUZZ_SEEDS ?= 1 2 3
fuzz:
	for s in $(FUZZ_SEEDS); do python3 tools/vectors/fuzz.py --ref-module $(REF_MODULE) --seed $$s --base 150 --chained 150 --chained2 30 --terraform $(TERRAFORM) || exit 1; done
	go test -count=1 -run TestFuzzParity ./internal/label/

# Replay every generated fuzz file through the built provider (data source + function) under real Terraform.
fuzz-provider: build
	for f in testdata/fuzz-*.json; do python3 tools/vectors/check_provider.py --vectors $$f --provider-dir ./bin --terraform $(TERRAFORM) || exit 1; done
