package main

import "strings"

type TermFrequencies = map[string]uint

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
