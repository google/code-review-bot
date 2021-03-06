# Code Review Bot

[![Build Status][github-ci-badge]][github-ci-url]
[![Go Report Card][go-report-card-badge]][go-report-card-url]
[![API docs][godoc-badge]][godoc-url]

[github-ci-badge]: https://github.com/google/code-review-bot/actions/workflows/main.yml/badge.svg
[github-ci-url]: https://github.com/google/code-review-bot/actions/workflows/main.yml
[go-report-card-badge]: https://goreportcard.com/badge/github.com/google/code-review-bot
[go-report-card-url]: https://goreportcard.com/report/github.com/google/code-review-bot
[godoc-badge]: https://img.shields.io/badge/godoc-reference-5272B4.svg
[godoc-url]: https://godoc.org/github.com/google/code-review-bot

## Prerequisites

Ensure that you have installed Go 1.11 or higher to enable support for [Go
modules](https://github.com/golang/go/wiki/Modules) via `go mod`.

If you're using Go 1.11 or 1.12, set the environment variable `GO111MODULE=on`
(Go 1.13 and later versions [automatically enable module
support](https://blog.golang.org/modules2019)).

## Building

To build the `crbot` tool without a cloned repo (assuming that `$GOPATH/bin` is
in your `$PATH`):

```bash
$ go get github.com/google/code-review-bot/cmd/crbot
$ crbot [options]
```

Or, from a cloned repo:

```bash
$ git clone https://github.com/google/code-review-bot.git
$ cd code-review-bot
$ go build ./cmd/crbot
$ ./crbot [options]
```

## Developing

Install [GoMock](https://github.com/golang/mock):

```bash
$ go get github.com/golang/mock/gomock@v1.4.0
$ go get github.com/golang/mock/mockgen@v1.4.0
```

Generate the mocks:

```bash
$ go generate ./...
```

This specific version of both `gomock` and `mockgen` tools is what's used in
this repo, and tests will fail if your version of these tools generates
different code, including comments.

To update the versions of these tools used in this repo:

1. update the version numbers in this file (above) as well as in
   [`.github/workflows/main.yml`](.github/workflows/main.yml) and
   [`go.mod`](go.mod) to match
1. run `go mod tidy` to update the `go.sum` file
1. run the updated `go get` commands above to get newer versions of the tools
1. run the `go generate` command above to regenerate the mocks
1. [run the tests](#testing) from the top-level of the tree
1. commit your changes to this file (`README.md`), `go.mod`, `go.sum`, and
   `.travis.yml`, making sure that the build passes on Travis CI before merging
   the change

## Testing

Just what you might expect:

```bash
$ make test
```

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for more details.

## License

Apache 2.0; see [`LICENSE`](LICENSE) for more details.

## Disclaimer

This project is not an official Google project. It is not supported by Google
and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
