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
	s, err := NewScanner(r)
	if err != nil {
		return nil, err
	}

	p := Parser{scan: s}
	p.next()
	p.next()

	return p.Parse()
}

func (p *Parser) Parse() (Node, error) {
	t := Table{
		key:  Token{Literal: "default", Type: TokIdent},
		kind: tableRegular,
	}

	if err := p.parseOptions(&t); err != nil {
		return nil, err
	}
	for !p.isDone() {
		if !p.curr.isTable() {
			return nil, p.unexpectedToken("'[, [['", "parse")
		}
		kind := tableRegular
		if p.curr.Type == TokBegArrayTable {
			kind = tableItem
		}
		p.next()
		if err := p.parseTable(&t, kind); err != nil {
			return nil, err
		}
	}
	return &t, nil
}

func (p *Parser) parseTable(t *Table, kind tableType) error {
Loop:
	for !p.isDone() {
		if !p.curr.IsIdent() {
			return p.unexpectedToken("ident", "table")
		}
		switch p.peek.Type {
		case TokDot:
			x, err := t.retrieveTable(p.curr)
			if err != nil {
				return err
			}
			t = x
			p.next()
		case TokEndRegularTable, TokEndArrayTable:
			x := &Table{
				key:  p.curr,
				kind: kind,
			}
			if err := t.registerTable(x); err != nil {
				return err
			}
			t = x
			if t.kind == tableItem && p.peek.Type != TokEndArrayTable {
				return p.unexpectedToken("']'", "table")
			}
			p.next()
			p.next()
			break Loop
		default:
			return p.unexpectedToken("'], .'", "table")
		}
		p.next()
	}
	return p.parseOptions(t)
}

func (p *Parser) parseOptions(t *Table) error {
	for !p.isDone() {
		if p.curr.isTable() {
			break
		}
		if err := p.parseOption(t, true); err != nil {
			return err
		}
		p.next()
	}
	if !p.curr.isTable() && !p.isDone() {
		return p.unexpectedToken("'[, [['", "body")
	}
	return nil
}

func (p *Parser) parseOption(t *Table, dotted bool) error {
	if !p.curr.IsIdent() {
		return p.unexpectedToken("ident", "option")
	}
	if p.peek.Type == TokDot {
		if !dotted {
			return fmt.Errorf("dot not allow")
		}
		x, err := t.retrieveTable(p.curr)
		if err != nil {
			return err
		}
		p.next()
		p.next()
		return p.parseOption(x, dotted)
	}
	var (
		opt = Option{key: p.curr}
		err error
	)
	p.next()
	if p.curr.Type != TokEqual {
		return p.unexpectedToken("'='", "option")
	}
	p.next()
	switch p.curr.Type {
	case TokBegArray:
		opt.value, err = p.parseArray()
	case TokBegInline:
		opt.value, err = p.parseInline()
	default:
		opt.value, err = p.parseLiteral()
	}
	if err == nil {
		err = t.registerOption(&opt)
	}
	return err
}

func (p *Parser) parseLiteral() (Node, error) {
	if !p.curr.isValue() {
		return nil, p.unexpectedToken("literal", "value")
	}
	lit := Literal{
		token: p.curr,
	}
	return &lit, nil
}

func (p *Parser) parseArray() (Node, error) {
	p.next()

	a := Array{pos: p.curr.Pos}
	for !p.isDone() {
		if p.curr.Type == TokEndArray {
			break
		}
		var (
			node Node
			err  error
		)
		switch p.curr.Type {
		case TokBegArray:
			node, err = p.parseArray()
		case TokBegInline:
			node, err = p.parseInline()
		default:
			node, err = p.parseLiteral()
		}
		if err != nil {
			return nil, err
		}
		a.Append(node)

		p.next()
		switch p.curr.Type {
		case TokEndArray:
		case TokComma:
			p.next()
		default:
			return nil, p.unexpectedToken("','", "array")
		}
	}
	if p.curr.Type != TokEndArray {
		return nil, p.unexpectedToken("']'", "array")
	}
	return &a, nil
}

func (p *Parser) parseInline() (Node, error) {
	p.next()

	t := Table{
		key:  Token{Pos: p.curr.Pos},
		kind: tableInline,
	}
	for !p.isDone() {
		if p.curr.Type == TokEndInline {
			break
		}
		if err := p.parseOption(&t, false); err != nil {
			return nil, err
		}
		p.next()
		if p.curr.Type == TokEndInline {
			break
		}
		if p.curr.Type != TokComma {
			return nil, p.unexpectedToken("','", "inline")
		}
		if p.peek.Type == TokEndInline {
			return nil, p.unexpectedToken("ident", "inline")
		}
		p.next()
	}
	if p.curr.Type != TokEndInline {
		return nil, p.unexpectedToken("'}'", "inline")
	}
	return &t, nil
}

func (p *Parser) next() {
	if p.curr.Type == TokEOF {
		return
	}
	p.curr = p.peek
	p.peek = p.scan.Scan()

	for p.curr.isComment() {
		p.curr = p.peek
		p.peek = p.scan.Scan()
	}
}

func (p *Parser) isDone() bool {
	return p.curr.Type == TokEOF
}

func (p *Parser) unexpectedToken(want, ctx string) error {
	return fmt.Errorf("[%s] %s: unexpected token %s (want: %s)", ctx, p.curr.Pos, p.curr, want)
}
