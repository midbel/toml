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
	t := Table{
		key:  Token{Literal: "root"},
		kind: typeRegular,
	}

	p.skipNewline()
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
	p.nextToken()
	a, t, err := p.parseHeaders(t)
	if err != nil {
		return err
	}
	if err = p.parseOptions(a); err == nil {
		err = t.Append(a)
	}
	return err
}

func (p *Parser) parseHeaders(t *Table) (*Table, *Table, error) {
	kind := typeRegular
	if p.curr.Type == lsquare {
		kind = typeItem
		p.nextToken()
	}

	var key Token
	for p.curr.Type != rsquare {
		if !p.curr.IsIdent() {
			return nil, nil, p.unexpectedToken("ident")
		}
		key = p.curr
		p.nextToken()
		switch p.curr.Type {
		case dot:
			p.nextToken()
			x, err := t.GetOrCreate(key.Literal)
			if err != nil {
				return nil, nil, err
			}
			t = x
		case rsquare:
		default:
			return nil, nil, p.unexpectedToken("dot/rsquare")
		}
	}
	p.nextToken()
	if kind == typeItem {
		if p.curr.Type != rsquare {
			return nil, nil, p.unexpectedToken("rsquare")
		}
		p.nextToken()
	}
	if p.curr.Type != Newline {
		return nil, nil, p.unexpectedToken("newline")
	} else {
		p.skipNewline()
	}

	a := Table{
		key:  key,
		kind: kind,
	}

	return &a, t, nil
}

func (p *Parser) parseOptions(t *Table) error {
	for !p.isDone() {
		if p.curr.Type == lsquare {
			break
		}
		if err := p.parseOption(t, true); err != nil {
			return err
		}
		p.nextToken()
		if p.curr.Type != Newline {
			return p.unexpectedToken("newline")
		}
		p.skipNewline()
	}
	if p.curr.Type != lsquare && !p.isDone() {
		return p.unexpectedToken("lsquare")
	}
	return nil
}

func (p *Parser) parseOption(t *Table, dotted bool) error {
	if !p.curr.IsIdent() {
		return p.unexpectedToken("ident")
	}
	if p.peek.Type == dot {
		if !dotted {
			return p.unexpectedToken("equal")
		}
		x, err := t.GetOrCreate(p.curr.Literal)
		if err != nil {
			return err
		}
		x.kind = typeRegular

		p.nextToken()
		p.nextToken()
		return p.parseOption(x, dotted)
	}
	var (
		opt = Option{key: p.curr}
		err error
	)
	p.nextToken()
	if p.curr.Type != equal {
		return p.unexpectedToken("equal")
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
	if err == nil {
		err = t.Append(&opt)
	}
	return err
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

	a := Array{pos: p.curr.Pos}
	for !p.isDone() {
		p.skipNewline()
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
		a.Append(node)

		p.nextToken()
		p.skipNewline()
		if p.curr.Type == rsquare {
			break
		}
		if p.curr.Type != comma {
			return nil, p.unexpectedToken("comma")
		}
		p.nextToken()
		p.skipNewline()
	}
	if p.curr.Type != rsquare {
		return nil, p.unexpectedToken("rsquare")
	}
	return &a, nil
}

func (p *Parser) parseInline() (Node, error) {
	p.nextToken()

	t := Table{
		key:  Token{Pos: p.curr.Pos},
		kind: typeInline,
	}
	for !p.isDone() {
		if p.curr.Type == rcurly {
			break
		}
		if err := p.parseOption(&t, false); err != nil {
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
	return fmt.Errorf("%s: unexpected token %s (want: %s)", p.curr.Pos, p.curr, want)
}
