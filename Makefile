ANY-EXPORTER=./any-exporter
IMAGE_NAME ?= ghcr.io/peng225/any-exporter

GO_FILES:=$(shell find . -type f -name '*.go' -print)

$(ANY-EXPORTER): $(GO_FILES)
	CGO_ENABLED=0 go build -o $@ -v

.PHONY: test
test: $(ANY-EXPORTER)
	go test -v `go list ./... | grep -v e2e`

.PHONY: clean
clean:
	rm -f $(ANY-EXPORTER)
