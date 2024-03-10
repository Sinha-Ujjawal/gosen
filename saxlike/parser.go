package saxlike

import (
    "encoding/xml"
    "errors"
    "io"
)

// SAX-like XML Parser
type Parser struct {
    *xml.Decoder
    handler Handler
}

// Create a New Parser
func NewParser(reader io.Reader, handler Handler) *Parser {
    decoder := xml.NewDecoder(reader)
    return &Parser{decoder, handler}
}

// SetHTMLMode make Parser can parse invalid HTML
func (p *Parser) SetHTMLMode() {
    p.Strict = false
    p.AutoClose = xml.HTMLAutoClose
    p.Entity = xml.HTMLEntity
}

// Parse calls handler's methods
// when the parser encount a start-element,a end-element, a comment and so on.
func (p *Parser) Parse() error {
    var err error
    p.handler.StartDocument()

    for err == nil {
        token, err := p.Token()
        if err != nil {
            break
        }
        switch token.(type) {
        case xml.StartElement:
            s := token.(xml.StartElement)
            p.handler.StartElement(s)
        case xml.EndElement:
            e := token.(xml.EndElement)
            p.handler.EndElement(e)
        case xml.CharData:
            c := token.(xml.CharData)
            p.handler.CharData(c)
        case xml.Comment:
            com := token.(xml.Comment)
            p.handler.Comment(com)
        case xml.ProcInst:
            pro := token.(xml.ProcInst)
            p.handler.ProcInst(pro)
        case xml.Directive:
            dir := token.(xml.Directive)
            p.handler.Directive(dir)
        default:
            err = errors.New("unknown xml token!")
        }
    }
    if err != nil {
        return err
    }
    p.handler.EndDocument()
    return nil
}

// Create a parser and parse
func Parse(reader io.Reader, handler Handler, htmlMode bool) error {
    decoder := xml.NewDecoder(reader)
    parser := &Parser{decoder, handler}
    if htmlMode {
        parser.SetHTMLMode()
    }
    return parser.Parse()
}
