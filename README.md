engineio
========

[engine.io](https://github.com/Automattic/engine.io) in Go.

## Status

Under active developement.

## Highlights
- **Compatible with the latest engine.io version**
- **Tested with the original test suites**
- **Robust salability**

## How to use

Check [this link](https://github.com/Automattic/engine.io/blob/master/README.md)
for the document of the original engine.io.

## API

TODO: Add golang style api document.

## Development

Get the repository by:

```
export GOPATH=/your/workspace
mkdir -p $GOPATH
cd $GOPATH
go get github.com/kaicheng/engineio
```

The source tree will be located at $GOPATH/src/github.com/kaicheng/engineio.

For testings, you need more dependencies:

```
cd $GOPATH/src/github.com/kaicheng/engineio/test
npm install
```

## Tests

```
cd $GOPATH/src/github.com/kaicheng/engineio/test
npm install
cd $GOPATH/src/github.com/kaicheng/engineio
make test
```

It test the server with `engine.io-client`.

## Contribution
