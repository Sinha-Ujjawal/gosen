package tokenizer

type Tokenizer interface {
	// Checks if the Tokenizer still contain tokens
	Contains() bool
	// Returns next token from the tokenizer, also moves to next tokens position
	NextToken() string
	// Returns a slice of tokens from the tokenizer
	Tokens() []string
}
