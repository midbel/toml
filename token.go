package toml

import (
	"fmt"
)

const (
	TokEOF rune = -(iota) + 1
	TokIdent
	TokString
	TokInteger
	TokFloat
	TokBool
	TokDate
	TokDatetime
	TokTime
	TokComment
	TokIllegal
	TokBegArray
	TokEndArray
	TokBegInline
	TokEndInline
	TokBegRegularTable
	TokEndRegularTable
	TokBegArrayTable
	TokEndArrayTable
	TokEqual
	TokDot
	TokComma
	TokNewline
)

var constants = map[string]rune{
	"true":  TokBool,
	"false": TokBool,
	"inf":   TokFloat,
	"+inf":  TokFloat,
	"-inf":  TokFloat,
	"nan":   TokFloat,
	"+nan":  TokFloat,
	"-nan":  TokFloat,
}

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

func (t Token) isZero() bool {
	return t.Literal == "" && t.Type == 0
}

func (t Token) isComment() bool {
	return t.Type == TokComment
}

func (t Token) isTable() bool {
	return t.Type == TokBegRegularTable || t.Type == TokBegArrayTable
}

func (t Token) IsIdent() bool {
	switch t.Type {
	case TokIdent, TokString, TokInteger:
		return true
	default:
		return false
	}
}

func (t Token) isValue() bool {
	switch t.Type {
	case TokString, TokInteger, TokFloat, TokBool, TokDate, TokTime, TokDatetime:
		return true
	default:
		return false
	}
}

func (t Token) IsValid() bool {
	return t.Type != TokIllegal
}

func (t Token) IsNumber() bool {
	return t.Type == TokInteger || t.Type == TokFloat
}

func (t Token) IsTime() bool {
	return t.Type == TokDatetime || t.Type == TokDate || t.Type == TokTime
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	default:
		prefix = "unknown"
	case TokEOF:
		return "<eof>"
	case TokIdent:
		prefix = "ident"
	case TokString:
		prefix = "string"
	case TokInteger:
		prefix = "integer"
	case TokFloat:
		prefix = "float"
	case TokBool:
		prefix = "boolean"
	case TokDate:
		prefix = "date"
	case TokDatetime:
		prefix = "datetime"
	case TokTime:
		prefix = "time"
	case TokComment:
		prefix = "comment"
	case TokIllegal:
		prefix = "illegal"
	case TokBegArray:
		return "<begin-array>"
	case TokEndArray:
		return "<end-array>"
	case TokBegInline:
		return "<begin-inline>"
	case TokEndInline:
		return "<end-inline>"
	case TokBegRegularTable:
		return "<begin-regular-table>"
	case TokEndRegularTable:
		return "<end-regular-table>"
	case TokBegArrayTable:
		return "<begin-array-table>"
	case TokEndArrayTable:
		return "<end-array-table>"
	case TokEqual:
		return "<equal>"
	case TokDot:
		return "<dot>"
	case TokComma:
		return "<comma>"
	case TokNewline:
		return "<newline>"
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Literal)
}
