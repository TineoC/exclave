# exclave — every task runs the same way on a laptop and on a runner.

quickstart := "examples/quickstart"
contracts  := "examples/contracts"

default:
    @just --list --unsorted

# Build the resolver.
build:
    go build -o bin/exclave ./cmd/exclave

# Unit tests, including the constraint explanation strings.
test:
    go test ./...

# Everything CI checks, before you push.
check: test
    go vet ./...
    gofmt -l . | (! grep .) || (echo "run: just fmt" && exit 1)
    just validate
    just lint-charts
    just contract
    just landing-zone

fmt:
    gofmt -w .

# Structural check of the example catalog and fleet.
validate:
    go run ./cmd/exclave validate \
      -catalog {{quickstart}}/catalog \
      -fleet {{quickstart}}/fleet/environments
    go run ./cmd/exclave validate \
      -catalog {{contracts}}/catalog \
      -fleet {{contracts}}/fleet/contracts

# Which release does each environment get, and why not the newer ones?
plan:
    @go run ./cmd/exclave plan \
      -catalog {{quickstart}}/catalog \
      -fleet {{quickstart}}/fleet/environments

# Every constraint tested against one environment.
explain env version="":
    @go run ./cmd/exclave explain {{env}} {{version}} \
      -catalog {{quickstart}}/catalog \
      -fleet {{quickstart}}/fleet/environments

# Depth-first: svc must vendor `common` before product packages svc.
# The contract portfolio: which contract may take which baseline, and why not.
portfolio:
    @go run ./cmd/exclave plan \
      -catalog {{contracts}}/catalog \
      -fleet {{contracts}}/fleet/contracts

# One contract, one baseline version, check by check.
explain-contract contract version="":
    @go run ./cmd/exclave explain {{contract}} {{version}} \
      -catalog {{contracts}}/catalog \
      -fleet {{contracts}}/fleet/contracts

lint-charts:
    helm dependency update {{quickstart}}/charts/svc
    helm dependency update {{quickstart}}/charts/product
    helm lint {{quickstart}}/charts/product

# Assert the site-values contract: bad input rejected, every platform seam renders.
contract:
    ./ci/contract-check.sh

# Assert the landing-zone interface is identical across every provider.
landing-zone:
    ./ci/landing-zone-check.sh

# All four planes against a real cluster. Needs docker, kind and kubectl.
demo:
    {{quickstart}}/demo.sh

demo-clean:
    {{quickstart}}/demo-clean.sh
