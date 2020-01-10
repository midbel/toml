package toml

import (
	"io"
)

type Parser struct {
	scan *Scanner
	peek Token
	curr Token
}

func Parse(r io.Reader) (Node, error) {
	s, err := Scan(r)
	if err != nil {
		return nil, err
	}
	s.KeepMultiline = false
	s.KeepComment = false

	p := Parser{scan: s}
	p.next()
	p.next()

	return p.Parse()
}

func (p *Parser) Parse() (Node, error) {
	var t Table
	if err := p.parseOptions(&t); err != nil {
		return nil, err
	}
	for !p.isDone() {
		if err := p.parseTable(&t); err != nil {
			return nil, err
		}
	}
	return &t, nil
}

func (p *Parser) parseOptions(t *Table) error {
	return nil
}

func (p *Parser) parseTable(t *Table) error {
	return nil
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == EOF || p.curr.Type == Illegal
}
