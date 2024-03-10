package tfIndex

import (
    "encoding/json"
    "fmt"
    "math"
    "os"
    "sort"
)

type SimpleTFINdex struct {
    index map[string]map[string]uint
}

func NewSimpleTFIndex() *SimpleTFINdex {
    return &SimpleTFINdex{index: map[string]map[string]uint{}}
}

func (simpleTFIndex *SimpleTFINdex) Update(docId string, tokens []string) error {
    freqMap, ok := simpleTFIndex.index[docId]
    if !ok {
        freqMap = map[string]uint{}
        simpleTFIndex.index[docId] = freqMap
    }
    tf := TermFrequency(tokens)
    for token, freq := range tf {
        freqMap[token] = freq
    }
    return nil
}

func (simpleTFIndex *SimpleTFINdex) BulkUpdate(docTokens map[string][]string) error {
    for docId, tokens := range docTokens {
        simpleTFIndex.Update(docId, tokens)
    }
    return nil
}

func (simpleTFINdex *SimpleTFINdex) BulkUpdateChan(docTokensCH <-chan DocTokens) error {
    for docToken := range docTokensCH {
        simpleTFINdex.Update(docToken.DocID, docToken.Tokens)
    }
    return nil
}

func (simpleTFINdex SimpleTFINdex) TF(docId string, token string) uint {
    freqMap, ok := simpleTFINdex.index[docId]
    if !ok {
        return 0
    }
    return freqMap[token]
}

func (simpleTFINdex SimpleTFINdex) DF(token string) uint {
    df := uint(0)
    for _, freqMap := range simpleTFINdex.index {
        if _, ok := freqMap[token]; ok {
            df++
        }
    }
    return df
}

func (simpleTFINdex SimpleTFINdex) IDF(token string) float64 {
    numer := len(simpleTFINdex.index)
    denom := simpleTFINdex.DF(token)
    return math.Log(float64(numer) / float64(denom))
}

func (simpleTFIndex SimpleTFINdex) Query(tokens []string) ([]QueryResult, error) {
    if len(tokens) == 0 {
        return nil, nil
    }
    idfs := map[string]float64{}
    for _, token := range tokens {
        if _, ok := idfs[token]; !ok {
            idfs[token] = simpleTFIndex.IDF(token)
        }
    }
    ret := []QueryResult{}
    for docId := range simpleTFIndex.index {
        tfIdf := 0.0
        for token, idf := range idfs {
            tf := simpleTFIndex.TF(docId, token)
            tfIdf += float64(tf) * idf
        }
        if tfIdf > 0.0 {
            ret = append(ret, QueryResult{DocID: docId, Score: tfIdf})
        }
    }
    sort.Slice(ret, func(i, j int) bool { return ret[i].Score > ret[j].Score })
    return ret, nil
}

func (simpleTFIndex SimpleTFINdex) QueryTopN(tokens []string, topN uint) ([]QueryResult, error) {
    results, err := simpleTFIndex.Query(tokens)
    return results[:min(topN, uint(len(results)))], err
}

func (simpleTFINdex SimpleTFINdex) ToJSON() ([]byte, error) {
    bytes, err := json.Marshal(simpleTFINdex.index)
    if err != nil {
        return bytes, fmt.Errorf("SimpleTFINdex.ToJSON: cannot convert to JSON: %w", err)
    }
    return bytes, nil
}

func (simpleTFIndex SimpleTFINdex) DumpToJSON(jsonPath string) error {
    bytes, err := simpleTFIndex.ToJSON()
    if err != nil {
        return fmt.Errorf("simpleTFIndex.DumpToJSON %w", err)
    }
    fd, err := os.Create(jsonPath)
    if err != nil {
        return fmt.Errorf("simpleTFIndex.DumpToJSON %w", err)
    }
    _, err = fd.Write(bytes)
    if err != nil {
        return fmt.Errorf("simpleTFIndex.DumpToJSON %w", err)
    }
    return nil
}

func SimpleTFINdexFromJSON(jsonPath string) (*SimpleTFINdex, error) {
    bytes, err := os.ReadFile(jsonPath)
    if err != nil {
        return nil, fmt.Errorf("SimpleTFINdexFromJSON: cannot read the file `%s`: %w", jsonPath, err)
    }
    ret := SimpleTFINdex{}
    err = json.Unmarshal(bytes, &ret.index)
    if err != nil {
        return nil, fmt.Errorf("SimpleTFINdexFromJSON: cannot convert from JSON: %w", err)
    }
    return &ret, nil
}
