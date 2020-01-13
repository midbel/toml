package toml

import (
	"fmt"
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
	p.nextToken()
	p.nextToken()

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

func (p *Parser) parseTable(t *Table) error {
	return nil
}

func (p *Parser) parseOptions(t *Table) error {
	for !p.isDone() {
		if p.curr.Type == lsquare {
			break
		}
		opt, err := p.parseOption()
		if err != nil {
			return err
		}
		_ = opt
	}
	if p.curr.Type != lsquare {
		return p.unexpectedToken("lsquare")
	}
	return nil
}

func (p *Parser) parseOption() (Node, error) {
	if !p.curr.IsIdent() {
		return nil, p.unexpectedToken("ident")
	}
	var (
		opt = Option{key: p.curr}
		err error
	)
	p.nextToken()
	if p.curr.Type != equal {
		return nil, p.unexpectedToken("equal")
	}
	p.nextToken()
	switch p.curr.Type {
	case lsquare:
		opt.value, err = p.parseArray()
	case lcurly:
		opt.value, err = p.parseInline()
	default:
		opt.value, err = p.parseLiteral()
	}
	if err != nil {
		return nil, err
	}
	p.nextToken()
	if p.curr.Type != Newline {
		return nil, p.unexpectedToken("newline")
	}
	return &opt, nil
}

func (p *Parser) parseLiteral() (Node, error) {
	switch p.curr.Type {
	default:
		return nil, p.unexpectedToken("literal type")
	case String, Integer, Float, Bool, Date, Time, DateTime:
		lit := Literal{
			token: p.curr,
		}
		return &lit, nil
	}
}

func (p *Parser) parseArray() (Node, error) {
	p.nextToken()

	var a Array
	for !p.isDone() {
		if p.curr.Type == rsquare {
			break
		}
		var (
			node Node
			err  error
		)
		switch p.curr.Type {
		case lsquare:
			node, err = p.parseArray()
		case lcurly:
			node, err = p.parseInline()
		default:
			node, err = p.parseLiteral()
		}
		if err != nil {
			return nil, err
		}
	}
	if p.curr.Type != rsquare {
		return nil, p.unexpectedToken("rsquare")
	}
	p.nextToken()
	return &a, nil
}

func (p *Parser) parseInline() (Node, error) {
	p.nextToken()

	var t Table
	for !p.isDone() {
		if p.curr.Type == rcurly {
			break
		}
	}
	if p.curr.Type != rcurly {
		return nil, p.unexpectedToken("rcurly")
	}
	p.nextToken()
	return &t, nil
}

func (p *Parser) nextToken() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == EOF || p.curr.Type == Illegal
}

func (p *Parser) unexpectedToken(want string) error {
	return fmt.Errorf("unexpected token %s (want: %s)", p.curr, want)
}
