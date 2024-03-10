package stemmer

type Stemmer interface {
    Stem(token string) string
}
