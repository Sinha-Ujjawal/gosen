package main

import "unicode"

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
