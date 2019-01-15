# Code Review Bot

[![Build Status][travis-shield]][travis-link]

[travis-shield]: https://travis-ci.org/google/code-review-bot.svg?branch=master
[travis-link]: https://travis-ci.org/google/code-review-bot

## Prerequisites

First, ensure that you have installed Go 1.11 or higher since we need the
support for [Go modules via `go
mod`](https://github.com/golang/go/wiki/Modules).

On Travis CI, we also define the env var `GO111MODULE=on` to override the [Go
1.5 `vendor` experiment](http://golang.org/s/go15vendor); you may not
necessarily need this setting in your environment if you don't have Go 1.5
`vendor` experiment also enabled.

## Building

To `crbot` tool without a cloned repo (assuming that `$GOPATH/bin` is in your
`$PATH`):

```bash
$ go get github.com/google/code-review-bot/cmd/crbot
$ crbot [options]
```

Or, from a cloned copy:

```bash
$ git clone https://github.com/google/code-review-bot.git
$ cd code-review-bot
$ go build ./cmd/crbot
$ ./crbot [options]
```

## Developing

Install [GoMock](https://github.com/golang/mock):

```bash
$ go get github.com/golang/mock/gomock@v1.2.0
$ go get github.com/golang/mock/mockgen@v1.2.0
```

This specific version of both `gomock` and `mockgen` tools is what's used in
this repo, and tests will fail if your version of these tools generates
different code, including comments.

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
