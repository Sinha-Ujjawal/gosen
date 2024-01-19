package main

import (
	"flag"
	"fmt"
	"gosen/fileContents"
	"gosen/tfIndex"
	"gosen/tokenizer"
	"log"
	"os"
	"strings"
)

const (
	defaultDBPath   string = "index.db"
	buildSubCommand        = "build"
	querySubCommand        = "query"
)

var (
	dirPath string
	dbPath  string
	query   string
	topN    uint
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
	flg.StringVar(&query, "query", "", "Search query")
	flg.UintVar(&topN, "topN", 10, "Top N results to show")
	return flg
}

var (
	buildFlagSet *flag.FlagSet = configBuildFlagSet()
	queryFlagSet               = configQueryFlagSet()
)

func usage(program string) {
	fmt.Printf("Usage: ./%s <SUBCOMMAND> <FLAGS>\n", program)
	fmt.Println("    SUBCOMMANDS:")
	fmt.Printf("        1. %s\n", buildSubCommand)
	fmt.Printf("        2. %s\n", querySubCommand)
	fmt.Println()
	buildFlagSet.Usage()
	fmt.Println()
	queryFlagSet.Usage()
	os.Exit(1)
}

func tokenize(text string) []string {
	text = strings.ToUpper(strings.TrimSpace(text))
	return tokenizer.SimpleTokenizerFromString(text).Tokens()
}

func mkIndex(program string, subcommand string) tfIndex.TFIndex {
	parts := strings.Split(dbPath, ".")
	ext := parts[len(parts)-1]
	if subcommand == querySubCommand {
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
			if subcommand == querySubCommand {
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

func main() {
	defaultLogger := log.Default()
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
	var index tfIndex.TFIndex
	switch subcommand {
	case buildSubCommand:
		buildFlagSet.Parse(os.Args)
		defaultLogger.Printf("Reading directory `%s` contents...\n", dirPath)
		fileContents, errs := fileContents.FromDirectory(dirPath)
		for _, err := range errs {
			if err != nil {
				log.Fatalf("%s", err)
			}
		}
		defaultLogger.Println("Successfully read directory contents")
		defaultLogger.Println("Building index...")
		index := mkIndex(program, subcommand)
		fileTokens := map[string][]string{}
		for filePath, fileContent := range fileContents {
			fileTokens[filePath] = tokenize(fileContent)
		}
		err := index.BulkUpdate(fileTokens)
		if err != nil {
			log.Fatalf("%s", err)
		}
		defaultLogger.Println("Successfully build the index")
		defaultLogger.Printf("Saving index to `%s`...\n", dbPath)
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
		defaultLogger.Printf("Index saved to `%s`\n", dbPath)
	case querySubCommand:
		queryFlagSet.Parse(os.Args)
		index = mkIndex(program, subcommand)
		tokens := tokenize(query)
		results, err := index.QueryTopN(tokens, topN)
		if err != nil {
			log.Fatalf("%s", err)
		}
		defaultLogger.Printf("Top %d results for the query: `%s`:\n", topN, query)
		for _, result := range results {
			defaultLogger.Printf("Score: %.2f, Doc: `%s`\n", result.Score, result.DocID)
		}
	default:
		fmt.Printf("Unknown subcommand `%s`\n", subcommand)
		usage(program)
	}
}
