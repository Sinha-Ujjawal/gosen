### Local Search Engine in Golang

Inspired by [Local Search Engine in Rust](https://github.com/tsoding/seroost) from [@tsoding](https://github.com/tsoding) made a similar project in [Golang](https://go.dev/).

### TODOS

1. [✅] Write code for indexing a folder containing `.xhtmls` as `TermFrequenciesIndex`

2. [✅] Write code for storing the `TermFrequenciesIndex` in a file. In my case I have used [SQLite3](https://www.sqlite.org/index.html)

3. [✅] Write code for loading `TermFrequenciesIndex` from the [SQLite3](https://www.sqlite.org/index.html) DB.

4. [🟨] Write code for answering query using the index. Have to use [TF-IDF](https://en.wikipedia.org/wiki/Tf%E2%80%93idf) algorithm for this somehow.

5. [🟨] Write code for adding subcommands for-
    1. Indexing a folder and storing it in [SQLite3](https://www.sqlite.org/index.html) DB.
    ```console
    --index --folder --db
    ```
    2. Loading the index from the DB, and answer the user provided query
    ```console
    --query --db --search_text --top_n[Default: 10]
    ```

### References

1. Stolen [saxlike](./saxlike/) from [@kokardy/saxlike](https://github.com/kokardy/saxlike/tree/master)

2. [@tsoding's](https://github.com/tsoding) Playlist ["Search Engine in Rust"](https://youtube.com/playlist?list=PLpM-Dvs8t0VZXC-91PpIp-eAt0WF5SKEv&si=M0LhV-bsL8jHrE5t)

### Copyrights

Licensed under [@MIT](./LICENSE)
