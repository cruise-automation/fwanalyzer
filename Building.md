# Building FwAnalyzer

## Requirements

- golang (with mod support) + golang-lint
- Python
- filesystem tools such as e2tools, mtools

The full list of dependencies is tracked in the [Dockerfile](Dockerfile).

## Clone Repository

```sh
go get github.com/cruise-automation/fwanalyzer
```

## Building

Before building you need to download third party go packages, run `make deps` before the first build.

```sh
cd go/src/github.com/cruise-automation/fwanalyzer
make deps
make
```

The `fwanalyzer` binary will be in `build/`.

# Testing

We have two types of tests: unit tests and integration tests, both tests will be triggered by running `make test`.
Run `make testsetup` once to setup the test environment in `test/`.
Tests rely on e2tools, mtools, squashfs-tools, and ubi_reader, as well as Python.

```sh
cd go/src/github.com/cruise-automation/fwanalyzer
make testsetup
make test
```
