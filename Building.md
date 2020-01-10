# Building FwAnalyzer

## Requirements

- golang + dep + golang-lint
- Python
- filesystem tools such as e2tools, mtools

The full list of dependencies is tracked in the [Dockerfile](Dockerfile).

## Clone Repository

```sh
go get github.com/cruise-automation/fwanalyzer
```

## Building

Before building you need to download some third party go libraries, run `make deps` before the first build.

```sh
cd go/src/github.com/cruise-automation/fwanalyzer
make deps
make
```

The `fwanalyzer` binary will be in `build/`.

# Testing

We have two types of tests: unit tests and integration tests, both tests will be triggered by running `make test`.
Tests rely on e2tools, mtools, squashfs-tools, and ubi_reader, as well as Python.

```sh
cd go/src/github.com/cruise-automation/fwanalyzer
make test
```
