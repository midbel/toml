package scan

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	EOF rune = -(iota + 1)
	Ident
	String
	Char
	Int
	Uint
	Float
	Decimal
	Punct
	DateTime
	Date
	Time
)

const (
	Dot                = '.'
	Equal              = '='
	Comma              = ','
	LeftCurlyBracket   = '{'
	RightCurlyBracket  = '}'
	LeftSquareBracket  = '['
	RightSquareBracket = ']'
)

const (
	hash   = '#'
	minus  = '-'
	plus   = '+'
	space  = ' '
	colon  = ':'
	scolon = ';'
	nl     = '\n'
	squote = '\''
	dquote = '"'
)

type Scanner struct {
	Last   rune
	offset int
	buffer []byte

	token bytes.Buffer
	Position
}

func NewScanner(r io.Reader) *Scanner {
	var s Scanner
	s.Reset(r)

	return &s
}

func (s *Scanner) Text() string {
	return s.token.String()
}

// func (s *Scanner) Peek() rune {
// 	offset := s.offset
// 	for {
// 		r, z := utf8.DecodeRune(s.buffer[offset:])
// 		if r == utf8.RuneError {
// 			return EOF
// 		}
// 		offset += z
// 		switch {
// 		case isWhitespace(r):
// 		case r == hash:
// 			for r != nl {
// 				r, z := utf8.DecodeRune(s.buffer[offset:])
// 				if r == utf8.RuneError {
// 					return EOF
// 				}
// 				offset += z
// 			}
// 		case isString(r):
// 			return String
// 		case isIdent(r):
// 			return Ident
// 		case isDigit(r):
// 			return Decimal
// 		default:
// 			return r
// 		}
// 	}
// }

func (s *Scanner) peek() rune {
	offset := s.offset
	for {
		r, z := utf8.DecodeRune(s.buffer[offset:])
		if r == utf8.RuneError {
			return EOF
		}
		offset += z
		switch {
		case isWhitespace(r):
		case r == hash:
			for r != nl {
				r, z := utf8.DecodeRune(s.buffer[offset:])
				if r == utf8.RuneError {
					return EOF
				}
				offset += z
			}
		default:
			return r
		}
	}
}

func (s *Scanner) Peek() rune {
	switch r := s.peek(); {
	case isString(r):
		return String
	case isIdent(r):
		return Ident
	case isDigit(r):
		return Decimal
	default:
		return r
	}
}

func (s *Scanner) Scan() rune {
	r := s.scanRune()
	switch {
	case isWhitespace(r):
		r = s.skipWhitespace()
	case isComment(r):
		r = s.skipComment()
	}

	s.Offset = s.offset - 1
	s.token.Reset()
	switch {
	case isIdent(r):
		s.Last = s.scanIdent(r)
	case isString(r):
		s.Last = s.scanString(r, r)
	case isDigit(r) || r == minus:
		s.Last = s.scanDecimal(r)
	case r == plus:
		s.Last = s.scanDecimal(s.scanRune())
		if s.Last != Float {
			s.Last = Uint
		}
	default:
		s.Last = r
	}
	return s.Last
}

func (s *Scanner) Reset(r io.Reader) {
	var w bytes.Buffer

	n, _ := io.Copy(&w, r)
	if n > 0 {
		s.buffer = make([]byte, n)
		copy(s.buffer, w.Bytes())
	}
	s.offset = 0
}

func (s *Scanner) scanNumber(r rune, accept func(rune) bool) rune {
	if accept == nil {
		accept = func(r rune) bool {
			return isDigit(r) || r == '_'
		}
	}
	s.token.WriteRune(r)
	for {
		r = s.scanRune()
		if !accept(r) {
			if r == EOF {
				break
			}
			s.offset -= utf8.RuneLen(r)
			return r
		}
		if r != plus {
			s.token.WriteRune(r)
		}
	}
	return Int
}

func (s *Scanner) scanDecimal(r rune) rune {
	s.token.WriteRune(r)
	switch n := s.peek(); n {
	case 'x':
		return s.scanNumber(s.scanRune(), isHexRune)
	case 'o':
		return s.scanNumber(s.scanRune(), isOctalRune)
	case 'b':
		return s.scanNumber(s.scanRune(), isBinRune)
	}
	for {
		switch r = s.scanRune(); {
		case r == colon:
			return s.scanTime(r)
		case r == minus:
			return s.scanDate(r)
		case r == '_' || isDigit(r):
			s.token.WriteRune(r)
		case r == Dot:
			s.token.WriteRune(r)
			s.scanNumber(s.scanRune(), nil)
			if r = s.scanRune(); r == 'e' || r == 'E' {
				s.scanNumber(r, func(r rune) bool {
					return isDigit(r) || r == minus || r == plus
				})
			} else {
				s.offset -= utf8.RuneLen(r)
			}
			return Float
		case r == 'e' || r == 'E':
			s.scanNumber(r, func(r rune) bool {
				return isDigit(r) || r == minus || r == plus
			})
			return Float
		default:
			s.offset -= utf8.RuneLen(r)
			return Int
		}
	}
}

func (s *Scanner) scanDate(r rune) rune {
	s.token.WriteRune(r)
	for {
		switch r = s.scanRune(); {
		case r == ' ' || r == 'T':
			r = s.scanTime(r)
			switch r {
			case 'Z':
				s.token.WriteRune(r)
			case minus:
				s.scanTime(r)
			case Time:
			default:
				return EOF
			}
			return DateTime
		case isDigit(r) || r == minus:
			s.token.WriteRune(r)
		case r == nl || r == EOF:
			return Date
		default:
			return EOF
		}
	}
}

func (s *Scanner) scanTime(r rune) rune {
	s.token.WriteRune(r)
	for {
		switch r = s.scanRune(); {
		case r == nl || r == EOF:
			s.offset -= utf8.RuneLen(r)
			return Time
		case isDigit(r) || r == Dot || r == colon:
			s.token.WriteRune(r)
		default:
			return r
		}
	}
}

func (s *Scanner) scanIdent(r rune) rune {
	s.token.WriteRune(r)
	for {
		if r = s.scanRune(); r == EOF {
			return EOF
		}
		if !isIdentRune(r) {
			s.offset -= utf8.RuneLen(r)
			return Ident
		}
		s.token.WriteRune(r)
	}
}

func (s *Scanner) scanString(r, q rune) rune {
	s.token.WriteRune(r)
	for {
		r = s.scanRune()
		switch r {
		case EOF:
			return r
		case '\\':
			r = s.scanRune()
		}
		s.token.WriteRune(r)
		if r == q {
			break
		}
	}
	if n := s.Peek(); n == String {
		return s.scanString(s.scanRune(), q)
	}
	return String
}

func (s *Scanner) scanRune() rune {
	if s.offset >= len(s.buffer) {
		return EOF
	}
	r, z := utf8.DecodeRune(s.buffer[s.offset:])
	if r == utf8.RuneError {
		return EOF
	}
	s.offset += z
	return r
}

func (s *Scanner) skipWhitespace() rune {
	for {
		r := s.scanRune()
		if r == EOF {
			return EOF
		}
		if !isWhitespace(r) {
			if isComment(r) {
				return s.skipComment()
			}
			return r
		}
	}
}

func (s *Scanner) skipComment() rune {
	for {
		r := s.scanRune()
		if r == EOF {
			return EOF
		}
		if r == nl {
			return s.skipWhitespace()
		}
	}
}

var tokenTypes = map[rune]string{
	EOF:      "eof",
	Ident:    "ident",
	String:   "string",
	Int:      "int",
	Float:    "float",
	Decimal:  "decimal",
	DateTime: "datetime",
	Date:     "date",
	Time:     "time",
}

func TokenString(r rune) string {
	v, ok := tokenTypes[r]
	if ok {
		return v
	}
	return fmt.Sprintf("%q", r)
}

type Position struct {
	Line   int
	Column int
	Offset int
}

func (p Position) String() string {
	return fmt.Sprintf("<%d:%d>", p.Line, p.Column)
}

func isComment(r rune) bool {
	return r == hash
}

func isString(r rune) bool {
	return r == '\'' || r == '"'
}

func isIdent(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isIdentRune(r rune) bool {
	return isIdent(r) || r == '-' || r == '_' || isDigit(r)
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func isHexRune(r rune) bool {
	return r == '_' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'Z')
}

func isOctalRune(r rune) bool {
	return r == '_' || ('0' <= r && r <= '7')
}

func isBinRune(r rune) bool {
	return r == '_' || r == '0' || r == '1'
}
