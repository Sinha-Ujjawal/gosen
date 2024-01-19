package tfIndex

type QueryResult struct {
	DocID string
	Score float64
}

type TFIndex interface {
	Update(docId string, tokens []string) error
	BulkUpdate(docTokens map[string][]string) error
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
