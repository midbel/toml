package toml

import (
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	peek Token
	curr Token

	err error
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
	p.skipNewline()

	var t table
	err := p.parseOptions(&t)
	if err != nil {
		return nil, err
	}
	for !p.isDone() {
		if p.curr.Type != lsquare {
			return nil, unexpectedToken(p.curr)
		}
		p.next()
		err := p.parseTable(&t)
		if err != nil {
			return nil, err
		}
	}
	return &t, p.err
}

func (p *Parser) parseTable(t *table) error {
	var (
		a table
		k = regularTable
	)
	if p.curr.Type == lsquare {
		k = itemTable
		p.next()
	}
	a.kind = k
	for p.curr.Type != rsquare && !p.isDone() {
		if !p.curr.IsKey() {
			return unexpectedToken(p.curr)
		}
		k := p.curr
		p.next()
		switch p.curr.Type {
		case dot:
			p.next()
			x, err := t.getOrCreateTable(k)
			if err != nil {
				return err
			}
			t = x
		case rsquare:
			a.key = k
		default:
			return unexpectedToken(p.curr)
		}
	}
	if p.curr.Type != rsquare {
		return unexpectedToken(p.curr)
	}
	p.next()
	if p.curr.Type == rsquare && a.kind != itemTable {
		return unexpectedToken(p.curr)
	} else {
		p.next()
	}
	p.skipNewline()

	err := p.parseOptions(&a)
	if err == nil {
		err = t.appendTable(&a)
	}
	return err
}

func (p *Parser) parseOptions(t *table) error {
	for {
		if p.isDone() || p.curr.Type == lsquare {
			break
		}
		if err := p.parseOption(t, true); err != nil {
			return err
		}
		p.next()

		if !p.isDone() && p.curr.Type != Newline {
			return unexpectedToken(p.curr)
		}
		p.skipNewline()
	}
	return nil
}

func (p *Parser) parseOption(t *table, dotted bool) error {
	var key Token
	for {
		if !p.curr.IsKey() {
			return unexpectedToken(p.curr)
		}
		key = p.curr
		p.next()
		if p.curr.Type == equal {
			break
		} else if p.curr.Type == dot && dotted {
			x, err := t.getOrCreateTable(key)
			if err != nil {
				return err
			}
			t = x
		} else {
			return unexpectedToken(p.curr)
		}
		p.next()
	}
	p.next()

	n, err := p.parseValue()
	if err == nil {
		opt := option{key: key, value: n}
		err = t.appendOption(opt)
	}
	return err
}

func (p *Parser) parseInline() (Node, error) {
	t := table{
		kind: inlineTable,
	}
	for p.consume(rcurly) {
		if p.curr.Type == Newline {
			return nil, unexpectedToken(p.curr)
		}
		if err := p.parseOption(&t, false); err != nil {
			return nil, err
		}
		p.next()
		switch p.curr.Type {
		case comma:
			p.next()
		case rcurly:
		default:
			return nil, unexpectedToken(p.curr)
		}
	}
	return &t, nil
}

func (p *Parser) parseArray() (Node, error) {
	p.skipNewline()

	var (
		arr array
		typ = p.curr.Type
	)
	for p.consume(rsquare) {
		if p.curr.Type != typ {
			return nil, unexpectedToken(p.curr)
		}
		n, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		p.next()
		switch p.curr.Type {
		case comma:
			p.next()
			p.skipNewline()
		case Newline:
			p.skipNewline()
			if p.curr.Type != rsquare {
				return nil, unexpectedToken(p.peek)
			}
		case rsquare:
		default:
			return nil, unexpectedToken(p.curr)
		}
		arr.nodes = append(arr.nodes, n)
	}
	return &arr, nil
}

func (p *Parser) parseValue() (Node, error) {
	var (
		n   Node
		err error
	)
	switch p.curr.Type {
	case lsquare:
		p.next()
		n, err = p.parseArray()
	case lcurly:
		p.next()
		n, err = p.parseInline()
	default:
		n, err = p.parseLiteral()
	}
	return n, err
}

func (p *Parser) parseLiteral() (Node, error) {
	switch p.curr.Type {
	case String:
	case Integer:
	case Float:
	case Date:
	case DateTime:
	case Time:
	case Bool:
	default:
		return nil, unexpectedToken(p.curr)
	}
	return literal{token: p.curr}, nil
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) consume(want rune) bool {
	return !(p.curr.Type == want || p.isDone())
}

func (p *Parser) skipNewline() {
	for p.curr.Type == Newline {
		p.next()
	}
}

func (p *Parser) isDone() bool {
	if p.curr.Type == Illegal {
		p.err = unexpectedToken(p.curr)
	}
	return p.curr.Type == EOF || p.curr.Type == Illegal
}

func unexpectedToken(curr Token) error {
	return fmt.Errorf("%s %w: unexpected token %s", curr.Pos, ErrSyntax, curr)
}

func duplicateOption(opt option) error {
	return fmt.Errorf("%s %w: %s (%s)", opt.key.Pos, ErrDuplicate, opt.key.Literal, opt.value)
}
