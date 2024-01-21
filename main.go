package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gosen/fileContents"
	"gosen/tfIndex"
	"gosen/tokenizer"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	defaultDBPath   string = "index.db"
	defaultAddr            = "127.0.0.1:6969"
	buildSubCommand        = "build"
	querySubCommand        = "query"
	serveSubCommand        = "serve"
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
	fmt.Printf("        1. %s\n", buildSubCommand)
	fmt.Printf("        2. %s\n", querySubCommand)
	fmt.Printf("        3. %s\n", serveSubCommand)
	fmt.Println()
	buildFlagSet.Usage()
	fmt.Println()
	queryFlagSet.Usage()
	fmt.Println()
	serveFlagSet.Usage()
	os.Exit(1)
}

func tokenize(text string) []string {
	text = strings.ToUpper(strings.TrimSpace(text))
	return tokenizer.SimpleTokenizerFromString(text).Tokens()
}

func mkIndex(program string, subcommand string) tfIndex.TFIndex {
	parts := strings.Split(dbPath, ".")
	ext := parts[len(parts)-1]
	if (subcommand == querySubCommand) || (subcommand == serveSubCommand) {
		if _, err := os.Open(dbPath); err != nil {
			log.Fatalf("%s", err)
		}
	}
	switch ext {
	case "db":
		return tfIndex.NewSQLiteTFIndex(dbPath)
	case "json":
		index, err := tfIndex.SimpleTFINdexFromJSON(dbPath)
		if err != nil {
			if (subcommand == querySubCommand) || (subcommand == serveSubCommand) {
				log.Fatalf("%s", err)
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
	log.Printf("Reading directory `%s` contents...\n", dirPath)
	fileContents, errs := fileContents.FromDirectory(dirPath)
	for _, err := range errs {
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
	log.Println("Successfully read directory contents")
	log.Println("Building index...")
	index := mkIndex(program, buildSubCommand)
	fileTokens := map[string][]string{}
	for filePath, fileContent := range fileContents {
		fileTokens[filePath] = tokenize(fileContent)
	}
	err := index.BulkUpdate(fileTokens)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Println("Successfully build the index")
	log.Printf("Saving index to `%s`...\n", dbPath)
	switch index.(type) {
	case *tfIndex.SimpleTFINdex:
		err := index.(*tfIndex.SimpleTFINdex).DumpToJSON(dbPath)
		if err != nil {
			log.Fatalf("%s", err)
		}
	case *tfIndex.SQLiteTFIndex:
		// Already saving to DB directory, when doing bulk update, hence no need to do it here
		break
	default:
		log.Fatalf("Unreachable!")
	}
	log.Printf("Index saved to `%s`\n", dbPath)
}

func query(program string) {
	queryFlagSet.Parse(os.Args)
	index := mkIndex(program, querySubCommand)
	tokens := tokenize(queryString)
	results, err := index.QueryTopN(tokens, topN)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Printf("Top %d results for the query: `%s`:\n", topN, queryString)
	for _, result := range results {
		log.Printf("Score: %.2f, Doc: `%s`\n", result.Score, result.DocID)
	}
}

func serveStaticFile(w http.ResponseWriter, filePath string, contentType string) {
	w.Header().Add("Content-Type", contentType)
	responseBytes, err := os.ReadFile(filePath)
	if err != nil {
		responseBytes = []byte(fmt.Sprintf("Could not read the file `%s`. Please ask the server administrator to add this file\n", filePath))
		log.Print(string(responseBytes))
	}
	_, err = w.Write(responseBytes)
	if err != nil {
		log.Println("Could not respond back with content: ", string(responseBytes))
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
			log.Printf("handleSearch: error occurred while searching for the query: %s\n", err)
			return
		}
		searchResponses := []searchResponse{}
		for _, result := range results {
			searchResponses = append(searchResponses, searchResponse{result.DocID, result.Score})
		}
		bytes, err := json.Marshal(searchResponses)
		if err != nil {
			errWithInternalServerError(w)
			log.Printf("handleSearch: unexpected error!: %s\n", err)
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
	log.Printf("Got request: %s %s\n", method, url)
	l.handler.ServeHTTP(w, r)
}

func serve(program string) {
	serveFlagSet.Parse(os.Args)
	index := mkIndex(program, querySubCommand)
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
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, index)
	})
	server := loggerMux{handler: mux}
	log.Printf("Listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, server))
}

func main() {
	if len(os.Args) == 0 {
		log.Fatalf("This is unexpected, os.Args `%+v` is empty!", os.Args)
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
	default:
		fmt.Printf("Unknown subcommand `%s`\n", subcommand)
		usage(program)
	}
}
