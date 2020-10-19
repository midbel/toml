package toml

import (
	"bytes"
	"fmt"
	"io"
)

type Parser struct {
	scan *Scanner
	peek Token
	curr Token

	comment bytes.Buffer
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
			var comment string
			if p.curr.isComment() {
				comment = p.curr.Literal
				p.next()
			}
			t.withComment(p.comment.String(), comment)
			break Loop
		default:
			return p.unexpectedToken("'], .'", "table")
		}
		p.next()
	}
	if !p.curr.isNL() {
		return p.unexpectedToken("'\\n'", "table")
	}
	p.next()
	return p.parseOptions(t)
}

func (p *Parser) parseOptions(t *Table) error {
	for {
		p.parseComment()
		if p.curr.isTable() || p.isDone() {
			break
		}
		if err := p.parseOption(t, true); err != nil {
			return err
		}
		if !p.curr.isNL() {
			return p.unexpectedToken("'\\n'", "body")
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
	if p.peek.Type == TokDot && dotted {
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
		p.next()
	}
	var comment string
	if p.curr.isComment() {
		comment = p.curr.Literal
		p.next()
	}
	opt.withComment(p.comment.String(), comment)
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
	for !p.isDone() && p.curr.Type != TokEndArray {
		var (
			node Node
			err  error
			pre  string
			post string
		)
		p.parseComment()
		pre = p.comment.String()
		switch p.curr.Type {
		case TokBegArray:
			node, err = p.parseArray()
		case TokBegInline:
			node, err = p.parseInline()
		default:
			node, err = p.parseLiteral()
			p.next()
		}
		if err != nil {
			return nil, err
		}
		a.Append(node)

		switch p.curr.Type {
		case TokEndArray:
		case TokComma:
			p.next()
		case TokComment:
		default:
			return nil, p.unexpectedToken("','", "array")
		}
		if p.curr.isComment() {
			post = p.curr.Literal
			p.next()
		}
		node.withComment(pre, post)
		if p.curr.isNL() {
			p.next()
		}
	}
	if p.curr.Type != TokEndArray {
		return nil, p.unexpectedToken("']'", "array")
	}
	p.next()
	return &a, nil
}

func (p *Parser) parseInline() (Node, error) {
	p.next()

	t := Table{
		key:  Token{Pos: p.curr.Pos},
		kind: tableInline,
	}
	for !p.isDone() && p.curr.Type != TokEndInline {
		if err := p.parseOption(&t, false); err != nil {
			return nil, err
		}
		switch p.curr.Type {
		case TokComma:
			p.next()
		case TokEndInline:
		default:
			return nil, p.unexpectedToken("',, }'", "inline")
		}
	}
	if p.curr.Type != TokEndInline {
		return nil, p.unexpectedToken("'}'", "inline")
	}
	p.next()
	return &t, nil
}

func (p *Parser) parseComment() {
	p.comment.Reset()
	for i := 0; p.curr.isComment(); i++ {
		if i > 0 {
			p.comment.WriteRune(newline)
		}
		p.comment.WriteString(p.curr.Literal)
		p.next()
		if p.curr.Type == TokNL {
			p.next()
		}
	}
}

func (p *Parser) next() {
	if p.curr.Type == TokEOF {
		return
	}
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) isDone() bool {
	return p.curr.Type == TokEOF
}

func (p *Parser) unexpectedToken(want, ctx string) error {
	return fmt.Errorf("%s [%s]: unexpected token %s (want: %s)", p.curr.Pos, ctx, p.curr, want)
}
