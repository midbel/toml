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
		opt, err := p.parseOption(true)
		if err != nil {
			return err
		}
		p.nextToken()
		if p.curr.Type != Newline {
			return p.unexpectedToken("newline")
		}
		if err := t.Append(opt); err != nil {
			return err
		}
		p.nextToken()
	}
	if p.curr.Type != lsquare && !p.isDone() {
		return p.unexpectedToken("lsquare")
	}
	return nil
}

func (p *Parser) parseOption(dotted bool) (Node, error) {
	if !p.curr.IsIdent() {
		return nil, p.unexpectedToken("ident")
	}
	if p.peek.Type == dot {
		if dotted {

		} else {
			return nil, p.unexpectedToken("equal")
		}
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
	return &opt, nil
}

func (p *Parser) parseLiteral() (Node, error) {
	switch p.curr.Type {
	default:
		return nil, p.unexpectedToken("literal")
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
		p.nextToken()
		if p.curr.Type == rsquare {
			break
		}
		if p.curr.Type != comma {
			return nil, p.unexpectedToken("comma")
		}
		p.nextToken()
		_ = node
	}
	if p.curr.Type != rsquare {
		return nil, p.unexpectedToken("rsquare")
	}
	return &a, nil
}

func (p *Parser) parseInline() (Node, error) {
	p.nextToken()

	var t Table
	for !p.isDone() {
		if p.curr.Type == rcurly {
			break
		}
		_, err := p.parseOption(false)
		if err != nil {
			return nil, err
		}
		p.nextToken()
		if p.curr.Type == rcurly {
			break
		}
		if p.curr.Type != comma {
			return nil, p.unexpectedToken("comma")
		}
		if p.peek.Type == rcurly || p.peek.Type == Newline {
			return nil, p.unexpectedToken("ident")
		}
		p.nextToken()
	}
	if p.curr.Type != rcurly {
		return nil, p.unexpectedToken("rcurly")
	}
	return &t, nil
}

func (p *Parser) nextToken() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == EOF || p.curr.Type == Illegal
}

func (p *Parser) skipNewline() {
	for p.curr.Type == Newline {
		p.nextToken()
	}
}

func (p *Parser) unexpectedToken(want string) error {
	return fmt.Errorf("unexpected token %s (want: %s)", p.curr, want)
}
