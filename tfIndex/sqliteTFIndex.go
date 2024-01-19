package tfIndex

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const BatchSize = 1000

type SQLiteTFIndex struct {
	dbPath string
}

func NewSQLiteTFIndex(dbPath string) *SQLiteTFIndex {
	return &SQLiteTFIndex{dbPath}
}

func (sqliteTFIndex *SQLiteTFIndex) Update(docId string, tokens []string) error {
	return sqliteTFIndex.BulkUpdate(map[string][]string{docId: tokens})
}

func (sqliteTFIndex *SQLiteTFIndex) Connect() (*sql.DB, error) {
	dbPath := sqliteTFIndex.dbPath
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("SQLiteTFIndex.Connect cannot connect to db %s: %w", dbPath, err)
	}
	return db, nil
}

func (sqliteTFIndex *SQLiteTFIndex) BulkUpdate(docTokens map[string][]string) error {
	db, err := sqliteTFIndex.Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS termFrequenciesIndex (
			filePath            STRING  NOT NULL,
			token               TEXT    NOT NULL,
			frequency           INTEGER,
			docFrequency        INTEGER,
			totalDocuments      INTEGER,
			inverseDocFrequency REAL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS ux_filePath_token ON termFrequenciesIndex(filePath, token);
	`)
	if err != nil {
		return fmt.Errorf("SQLiteTFIndex.DumpToSQLite3 cannot create the table: %w", err)
	}
	var valueStrings []string
	var valueArgs []any
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
			return fmt.Errorf("SQLiteTFIndex.DumpToSQLite3 cannot execute the statement `%s`, valueArgs: %s: %w", stmt, valueArgsAsJSON, err)
		}
		return nil
	}
	for filePath, tokens := range docTokens {
		tf := TermFrequency(tokens)
		for term, freq := range tf {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, filePath)
			valueArgs = append(valueArgs, term)
			valueArgs = append(valueArgs, freq)
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
		return fmt.Errorf("SQLiteTFIndex.DumpToSQLite3 cannot refresh the stats using the query `%s`: %w", updateStats, err)
	}
	return nil
}

func (sqliteTFIndex *SQLiteTFIndex) queryHelper(tokens []string, topN *uint) ([]QueryResult, error) {
	db, err := sqliteTFIndex.Connect()
	if err != nil {
		return nil, err
	}
	args := []any{}
	seenBefore := map[string]bool{}
	for _, token := range tokens {
		if _, ok := seenBefore[token]; ok {
			continue
		}
		args = append(args, token)
		seenBefore[token] = true
	}
	query := `
		SELECT
			filePath,
			SUM(frequency * inverseDocFrequency) tfidf
		FROM termFrequenciesIndex
		WHERE token IN (?` + strings.Repeat(", ?", len(args)-1) + `)
		GROUP BY
			filePath
		ORDER BY
			tfidf
		DESC
	`
	if topN != nil {
		query += " LIMIT " + fmt.Sprintf("%d", *topN)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		argsAsJSON, _ := json.Marshal(args)
		return nil, fmt.Errorf("SQLiteTFIndex.Query cannot run the query `%s`, with args: %s: %w", query, argsAsJSON, err)
	}
	ret := []QueryResult{}
	for rows.Next() {
		docId := ""
		score := 0.0
		err := rows.Scan(&docId, &score)
		if err != nil {
			return nil, fmt.Errorf("SQLiteTFIndex.Query could not parse the rows into QueryResult: %w", err)
		}
		if score > 0.0 {
			ret = append(ret, QueryResult{DocID: docId, Score: score})
		}
	}
	return ret, nil
}

func (sqliteTFIndex *SQLiteTFIndex) Query(tokens []string) ([]QueryResult, error) {
	return sqliteTFIndex.queryHelper(tokens, nil)
}

func (sqliteTFIndex *SQLiteTFIndex) QueryTopN(tokens []string, topN uint) ([]QueryResult, error) {
	return sqliteTFIndex.queryHelper(tokens, &topN)
}
