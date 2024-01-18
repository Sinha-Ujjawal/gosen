package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type TermFrequenciesIndex struct {
	tfs map[string]TermFrequencies
}

func (tfIndex *TermFrequenciesIndex) DumpToSQLite3(dbPath string) error {
	if _, err := os.Stat(dbPath); err != nil {
		_, err := os.Create(dbPath)
		if err != nil {
			return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot create the file %s: %w", dbPath, err)
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot connect to db %s: %w", dbPath, err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS termFrequenciesIndex (
			filePath            STRING  NOT NULL,
			token               TEXT    NOT NULL,
			frequency           INTEGER NOT NULL,
			docFrequency        INTEGER,
			totalDocuments      INTEGER,
			inverseDocFrequency DOUBLE
		);
		CREATE UNIQUE INDEX IF NOT EXISTS ux_filePath_token ON termFrequenciesIndex(filePath, token);
	`)
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot create the table: %w", err)
	}
	var valueStrings []string
	var valueArgs []any
	const BatchSize = 333
	flushToDB := func() error {
		stmt := fmt.Sprintf(`
			INSERT INTO termFrequenciesIndex (filePath, token, frequency) VALUES %s
			ON CONFLICT(filePath, token) DO UPDATE SET
				frequency = excluded.frequency
			`,
			strings.Join(valueStrings, ","),
		)
		_, err = db.Exec(stmt, valueArgs...)
		if err != nil {
			valueArgsAsJSON, _ := json.Marshal(valueArgs)
			return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot execute the statement `%s`, valueArgs: %s: %w", stmt, valueArgsAsJSON, err)
		}
		return nil
	}
	for filePath, tf := range tfIndex.tfs {
		for term, freq := range tf {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, filePath)
			valueArgs = append(valueArgs, term)
			valueArgs = append(valueArgs, freq.frequency)
			if len(valueStrings) == BatchSize {
				if err := flushToDB(); err != nil {
					return err
				}
				valueStrings = nil
				valueArgs = nil
			}
		}
	}
	if len(valueStrings) > 0 {
		if err := flushToDB(); err != nil {
			return err
		}
	}
	updateStats := `
		WITH docFrequencyByToken AS (
			SELECT
				token,
				COUNT(DISTINCT filePath) docFrequency
			FROM termFrequenciesIndex
			GROUP BY
				token
			ORDER BY docFrequency ASC
		)
		UPDATE termFrequenciesIndex
		SET
			docFrequency = (
				SELECT docFrequency
				FROM docFrequencyByToken
				WHERE token = termFrequenciesIndex.token
			),
			totalDocuments = (
				SELECT COUNT(DISTINCT filePath)
				FROM termFrequenciesIndex
			)
		;
		UPDATE termFrequenciesIndex
		SET
			inverseDocFrequency = LN(totalDocuments / docFrequency)
		;
	`
	_, err = db.Exec(updateStats)
	if err != nil {
		return fmt.Errorf("TermFrequenciesIndex.DumpToSQLite3 cannot refresh the stats using the query `%s`: %w", updateStats, err)
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
		var frequency uint = 0
		var docFrequency uint = 0
		err = rows.Scan(&filePath, &token, &frequency, &docFrequency)
		if err != nil {
			return nil, fmt.Errorf("LoadTermFrequenciesIndexFromSQLite3 cannot parse the rows into filePath string, token string, freq uint: %w", err)
		}
		tf, ok := tfs[filePath]
		if !ok {
			tfs[filePath] = TermFrequencies{}
			tf = tfs[filePath]
		}
		tf[token] = &Freq{
			frequency,
			docFrequency,
		}
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
