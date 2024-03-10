package tokenizer

import "unicode"

// SimpleTokenizer for parsing tokens from string
type SimpleTokenizer struct {
    content string
}

// Construct SimpleTokenizer from a string
func SimpleTokenizerFromString(content string) *SimpleTokenizer {
    return &SimpleTokenizer{content}
}

// Trim whitespaces from left
func (simpleTokenizer *SimpleTokenizer) TrimLeft() {
    for len(simpleTokenizer.content) > 0 && unicode.IsSpace(rune(simpleTokenizer.content[0])) {
        simpleTokenizer.content = simpleTokenizer.content[1:]
    }
}

// Chop n bytes from left
func (simpleTokenizer *SimpleTokenizer) ChopLeft(n int) string {
    token := simpleTokenizer.content[:n]
    simpleTokenizer.content = simpleTokenizer.content[n:]
    return token
}

// Chop while the byte meets the given predicate
func (simpleTokenizer *SimpleTokenizer) ChopWhile(pred func(byte) bool) string {
    n := 0
    for n < len(simpleTokenizer.content) && pred(simpleTokenizer.content[n]) {
        n += 1
    }
    return simpleTokenizer.ChopLeft(n)
}

// Checks if the simpleTokenizer still contain tokens
func (simpleTokenizer *SimpleTokenizer) Contains() bool {
    return len(simpleTokenizer.content) != 0
}

// Returns next token from the simpleTokenizer, also moves the simpleTokenizer to next tokens position
func (simpleTokenizer *SimpleTokenizer) NextToken() string {
    simpleTokenizer.TrimLeft()
    if len(simpleTokenizer.content) == 0 {
        return ""
    }
    if unicode.IsNumber(rune(simpleTokenizer.content[0])) {
        return simpleTokenizer.ChopWhile(func(b byte) bool { return unicode.IsNumber(rune(b)) })
    }
    if unicode.IsLetter(rune(simpleTokenizer.content[0])) {
        return simpleTokenizer.ChopWhile(func(b byte) bool { return unicode.IsLetter(rune(b)) })
    }
    return simpleTokenizer.ChopLeft(1)
}

func (simpleTokenizer *SimpleTokenizer) Tokens() []string {
    ret := []string{}
    for simpleTokenizer.Contains() {
        ret = append(ret, simpleTokenizer.NextToken())
    }
    return ret
}
