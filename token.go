package toml

import (
	"fmt"
)

const (
	EOF rune = -(iota + 1)
	Ident
	String
	Integer
	Float
	Bool
	Date
	Time
	DateTime
	Illegal
	Newline
	Comment
	Punct
)

type Position struct {
	Line   int
	Column int
}

func (p Position) IsValid() bool {
	return p.Line >= 1
}

func (p Position) IsZero() bool {
	if p.IsValid() {
		return false
	}
	return p.Line == 0 && p.Column == 0
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

type Token struct {
	Literal string
	Type    rune
	Pos     Position
}

func (t Token) IsIdent() bool {
	switch t.Type {
	case Ident, String, Integer:
		return true
	default:
		return false
	}
}

func (t Token) IsValid() bool {
	return t.Type != Illegal
}

func (t Token) IsNumber() bool {
	return t.Type == Integer || t.Type == Float
}

func (t Token) IsTime() bool {
	return t.Type == DateTime || t.Type == Date || t.Type == Time
}

func (t Token) String() string {
	var str string
	switch t.Type {
	case Newline:
		return "<newline>"
	case Comment:
		str = "comment"
	case EOF:
		str = "eof"
	case Ident:
		str = "ident"
	case String:
		str = "string"
	case Integer:
		str = "integer"
	case Float:
		str = "float"
	case Bool:
		str = "boolean"
	case DateTime, Date, Time:
		str = "datetime"
	case Illegal:
		str = "unknown"
	default:
		return fmt.Sprintf("<punct(%c)>", t.Type)
	}
	return fmt.Sprintf("<%s(%s)>", str, t.Literal)
}

func tokenString(r rune) string {
	switch r {
	default:
		return "other"
	case lcurly:
		return "table"
	case lsquare:
		return "array"
	case EOF:
		return "eof"
	case Ident:
		return "ident"
	case String:
		return "string"
	case Integer:
		return "integer"
	case Float:
		return "float"
	case Bool:
		return "boolean"
	case Date:
		return "date"
	case Time:
		return "time"
	case DateTime:
		return "datetime"
	case Illegal:
		return "illegal"
	case Comment:
		return "comment"
	}
}
