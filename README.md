# Code Review Bot

## Usage

First, ensure that you have installed Go and set up `$GOPATH`. Then, build the
`crbot` tool:

* from scratch:

   ```bash
   $ go get github.com/google/code-review-bot/cmd/crbot
   ```

* or, from a cloned copy:

   ```bash
   $ go get ./...
   ```

And then to use it:

* if `$GOPATH/bin` is in your `$PATH`:

   ```bash
   $ crbot [options]
   ```

* or, from a cloned repo:

   ```bash
   $ cmd/crbot/crbot [options]
   ```

## Development

Install [GoMock](https://github.com/golang/mock):

```bash
$ go get github.com/golang/mock/gomock
$ go get github.com/golang/mock/mockgen
```

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
