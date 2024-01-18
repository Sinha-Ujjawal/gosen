package main

import "strings"

type Freq struct {
	frequency           uint
	inverseDocFrequency uint
}

type TermFrequencies = map[string]*Freq

// Returns the term frequencies for the texts inside the file
func (fileContent FileContent) TermFrequencies() TermFrequencies {
	termFrequencies := TermFrequencies{}
	lexer := &Lexer{[]rune(fileContent.content)}
	for lexer.Contains() {
		token := lexer.NextToken()
		if token != nil {
			key := strings.ToUpper(string(token))
			freq, ok := termFrequencies[key]
			if !ok {
				freq = &Freq{
					frequency:           0,
					inverseDocFrequency: 1,
				}
				termFrequencies[key] = freq
			}
			freq.frequency += 1
		} else {
			break
		}
	}
	return termFrequencies
}
