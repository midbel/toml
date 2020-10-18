package toml

import (
	"bytes"
	"io"
	"io/ioutil"
	"unicode/utf8"
)

const (
	carriage   = '\r'
	newline    = '\n'
	pound      = '#'
	space      = ' '
	tab        = '\t'
	equal      = '='
	dot        = '.'
	comma      = ','
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	plus       = '+'
	minus      = '-'
	underscore = '_'
	colon      = ':'
	backslash  = '\\'
	squote     = '\''
	dquote     = '"'
	backspace  = '\b'
	formfeed   = '\f'
	zero       = '0'
	hex        = 'x'
	oct        = 'o'
	bin        = 'b'
)

var escapes = map[rune]rune{
	backslash: backslash,
	dquote:    dquote,
	'n':       newline,
	't':       tab,
	'f':       formfeed,
	'b':       backspace,
	'r':       carriage,
}

type ScanFunc func(*Scanner) ScanFunc

type Scanner struct {
	pos   int
	next  int
	char  rune
	input []byte
	buf   bytes.Buffer

	line   int
	column int

	cursor Position

	queue chan Token
}

func NewScanner(r io.Reader) (*Scanner, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s := Scanner{
		input:  bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n")),
		line:   1,
		column: 0,
		queue:  make(chan Token),
	}
	s.readRune()
	s.skip(func(r rune) bool { return isBlank(r) || isNL(r) })
	go s.scan()

	return &s, nil
}

func (s *Scanner) Scan() Token {
	tok, ok := <-s.queue
	if !ok {
		tok.Literal = ""
		tok.Type = TokEOF
	}
	return tok
}

func (s *Scanner) backup() {
	s.cursor = Position{
		Line:   s.line,
		Column: s.column,
	}
}

func (s *Scanner) scan() {
	defer close(s.queue)
	scan := scanDefault
	for !s.isDone() {
		s.backup()
		scan = scan(s)
		if scan == nil {
			scan = scanDefault
		}
	}
}

func (s *Scanner) readRune() {
	if s.pos >= len(s.input) {
		s.char = 0
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = 0
		s.next = len(s.input)
	}
	s.char, s.pos, s.next = r, s.next, s.next+n
	if s.char == newline {
		s.line++
		s.column = 0
	}
	s.column++
}

func (s *Scanner) nextRune() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) prevRune() rune {
	r, _ := utf8.DecodeLastRune(s.input[:s.pos])
	return r
}

func (s *Scanner) skip(fn func(rune) bool) {
	s.skipN(0, fn)
}

func (s *Scanner) skipN(n int, fn func(rune) bool) {
	for i := 0; (n <= 0 || i < n) && fn(s.char); i++ {
		s.readRune()
	}
}

func (s *Scanner) writeRune(char rune) {
	s.buf.WriteRune(char)
}

func (s *Scanner) written() int {
	return s.buf.Len()
}

func (s *Scanner) literal() string {
	return s.buf.String()
}

func (s *Scanner) isDone() bool {
	return s.pos >= len(s.input) || isEOF(s.char)
}

func (s *Scanner) emit(kind rune) {
	defer s.buf.Reset()
	s.queue <- Token{
		Literal: s.literal(),
		Type:    kind,
		Pos:     s.cursor,
	}
}

func scanDefault(s *Scanner) ScanFunc {
	// s.skip(func(r rune) bool { return isBlank(r) || isNL(r) })
	s.skip(isBlank)
	switch {
	case s.char == newline:
		s.skip(func(r rune) bool { return isBlank(r) || isNL(r) })
		s.emit(TokNL)
	case s.char == lsquare:
		s.readRune()
		k := TokBegRegularTable
		if s.char == lsquare {
			s.readRune()
			k = TokBegArrayTable
		}
		s.emit(k)
	case s.char == rsquare:
		s.readRune()
		k := TokEndRegularTable
		if s.char == rsquare {
			s.readRune()
			k = TokEndArrayTable
		}
		s.skip(isBlank)
		if !isComment(s.char) && !isNL(s.char) {
			k = TokIllegal
		}
		s.emit(k)
	case s.char == dot:
		s.readRune()
		s.emit(TokDot)
	case s.char == equal:
		s.readRune()
		s.emit(TokEqual)
		return scanValue
	case isDigit(s.char):
		scanDigit(s)
	case isQuote(s.char):
		scanString(s)
	case isLetter(s.char):
		scanIdent(s)
	case isComment(s.char):
		scanComment(s)
	default:
		scanIllegal(s)
	}
	return scanDefault
}

func scanValue(s *Scanner) ScanFunc {
	s.skip(isBlank)
	switch {
	case s.char == lsquare:
		scanArray(s)
	case s.char == lcurly:
		scanInline(s)
	case isQuote(s.char):
		scanString(s)
	case isDigit(s.char) || (isSign(s.char) && isDigit(s.nextRune())):
		if k := s.nextRune(); s.char == zero && (k == hex || k == oct || k == bin) {
			scanBase(s)
		} else {
			scanDecimal(s)
		}
	case isLetter(s.char) || (isSign(s.char) && isLetter(s.nextRune())):
		scanConstant(s)
	default:
		scanIllegal(s)
	}
	s.skip(isBlank)
	if isAlpha(s.char) || isQuote(s.char) {
		scanIllegal(s)
	}
	return nil
}

func scanConstant(s *Scanner) {
	if isSign(s.char) {
		s.writeRune(s.char)
		s.readRune()
	}
	for !s.isDone() && isLetter(s.char) {
		s.writeRune(s.char)
		s.readRune()
	}
	kind := TokIllegal
	if k, ok := constants[s.literal()]; ok {
		kind = k
	}
	s.emit(kind)
}

func scanArray(s *Scanner) {
	s.emit(TokBegArray)
	s.readRune()
	for !s.isDone() {
		s.skip(func(r rune) bool { return isBlank(r) || isNL(r) })
		switch {
		default:
			scanValue(s)
		case s.char == rsquare:
			s.readRune()
			s.emit(TokEndArray)
			return
		case s.char == lsquare:
			scanArray(s)
		case s.char == lcurly:
			scanInline(s)
		case s.char == comma:
			s.readRune()
			s.emit(TokComma)
		case isQuote(s.char):
			scanString(s)
		case isComment(s.char):
			scanComment(s)
		}
	}
}

func scanInline(s *Scanner) {
	s.emit(TokBegInline)
	s.readRune()
	s.skip(isBlank)
	for !s.isDone() {
		switch {
		default:
			scanIllegal(s)
			return
		case s.char == rcurly:
			s.readRune()
			s.emit(TokEndInline)
			return
		case s.char == comma:
			s.readRune()
			s.emit(TokComma)
		case s.char == equal:
			s.readRune()
			s.emit(TokEqual)
			s.skip(isBlank)
			scanValue(s)
		case isLetter(s.char):
			scanIdent(s)
		case isDigit(s.char):
			scanDigit(s)
		case isQuote(s.char):
			scanString(s)
		case isComment(s.char):
			scanComment(s)
		}
		s.skip(isBlank)
	}
}

func scanString(s *Scanner) {
	var (
		quote = s.char
		multi bool
	)
	s.readRune()
	if multi = s.char == quote && s.nextRune() == quote; multi {
		s.skipN(2, isQuote)
		s.skip(func(r rune) bool { return isBlank(r) || isNL(r) })
	}
	for !s.isDone() {
		if s.char == quote {
			s.readRune()
			if !multi {
				break
			}
			if s.char == quote && s.nextRune() == quote {
				s.skipN(2, isQuote)
				break
			}
			s.writeRune(quote)
		}
		if quote == dquote && s.char == backslash {
			switch char := scanEscape(s, multi); char {
			case utf8.RuneError:
				s.emit(TokIllegal)
				return
			case 0:
				continue
			default:
				s.writeRune(char)
				continue
			}
			continue
		}
		s.writeRune(s.char)
		s.readRune()
	}
	kind := TokString
	if s.isDone() {
		kind = TokIllegal
	}
	s.emit(kind)
}

func scanEscape(s *Scanner, multi bool) rune {
	s.readRune()
	if multi && s.char == newline {
		s.readRune()
		s.skip(isBlank)
		return 0
	}
	if s.char == 'u' || s.char == 'U' {
		return scanUnicodeEscape(s)
	}
	if char, ok := escapes[s.char]; ok {
		s.readRune()
		return char
	}
	return utf8.RuneError
}

func scanUnicodeEscape(s *Scanner) rune {
	var (
		char   int32
		offset int32
		step   int32
	)
	if s.char == 'u' {
		step, offset = 4, 12
	} else {
		step, offset = 8, 28
	}
	for i := int32(0); i < step; i++ {
		s.readRune()
		var x rune
		switch {
		case s.char >= '0' && s.char <= '9':
			x = s.char - '0'
		case s.char >= 'a' && s.char <= 'f':
			x = s.char - 'a'
		case s.char >= 'A' && s.char <= 'F':
			x = s.char - 'A'
		default:
			return utf8.RuneError
		}
		char |= x << offset
		offset -= step
	}
	s.readRune()
	return char
}

func scanBase(s *Scanner) {
	var accept func(r rune) bool
	s.writeRune(s.char)
	s.readRune()
	switch s.char {
	case hex:
		accept = isHexa
	case oct:
		accept = isOctal
	case bin:
		accept = isBinary
	default:
		s.emit(TokIllegal)
		return
	}
	s.writeRune(s.char)
	s.readRune()
	for !s.isDone() {
		if s.char == underscore {
			ok := accept(s.prevRune()) && accept(s.nextRune())
			if !ok {
				s.emit(TokIllegal)
				return
			}
			s.readRune()
		}
		if !accept(s.char) {
			break
		}
		s.writeRune(s.char)
		s.readRune()
	}
	s.emit(TokInteger)
}

func scanDecimal(s *Scanner) {
	kind := TokInteger
	if isSign(s.char) {
		if s.char == minus {
			s.writeRune(s.char)
		}
		s.readRune()
	}
	if s.char == zero && isDigit(s.nextRune()) {
		s.readRune()
		if isDigit(s.char) && s.nextRune() != colon {
			s.emit(TokIllegal)
			return
		}
		s.writeRune(zero)
	}
Loop:
	for !s.isDone() {
		switch {
		case s.char == minus:
			kind = scanDate(s)
			break Loop
		case s.char == colon:
			kind = scanTime(s)
			break Loop
		case s.char == underscore:
			ok := isDigit(s.prevRune()) && isDigit(s.nextRune())
			if !ok {
				s.emit(TokIllegal)
				return
			}
		case s.char == dot:
			kind = scanFraction(s)
			break Loop
		case s.char == 'e' || s.char == 'E':
			kind = scanExponent(s)
			break Loop
		case isDigit(s.char):
			s.writeRune(s.char)
		default:
			break Loop
		}
		s.readRune()
	}
	s.emit(kind)
}

func scanDate(s *Scanner) rune {
	scan := func() bool {
		if s.char != minus {
			return false
		}
		s.writeRune(s.char)
		s.readRune()
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.writeRune(s.char)
			s.readRune()
		}
		return true
	}
	if !scan() {
		return TokIllegal
	}
	if !scan() {
		return TokIllegal
	}
	if (s.char == space || s.char == 'T') && isDigit(s.nextRune()) {
		s.writeRune(s.char)
		s.readRune()
		if kind := scanTime(s); kind == TokIllegal {
			return kind
		}
		if kind := scanTimezone(s); kind == TokIllegal {
			return kind
		}
		return TokDatetime
	}
	return TokDate
}

func scanTime(s *Scanner) rune {
	scan := func(check bool) bool {
		if check && s.char != colon {
			return false
		}
		s.writeRune(s.char)
		s.readRune()
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.writeRune(s.char)
			s.readRune()
		}
		return true
	}
	if s.char != colon {
		scan(false)
	}
	if !scan(true) {
		return TokIllegal
	}
	if !scan(true) {
		return TokIllegal
	}
	if s.char == dot {
		s.writeRune(s.char)
		s.readRune()
		n := s.written()
		for isDigit(s.char) {
			s.writeRune(s.char)
			s.readRune()
		}
		if diff := s.written() - n; diff > 9 {
			return TokIllegal
		}
	}
	return TokTime
}

func scanTimezone(s *Scanner) rune {
	if s.char == 'Z' {
		s.writeRune(s.char)
		s.readRune()
		return TokDatetime
	}
	if s.char != plus && s.char != minus {
		return TokDatetime
	}
	scan := func() bool {
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.writeRune(s.char)
			s.readRune()
		}
		return true
	}
	s.writeRune(s.char)
	s.readRune()
	if !scan() {
		return TokIllegal
	}
	s.writeRune(s.char)
	if s.char != colon {
		return TokIllegal
	}
	s.readRune()
	if !scan() {
		return TokIllegal
	}
	return TokDatetime
}

func scanFraction(s *Scanner) rune {
	s.writeRune(s.char)
	s.readRune()
Loop:
	for !s.isDone() {
		switch {
		case s.char == 'e' || s.char == 'E':
			return scanExponent(s)
		case s.char == underscore:
			ok := isDigit(s.prevRune()) && isDigit(s.nextRune())
			if !ok {
				return TokIllegal
			}
		case isDigit(s.char):
			s.writeRune(s.char)
		default:
			break Loop
		}
		s.readRune()
	}
	return TokFloat
}

func scanExponent(s *Scanner) rune {
	s.writeRune(s.char)
	s.readRune()
	if isSign(s.char) {
		s.writeRune(s.char)
		s.readRune()
	}
Loop:
	for !s.isDone() {
		switch {
		case s.char == underscore:
			ok := isDigit(s.prevRune()) && isDigit(s.nextRune())
			if !ok {
				return TokIllegal
			}
		case isDigit(s.char):
			s.writeRune(s.char)
		default:
			break Loop
		}
		s.readRune()
	}
	return TokFloat
}

func scanWhile(s *Scanner, kind rune, accept func(r rune) bool) {
	for !s.isDone() && accept(s.char) {
		s.writeRune(s.char)
		s.readRune()
	}
	s.emit(kind)
}

func scanIdent(s *Scanner) {
	scanWhile(s, TokIdent, isAlpha)
}

func scanDigit(s *Scanner) {
	scanWhile(s, TokIdent, isDigit)
}

func scanComment(s *Scanner) {
	s.readRune()
	s.skip(isBlank)
	scanWhile(s, TokComment, func(r rune) bool { return !isNL(r) })
}

func scanIllegal(s *Scanner) {
	scanWhile(s, TokIllegal, func(r rune) bool { return !isNL(r) })
}

func isHexa(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isOctal(r rune) bool {
	return r >= '0' && r <= '7'
}

func isBinary(r rune) bool {
	return r == '0' || r == '1'
}

func isAlpha(r rune) bool {
	return isDigit(r) || isLetter(r) || r == minus || r == underscore
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSign(r rune) bool {
	return r == minus || r == plus
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isQuote(r rune) bool {
	return r == squote || r == dquote
}

func isComment(r rune) bool {
	return r == pound
}

func isBlank(r rune) bool {
	return r == space || r == tab
}

func isNL(r rune) bool {
	return r == newline
}

func isEOF(r rune) bool {
	return r == 0 || r == utf8.RuneError
}
