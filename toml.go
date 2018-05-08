package toml

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"
	"time"
)

const (
	dot                = '.'
	comma              = ','
	minus              = '-'
	plus               = '+'
	equal              = '='
	hash               = '#'
	leftSquareBracket  = '['
	rightSquareBracket = ']'
	leftCurlyBracket   = '{'
	rightCurlyBracket  = '}'
)

var booleans = map[string]bool{
	"true":  true,
	"false": false,
}

type Duration struct {
	Value time.Duration
}

func (d *Duration) Set(v string) error {
	a, err := time.ParseDuration(v)
	if err != nil {
		return err
	}
	d.Value = a
	return nil
}

type Setter interface {
	Set(string) error
}

var setterType = reflect.TypeOf((*Setter)(nil)).Elem()

type Decoder struct {
	lex *lexer
}

func NewDecoder(r io.Reader) *Decoder {
	s := new(scanner.Scanner)
	s.Init(bufio.NewReader(r))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanFloats | scanner.ScanInts

	if f, ok := r.(*os.File); ok {
		s.Filename = f.Name()
	}

	return &Decoder{lex: &lexer{Scanner: s}}
}

func (d *Decoder) Decode(v interface{}) error {
	val := reflect.ValueOf(v)
	if k := val.Kind(); k != reflect.Ptr {
		return fmt.Errorf("value is not a ptr (%s)", val.Type())
	}
	d.lex.Scan()
	return parseDocument(d.lex, val.Elem())
}

func parseDocument(lex *lexer, v reflect.Value) error {
	fs := fields(v)
	if lex.token == scanner.Ident {
		if err := parseBody(lex, fs); err != nil {
			return err
		}
	}
	for !lex.Done() {
		switch t := lex.Scan(); t {
		case scanner.Ident:
			f, ok := fs[lex.Text()]
			if !ok {
				return fmt.Errorf("unrecognized table %s", lex.Text())
			}
			if t := lex.Peek(); t == dot {
				if k := f.Kind(); (k == reflect.Slice || k == reflect.Array) && f.Len() > 0 {
					f = f.Index(f.Len() - 1)
				}
			}
			if err := parse(lex, f); err != nil {
				return err
			}
		case leftSquareBracket:
			lex.Scan()
			f, ok := fs[lex.Text()]
			if !ok {
				return fmt.Errorf("unrecognized table %s", lex.Text())
			}
			var (
				z reflect.Value
				a bool
			)
			if k := f.Kind(); k == reflect.Slice || k == reflect.Array {
				if t := lex.Peek(); t == dot && f.Len() > 0 {
					z = f.Index(f.Len() - 1)
				} else {
					z = reflect.New(f.Type().Elem()).Elem()
					a = true
				}
			} else {
				z = f
			}
			if err := parse(lex, z); err != nil {
				return err
			}
			if k := f.Kind(); (k == reflect.Slice || k == reflect.Array) && a {
				f.Set(reflect.Append(f, z))
			}
		default:
			return fmt.Errorf("invalid syntax! expected identifier, got %s (%s)", lex.Text(), lex.Position)
		}
	}
	return nil
}

func parse(lex *lexer, v reflect.Value) error {
	z := v
	if k := v.Kind(); k == reflect.Ptr && v.IsNil() {
		v = reflect.New(z.Type().Elem())
		v = reflect.Indirect(v)

		defer z.Set(v.Addr())
	}

	fs := fields(v)
	if t := lex.Peek(); t == dot {
		lex.Scan()
		lex.Scan()
		f, ok := fs[lex.Text()]
		if !ok {
			return fmt.Errorf("unrecognized table %s", lex.Text())
		}
		if k := f.Kind(); k == reflect.Slice || k == reflect.Array {
			z := reflect.New(f.Type().Elem()).Elem()
			if err := parse(lex, z); err != nil {
				return err
			}
			f.Set(reflect.Append(f, z))
			return nil
		}
		return parse(lex, f)
	}
	if t := lex.Scan(); t != rightSquareBracket {
		return fmt.Errorf("invalid syntax! expected ], got %s", lex.Text())
	}
	for t := lex.Scan(); t == rightSquareBracket; t = lex.Scan() {
	}
	return parseBody(lex, fs)
}

func parseBody(lex *lexer, fs map[string]reflect.Value) error {
	for t := lex.token; t != leftSquareBracket && t != scanner.EOF; t = lex.Scan() {
		f, ok := fs[strings.Trim(lex.Text(), "\"")]
		if !ok {
			return fmt.Errorf("option %q not recognized", lex.Text())
		}
		var set bool
		z := f
		if k := z.Kind(); k == reflect.Ptr && z.IsNil() {
			f = reflect.New(z.Type().Elem())
			f = reflect.Indirect(f)
			set = true
		}
		if err := parseOption(lex, f); err != nil {
			return err
		}
		if set {
			z.Set(f.Addr())
		}
	}
	return nil
}

func parseOption(lex *lexer, v reflect.Value) error {
	if t := lex.Peek(); t == dot {
		lex.Scan()
		if t := lex.Scan(); t != scanner.Ident {
			return fmt.Errorf("invalid syntax! expected option, got %s", lex.Text())
		}
		fs := fields(v)
		f, ok := fs[lex.Text()]
		if !ok {
			return fmt.Errorf("unrecognized option: %s (%s)", lex.Text(), lex.Position)
		}
		return parseOption(lex, f)
	}
	if t := lex.Scan(); t != equal {
		return fmt.Errorf("invalid syntax! expected =, got %s", lex.Text())
	}
	var err error
	switch t := lex.Scan(); t {
	case leftSquareBracket:
		err = parseArray(lex, v)
	case leftCurlyBracket:
		err = parseTable(lex, v)
	default:
		err = parseSimple(lex, v)
	}
	return err
}

func parseArray(lex *lexer, v reflect.Value) error {
	if k := v.Kind(); !(k == reflect.Array || k == reflect.Slice) {
		return fmt.Errorf("array like value expected, got %s", k)
	}
	for t := lex.Scan(); t != rightSquareBracket; t = lex.Scan() {
		if t == comma {
			continue
		}
		f := reflect.New(v.Type().Elem()).Elem()

		var err error
		switch t {
		case leftSquareBracket:
			err = parseArray(lex, f)
		case leftCurlyBracket:
			err = parseTable(lex, f)
		default:
			err = parseSimple(lex, f)
		}
		if err != nil {
			return err
		}
		v.Set(reflect.Append(v, f))
	}
	return nil
}

func parseTable(lex *lexer, v reflect.Value) error {
	z := v
	if k := z.Kind(); k == reflect.Ptr && z.IsNil() {
		v = reflect.New(v.Type().Elem())
		v = reflect.Indirect(v)
	}
	if k := v.Kind(); k != reflect.Struct {
		return fmt.Errorf("struct like value expected, got %s", k)
	}
	fs := fields(v)
	for t := lex.Scan(); t != rightCurlyBracket; t = lex.Scan() {
		if t == comma {
			continue
		}
		f, ok := fs[lex.Text()]
		if !ok {
			return fmt.Errorf("option %s not found", lex.Text())
		}
		if err := parseOption(lex, f); err != nil {
			return err
		}
	}
	if k := z.Kind(); k == reflect.Ptr {
		z.Set(v.Addr())
	} else {
		z.Set(v)
	}
	return nil
}

func parseSimple(lex *lexer, f reflect.Value) error {
	v := lex.Text()

	z := f
	if k := f.Kind(); k != reflect.Ptr && f.CanAddr() {
		z = f.Addr()
	}
	if z.Type().Implements(setterType) {
		if z.IsNil() {
			z.Set(reflect.New(z.Type().Elem()))
		}
		s := z.Interface().(Setter)
		return s.Set(strings.Trim(v, "\""))
	}
	switch t, k := lex.token, f.Kind(); {
	case t == scanner.Ident && k == reflect.Bool:
		f.SetBool(booleans[v])
	case lex.token == scanner.String && k == reflect.String:
		f.SetString(strings.Trim(v, "\""))
	case t == scanner.Int && isUint(k):
		v, _ := strconv.ParseUint(v, 0, 64)
		f.SetUint(v)
	case (t == scanner.Int || t == minus) && isInt(k):
		if t == minus {
			lex.Scan()
			v = lex.Text()
		}
		v, _ := strconv.ParseInt(v, 0, 64)
		if t == minus {
			v = -v
		}
		f.SetInt(v)
	case t == scanner.Int && isTime(f):
		return parseTime(lex, f)
	case t == scanner.Float && isFloat(k):
		v, _ := strconv.ParseFloat(v, 64)
		f.SetFloat(v)
	case t == plus:
		lex.Scan()
		return parseSimple(lex, f)
	default:
		return fmt.Errorf("oups - %s: %s (%s)", v, k, scanner.TokenString(t))
	}
	return nil
}

func fields(v reflect.Value) map[string]reflect.Value {
	fs := make(map[string]reflect.Value)
	if k := v.Kind(); k == reflect.Ptr {
		v = v.Elem()
	}
	for i, t := 0, v.Type(); i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		z := t.Field(i)
		switch n := z.Tag.Get("toml"); n {
		case "-":
			continue
		case "":
			fs[strings.ToLower(z.Name)] = f
		default:
			fs[n] = f
		}
	}
	return fs
}

type lexer struct {
	*scanner.Scanner
	token rune
}

func (l *lexer) Done() bool {
	return l.token == scanner.EOF
}

func (l *lexer) Token() string {
	return scanner.TokenString(l.token)
}

func (l *lexer) Text() string {
	return l.TokenText()
}

func (l *lexer) Scan() rune {
	l.token = l.Scanner.Scan()
	if l.token == hash {
		p := l.Scanner.Position
		for {
			if l.Position.Line > p.Line {
				break
			}
			l.token = l.Scan()
		}
	}
	return l.token
}

func isInt(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUint(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func isTime(v reflect.Value) bool {
	var z time.Time
	return v.Type().AssignableTo(reflect.TypeOf(z))
}

func parseString(lex *lexer, v reflect.Value) error {
	s := lex.Text()
	if s == "\"\"" {
		lex.Error = func(s *scanner.Scanner, m string) {}
		defer func() {
			lex.Error = nil
		}()
		lex.Scan()
		rs := make([]string, 0)
		line := lex.Position.Line
		for t := lex.Scan(); !(t == '"' || lex.Done()); t = lex.Scan() {
			if d := lex.Position.Line - line; d > 0 {
				rs = append(rs, strings.Repeat("\n", d))
				line = lex.Position.Line
			}
			rs = append(rs, lex.Text())
			if t := lex.Peek(); t == ' ' {
				rs = append(rs, " ")
			}
		}
		s = strings.TrimSpace(strings.Join(rs, ""))
	}
	v.SetString(strings.Trim(s, "\""))
	return nil
}

func parseTime(lex *lexer, v reflect.Value) error {
	var ps []string

	for {
		ps = append(ps, lex.Text())
		if lex.Peek() == '\n' {
			break
		}
		lex.Scan()
	}
	t, err := time.Parse(time.RFC3339, strings.Join(ps, ""))
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(t))
	return nil
}

type value struct {
	reflect.Value
	IsSet bool
}
