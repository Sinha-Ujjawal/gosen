### Local Search Engine in Golang

Inspired by [Local Search Engine in Rust](https://github.com/tsoding/seroost) from [@tsoding](https://github.com/tsoding) made a similar project in [Golang](https://go.dev/).

### Getting Started

```console
go build -tags "sqlite_math_functions" -o <PROGRAM> .
```

### Usage
```console
Usage: <PROGRAM> <SUBCOMMAND> <FLAGS>
    SUBCOMMANDS:
        1. build
        2. query

Usage of build:
  -db string
        Path of db to store the index. Supported formats: [.db, .json] (default "index.db")
  -dir string
        Directory containing the files.

Usage of query:
  -db string
        Path of db to store the index. Supported formats: [.db, .json] (default "index.db")
  -query string
        Search query
  -topN uint
        Top N results to show (default 10)
```

### References

1. Stolen [saxlike](./saxlike/) from [@kokardy/saxlike](https://github.com/kokardy/saxlike/tree/master)

2. [@tsoding's](https://github.com/tsoding) Playlist ["Search Engine in Rust"](https://youtube.com/playlist?list=PLpM-Dvs8t0VZXC-91PpIp-eAt0WF5SKEv&si=M0LhV-bsL8jHrE5t)

### Copyrights

Licensed under [@MIT](./LICENSE)
