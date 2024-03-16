package tfIndex

type QueryResult struct {
	DocID string
	Score float64
}

type DocTokens struct {
	DocID  string
	Tokens []string
}

type TFIndex interface {
	Update(docId string, tokens []string) error
	BulkUpdate(docTokens map[string][]string) error
	BulkUpdateChan(docTokensCH <-chan DocTokens) error
	Query(tokens []string) ([]QueryResult, error)
	QueryTopN(tokens []string, topN uint) ([]QueryResult, error)
}

func TermFrequency(tokens []string) map[string]uint {
	ret := map[string]uint{}
	for _, token := range tokens {
		ret[token] += 1
	}
	return ret
}
