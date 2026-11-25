.PHONY: build vet test race lint generate generate-check verify

GOYACC := go -C cmd/yaccspan run golang.org/x/tools/cmd/goyacc@v0.52.0

build:
	go build ./...

vet:
	go vet ./...

test:
	go test -count=1 ./...

race:
	go test -race -shuffle=on -count=1 ./...

lint:
	golangci-lint run ./...

generate:
	$(GOYACC) -l -p yy -o testdata/reduction.go.txt -v /dev/null testdata/reduction.y

generate-check:
	@set -eu; \
	dir=$$(mktemp -d); \
	trap 'rm -f "$$dir/parser.go" "$$dir/expected" "$$dir/actual"; rmdir "$$dir"' EXIT HUP INT TERM; \
	$(GOYACC) -l -p yy -o "$$dir/parser.go" -v /dev/null testdata/reduction.y; \
	sed '1d' cmd/yaccspan/testdata/reduction.go.txt >"$$dir/expected"; \
	sed '1d' "$$dir/parser.go" >"$$dir/actual"; \
	cmp "$$dir/expected" "$$dir/actual"

verify: build vet test race lint generate-check
