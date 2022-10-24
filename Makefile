.PHONY: build

ifeq ($(GOOS),)
GOOS := "linux"
endif

VERSION=1.4.1

PWD := $(shell pwd)

all: build

.PHONY: build
build:
	go mod verify
	mkdir -p build
	GOOS=$(GOOS) go build -a -ldflags '-w -s' -o build/fwanalyzer ./cmd/fwanalyzer

.PHONY: release
release: build
	mkdir -p release
	cp build/fwanalyzer release/fwanalyzer-$(VERSION)-linux-amd64

.PHONY: testsetup
testsetup:
	gunzip -c test/test.img.gz >test/test.img
	gunzip -c test/ubifs.img.gz >test/ubifs.img
	gunzip -c test/cap_ext2.img.gz >test/cap_ext2.img
	sudo setcap cap_net_admin+p test/test.cap.file
	getcap test/test.cap.file

.PHONY: test
test:
	PATH="$(PWD)/scripts:$(PWD)/test:$(PATH)" go test -count=3 -cover ./...

.PHONY: integration-test
integration-test: build
	PATH="$(PWD)/scripts:$(PWD)/test:$(PWD)/build:$(PATH)" ./test/test.py

.PHONY: ci-tests
ci-tests: build test integration-test
	echo "done"

.PHONY: modules
modules:
	go mod tidy

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
	go mod download
