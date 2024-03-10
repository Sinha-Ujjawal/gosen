package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "gosen/fileContents"
    "gosen/slog"
    "gosen/stemmer"
    "gosen/stemmer/snowball"
    "gosen/tfIndex"
    "gosen/tokenizer"
    "net/http"
    "os"
    "strings"
)

const fileBufferSize uint = 100

const (
    defaultDBPath   string = "index.db"
    defaultAddr            = "127.0.0.1:6969"
    buildSubCommand        = "build"
    querySubCommand        = "query"
    serveSubCommand        = "serve"
    helpSubCommand         = "help"
)

var (
    dirPath     string
    dbPath      string
    queryString string
    topN        uint
    addr        string
)

func configBuildFlagSet() *flag.FlagSet {
    flg := flag.NewFlagSet(buildSubCommand, flag.ExitOnError)
    flg.StringVar(&dirPath, "dir", "", "Directory containing the files")
    flg.StringVar(&dbPath, "db", defaultDBPath, "Path of db to store the index. Supported formats: [.db, .json]")
    return flg
}

func configQueryFlagSet() *flag.FlagSet {
    flg := flag.NewFlagSet(querySubCommand, flag.ExitOnError)
    flg.StringVar(&dbPath, "db", defaultDBPath, "Path of db to store the index. Supported formats: [.db, .json]")
    flg.StringVar(&queryString, "query", "", "Search query")
    flg.UintVar(&topN, "topN", 10, "Top N results to show")
    return flg
}

func configServeFlagSet() *flag.FlagSet {
    flg := flag.NewFlagSet(serveSubCommand, flag.ExitOnError)
    flg.StringVar(&dbPath, "db", defaultDBPath, "Path of db to store the index. Supported formats: [.db, .json]")
    flg.StringVar(&addr, "addr", defaultAddr, "Address to serve the server on")
    return flg
}

var (
    buildFlagSet *flag.FlagSet = configBuildFlagSet()
    queryFlagSet               = configQueryFlagSet()
    serveFlagSet               = configServeFlagSet()
)

func usage(program string) {
    fmt.Printf("Usage: ./%s <SUBCOMMAND> <FLAGS>\n", program)
    fmt.Println("    SUBCOMMANDS:")
    fmt.Printf("        - %s: for building index db on documents present in a given directory\n", buildSubCommand)
    fmt.Printf("        - %s: for finding closest matching document for a given query using tf-idf\n", querySubCommand)
    fmt.Printf("        - %s: for serving index db on web\n", serveSubCommand)
    fmt.Printf("        - %s: see help\n", helpSubCommand)
    fmt.Println()
    buildFlagSet.Usage()
    fmt.Println()
    queryFlagSet.Usage()
    fmt.Println()
    serveFlagSet.Usage()
    os.Exit(1)
}

func ngrams(token string, n uint) []string {
    var ret []string
    var ngram string
    len_ := uint(len(token))
    for i := uint(0); i+n < len_; i++ {
        ngram = token[i : i+n]
        ret = append(ret, ngram)
    }
    return ret
}

func tokenize(text string) []string {
    t := tokenizer.SimpleTokenizerFromString(text)
    var tokens []string
    var stem stemmer.Stemmer = &snowball.EnglishStemmer{}
    for t.Contains() {
        token := strings.TrimSpace(strings.ToLower(t.NextToken()))
        if _, ok := STOPWORDS[token]; !ok {
            token = stem.Stem(token)
            tokens = append(tokens, token)
            tokens = append(tokens, ngrams(token, 3)...)
            tokens = append(tokens, ngrams(token, 5)...)
            tokens = append(tokens, ngrams(token, 7)...)
        }
    }
    return tokens
}

func mkIndex(program string, subcommand string) tfIndex.TFIndex {
    parts := strings.Split(dbPath, ".")
    ext := parts[len(parts)-1]
    if (subcommand == querySubCommand) || (subcommand == serveSubCommand) {
        if _, err := os.Open(dbPath); err != nil {
            slog.Fatal(err)
        }
    }
    switch ext {
    case "db":
        return tfIndex.NewSQLiteTFIndex(dbPath)
    case "json":
        index, err := tfIndex.SimpleTFINdexFromJSON(dbPath)
        if err != nil {
            if (subcommand == querySubCommand) || (subcommand == serveSubCommand) {
                slog.Fatal(err)
                return index
            }
            index = tfIndex.NewSimpleTFIndex()
        }
        return index
    default:
    }
    fmt.Printf("Unknown extension `%s` found\n", dbPath)
    usage(program)
    return nil
}

func build(program string) {
    buildFlagSet.Parse(os.Args)
    slog.Infof("Building index for directory `%s`...", dirPath)
    fileContentsCH, err := fileContents.FromDirectory(dirPath, fileBufferSize)
    if err != nil {
        slog.Fatal(err)
    }
    fileTokensCH := make(chan tfIndex.DocTokens, fileBufferSize)
    go func() {
        defer close(fileTokensCH)
        for fileContent := range fileContentsCH {
            if fileContent.Err != nil {
                slog.Errorf("build: error occurred for file `%s`: %s", fileContent.FilePath, fileContent.Err)
                continue
            }
            DocID := fileContent.FilePath
            Tokens := tokenize(fileContent.Content)
            fileTokensCH <- tfIndex.DocTokens{DocID: DocID, Tokens: Tokens}
        }
    }()
    index := mkIndex(program, buildSubCommand)
    err = index.BulkUpdateChan(fileTokensCH)
    if err != nil {
        slog.Fatal(err)
    }
    slog.Info("Successfully build the index")
    slog.Infof("Saving index to `%s`...", dbPath)
    switch index.(type) {
    case *tfIndex.SimpleTFINdex:
        err := index.(*tfIndex.SimpleTFINdex).DumpToJSON(dbPath)
        if err != nil {
            slog.Fatal(err)
        }
    case *tfIndex.SQLiteTFIndex:
        // Already saving to DB directory, when doing bulk update, hence no need to do it here
        break
    default:
        slog.Fatal("Unreachable!")
    }
    slog.Infof("Index saved to `%s`", dbPath)
}

func query(program string) {
    queryFlagSet.Parse(os.Args)
    index := mkIndex(program, querySubCommand)
    tokens := tokenize(queryString)
    results, err := index.QueryTopN(tokens, topN)
    if err != nil {
        slog.Fatal(err)
    }
    slog.Infof("Top %d results for the query: `%s`:", topN, queryString)
    for _, result := range results {
        slog.Infof("Score: %.2f, Doc: `%s`", result.Score, result.DocID)
    }
}

func setContentType(w http.ResponseWriter, contentType string) {
    w.Header().Set("Content-Type", contentType)
}

func serveStaticFile(w http.ResponseWriter, filePath string, contentType string) {
    setContentType(w, contentType)
    responseBytes, err := os.ReadFile(filePath)
    if err != nil {
        responseBytes = []byte(fmt.Sprintf("Could not read the file `%s`. Please ask the server administrator to add this file\n", filePath))
        slog.Error(string(responseBytes))
    }
    _, err = w.Write(responseBytes)
    if err != nil {
        slog.Errorf("Could not respond back with content: %s", string(responseBytes))
    }
}

func errWithMethodNotAllowed(w http.ResponseWriter) {
    http.Error(w, "Method Not Allowed!", http.StatusMethodNotAllowed)
}

func errWithInternalServerError(w http.ResponseWriter) {
    http.Error(w, "Internal Server Error!", http.StatusInternalServerError)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        serveStaticFile(w, "./index.html", "text/html; charset=utf-8")
    default:
        errWithMethodNotAllowed(w)
    }
}

func handleIndexJS(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        serveStaticFile(w, "./index.js", "text/javascript; charset=utf-8")
    default:
        errWithMethodNotAllowed(w)
    }
}

func handleLogoPNG(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        serveStaticFile(w, "./Logo.png", "text/png; charset=utf-8")
    default:
        errWithMethodNotAllowed(w)
    }
}

type searchRequest struct {
    Search string `json:"search"`
    TopN   uint   `json:"topN"`
}

type searchResponse struct {
    DocID string  `json:"docId"`
    Score float64 `json:"score"`
}

func handleSearch(w http.ResponseWriter, r *http.Request, index tfIndex.TFIndex) {
    switch r.Method {
    case http.MethodPost:
        setContentType(w, "application/json; charset=utf-8")
        decoder := json.NewDecoder(r.Body)
        var req searchRequest
        err := decoder.Decode(&req)
        if err != nil {
            http.Error(w, "Could not interpret the request. Please send the POST request with JSON body as { search: <YOUR SEARCH TEXT HERE>, topN: <TOP N results> }", http.StatusBadRequest)
            return
        }
        tokens := tokenize(req.Search)
        topN := req.TopN
        if topN == 0 {
            topN = 10
        }
        results, err := index.QueryTopN(tokens, topN)
        if err != nil {
            errWithInternalServerError(w)
            slog.Errorf("handleSearch: error occurred while searching for the query: %s", err)
            return
        }
        searchResponses := []searchResponse{}
        for _, result := range results {
            searchResponses = append(searchResponses, searchResponse{result.DocID, result.Score})
        }
        bytes, err := json.Marshal(searchResponses)
        if err != nil {
            errWithInternalServerError(w)
            slog.Errorf("handleSearch: unexpected error!: %s", err)
            return
        }
        w.Write(bytes)
    default:
        errWithMethodNotAllowed(w)
    }
}

type loggerMux struct {
    handler http.Handler
}

func (l loggerMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    url := r.URL.String()
    method := r.Method
    slog.Infof("Got request: %s %s", method, url)
    l.handler.ServeHTTP(w, r)
}

func serve(program string) {
    serveFlagSet.Parse(os.Args)
    slog.Infof("Serving index: `%s`", dbPath)
    index := mkIndex(program, serveSubCommand)
    switch index.(type) {
    case *tfIndex.SQLiteTFIndex:
        defer index.(*tfIndex.SQLiteTFIndex).Close()
    default:
    }
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.String() != "/" {
            http.NotFound(w, r)
            return
        }
        handleIndex(w, r)
    })
    mux.HandleFunc("/index", handleIndex)
    mux.HandleFunc("/index.js", handleIndexJS)
    mux.HandleFunc("/logo.png", handleLogoPNG)
    mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
        handleSearch(w, r, index)
    })
    server := loggerMux{handler: mux}
    slog.Infof("Listening on %s", addr)
    slog.Fatal(http.ListenAndServe(addr, server))
}

func main() {
    if len(os.Args) == 0 {
        slog.Fatalf("This is unexpected, os.Args `%+v` is empty!", os.Args)
    }
    program := os.Args[0]
    os.Args = os.Args[1:]

    if len(os.Args) == 0 {
        fmt.Println("Did not provide any subcommand!")
        usage(program)
    }
    subcommand := os.Args[0]
    os.Args = os.Args[1:]
    switch subcommand {
    case buildSubCommand:
        build(program)
    case querySubCommand:
        query(program)
    case serveSubCommand:
        serve(program)
    case helpSubCommand:
        usage(program)
    default:
        fmt.Printf("Unknown subcommand `%s`\n", subcommand)
        usage(program)
    }
}
