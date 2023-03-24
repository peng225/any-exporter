PROMBLOCK=./promblock
IMAGE_NAME ?= ghcr.io/peng225/promblock

GO_FILES:=$(shell find . -type f -name '*.go' -print)

$(PROMBLOCK): $(GO_FILES)
	CGO_ENABLED=0 go build -o $@ -v

.PHONY: image
image:
	docker build . --file Dockerfile --tag $(IMAGE_NAME)

.PHONY: test
test: $(PROMBLOCK)
	go test -v ./...

.PHONY: clean
clean:
	rm -f $(PROMBLOCK)
