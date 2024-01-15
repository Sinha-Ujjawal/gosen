package main

import (
	"bufio"
	"database/sql"
	"encoding/xml"
	"fmt"
	"gosen/saxlike"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
)

// SAX Like XML Handler for capturing all strings from the file content
type TextHandler struct {
	saxlike.VoidHandler
	textDataSB strings.Builder
}

// Handler function to append string from the file to intermediary results
func (h *TextHandler) CharData(c xml.CharData) {
	h.textDataSB.Write(c)
	h.textDataSB.WriteString(" ")
}

// Lexer for parsing tokens from runes
type Lexer struct {
	content []rune
}

// Trim whitespaces from left
func (lexer *Lexer) TrimLeft() {
	for len(lexer.content) > 0 && unicode.IsSpace(lexer.content[0]) {
		lexer.content = lexer.content[1:]
	}
}

// Chop n runes from left
func (lexer *Lexer) ChopLeft(n int) []rune {
	token := lexer.content[:n]
	lexer.content = lexer.content[n:]
	return token
}

// Chop while the rune meets the given predicate
func (lexer *Lexer) ChopWhile(pred func(rune) bool) []rune {
	n := 0
	for n < len(lexer.content) && pred(lexer.content[n]) {
		n += 1
	}
	return lexer.ChopLeft(n)
}

// Returns next token from the lexer, also moves the lexer to next tokens position
func (lexer *Lexer) NextToken() []rune {
	lexer.TrimLeft()
	if len(lexer.content) == 0 {
		return nil
	}
	if unicode.IsNumber(lexer.content[0]) {
		return lexer.ChopWhile(unicode.IsNumber)
	}
	if unicode.IsLetter(lexer.content[0]) {
		return lexer.ChopWhile(func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsNumber(r)
		})
	}
	return lexer.ChopLeft(1)
}

// Checks if the lexer still contain tokens
func (lexer *Lexer) Contains() bool {
	return len(lexer.content) != 0
}

// FileContent struct for storing the file path of the file as well as the content
type FileContent struct {
	filePath string
	content  string
}

func FileContentFromFilePath(filePath string) (*FileContent, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("FileContentFromFilePath: failed reading the filePath %s: %w", filePath, err)
	}
	reader := bufio.NewReader(fp)
	handler := &TextHandler{}
	parser := saxlike.NewParser(reader, handler)
	err = parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("FileContentFromFilePath: failed parsing the file %s using saxlike: %w", filePath, err)
	}
	return &FileContent{filePath, handler.textDataSB.String()}, nil
}

func FileContentsFromDirectory(dirPath string) ([]FileContent, []error) {
	files, err := filepath.Glob(dirPath + "/*/*.xhtml")
	if err != nil {
		return nil, []error{fmt.Errorf("FileContentsFromDirectory: failed reading .xhtml files from the directory %s: %w", dirPath, err)}
	}
	var fileContents []FileContent
	var errs []error
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(files))
	for _, filePath := range files {
		filePath, _ := filepath.Abs(filePath)
		go func(filePath string) {
			defer wg.Done()
			fileContent, err := FileContentFromFilePath(filePath)
			lock.Lock()
			defer lock.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else {
				fileContents = append(fileContents, *fileContent)
			}
		}(filePath)
	}
	wg.Wait()
	return fileContents, errs
}

type TermFrequencies = map[string]uint

type TermFrequenciesIndex struct {
	tfs map[string]TermFrequencies
}

// Returns the term frequencies for the texts inside the file
func (fileContent FileContent) TermFrequencies() TermFrequencies {
	termFrequencies := map[string]uint{}
	lexer := &Lexer{[]rune(fileContent.content)}
	for lexer.Contains() {
		token := lexer.NextToken()
		if token != nil {
			termFrequencies[strings.ToUpper(string(token))] += 1
		} else {
			break
		}
	}
	return termFrequencies
}

func (tfIndex *TermFrequenciesIndex) DumpToSQLite3(dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		err = os.Remove(dbPath)
		if err != nil {
			return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot remove the file %s: %w", dbPath, err)
		}
	}
	_, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot create the file %s: %w", dbPath, err)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot connect to db %s: %w", dbPath, err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS termFrequenciesIndex (filePath STRING NOT NULL, token STRING NOT NULL, frequency INTEGER NOT NULL);")
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot create the table: %w", err)
	}
	var valueStrings []string
	var valueArgs []any
	const BatchSize = 333
	for filePath, tf := range tfIndex.tfs {
		for term, freq := range tf {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, filePath)
			valueArgs = append(valueArgs, term)
			valueArgs = append(valueArgs, freq)
			if len(valueStrings) == BatchSize {
				stmt := fmt.Sprintf("INSERT INTO termFrequenciesIndex (filePath, token, frequency) VALUES %s", strings.Join(valueStrings, ","))
				_, err = db.Exec(stmt, valueArgs...)
				if err != nil {
					return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot execute the statement: %w", err)
				}
				valueStrings = nil
				valueArgs = nil
			}
		}
	}
	if len(valueStrings) > 0 {
		stmt := fmt.Sprintf("INSERT INTO termFrequenciesIndex (filePath, token, frequency) VALUES %s", strings.Join(valueStrings, ","))
		_, err = db.Exec(stmt, valueArgs...)
		if err != nil {
			return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot execute the statement: %w", err)
		}
	}
	return nil
}

func LoadTermFrequenciesIndexFromSQLite3(dbPath string) (*TermFrequenciesIndex, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("LoadTermFrequenciesIndexFromSQLite3 cannot connect to db %s: %w", dbPath, err)
	}
	rows, err := db.Query("SELECT * FROM termFrequenciesIndex;")
	if err != nil {
		return nil, fmt.Errorf("LoadTermFrequenciesIndexFromSQLite3 cannot load data from the db %s, table: termFrequenciesIndex :%w", dbPath, err)
	}
	defer rows.Close()
	tfs := map[string]TermFrequencies{}
	for rows.Next() {
		var filePath string = ""
		var token string = ""
		var freq uint = 0
		err = rows.Scan(&filePath, &token, &freq)
		if err != nil {
			return nil, fmt.Errorf("LoadTermFrequenciesIndexFromSQLite3 cannot parse the rows into filePath string, token string, freq uint: %w", err)
		}
		tf, ok := tfs[filePath]
		if !ok {
			tfs[filePath] = TermFrequencies{}
			tf = tfs[filePath]
		}
		tf[token] = freq
	}
	return &TermFrequenciesIndex{tfs}, nil
}

func MkTermFrequenciesIndexFromFileContents(fileContents []FileContent) TermFrequenciesIndex {
	tfs := map[string]TermFrequencies{}
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(fileContents))
	for _, fileContent := range fileContents {
		go func(fileContent FileContent) {
			defer wg.Done()
			tf := fileContent.TermFrequencies()
			lock.Lock()
			defer lock.Unlock()
			tfs[fileContent.filePath] = tf
		}(fileContent)
	}
	wg.Wait()
	return TermFrequenciesIndex{tfs}
}

func main() {
	// dirPath := "./docs.gl"
	dbName := "docs.gl.db"
	// fileContents, errs := FileContentsFromDirectory(dirPath)
	// for _, err := range errs {
	// 	if err != nil {
	// 		log.Fatalf("%s", err)
	// 	}
	// }
	// termFrequenciesIndex := MkTermFrequenciesIndexFromFileContents(fileContents)
	// // for filePath, tf := range termFrequenciesIndex.tfs {
	// // 	fmt.Printf("File %s has %d terms\n", filePath, len(tf))
	// // }
	// err := termFrequenciesIndex.DumpToSQLite3(dbName)
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
	termFrequenciesIndex, err := LoadTermFrequenciesIndexFromSQLite3(dbName)
	if err != nil {
		log.Fatalf("%s", err)
	}
	for filePath, tf := range termFrequenciesIndex.tfs {
		fmt.Printf("File %s has %d terms\n", filePath, len(tf))
	}
	_ = termFrequenciesIndex
}
