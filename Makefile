.PHONY: build

ifeq ($(GOOS),)
GOOS := "linux"
endif

all: build

.PHONY: build
build:
	dep check
	mkdir -p build
	GOOS=$(GOOS) go build -a -ldflags '-w -s' -o build/fwanalyzer ./cmd/fwanalyzer

.PHONY: test
test: lint build
	gunzip -c test/test.img.gz >test/test.img
	gunzip -c test/ubifs.img.gz >test/ubifs.img
	PATH=$(PATH):`pwd`/scripts go test -count=3 -cover ./...
	PATH=./test:$(PATH):./scripts:./build ./test/test.py


.PHONY: lint
lint:
	golangci-lint run

.PHONY: deploy
deploy: build

.PHONY: clean
clean:
	rm -rf build

.PHONY: distclean
distclean: clean
	rm -rf vendor

.PHONY: deps
deps:
	dep ensure
