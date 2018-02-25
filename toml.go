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
	d.lex.Scan()
	return parse(d.lex, reflect.ValueOf(v).Elem())
}

func parse(lex *lexer, v reflect.Value) error {
	fs := fields(v)
	if lex.token == scanner.Ident {
		if err := parseBody(lex, fs); err != nil {
			return err
		}
	}
	for t := lex.Scan(); t != scanner.EOF; t = lex.Scan() {
		f, ok := fs[lex.Text()]
		if !ok {
			return fmt.Errorf("table %q not recognized", lex.Text())
		}
		if t := lex.Peek(); t != rightSquareBracket {
			return fmt.Errorf("invalid syntax! expected ], got %s", lex.Token())
		}
		for t := lex.Scan(); t == rightSquareBracket; t = lex.Scan() {
		}
		if k := f.Kind(); k == reflect.Slice || k == reflect.Array {
			v := reflect.New(f.Type().Elem()).Elem()
			fs := fields(v)
			if err := parseBody(lex, fs); err != nil {
				return err
			}
			f.Set(reflect.Append(f, v))
		} else {
			var set bool

			z := f
			if k := z.Kind(); k == reflect.Ptr && z.IsNil() {
				f = reflect.New(z.Type().Elem())
				f = reflect.Indirect(f)
				set = true
			}
			fs := fields(f)
			if err := parseBody(lex, fs); err != nil {
				return err
			}
			if set {
				z.Set(f.Addr())
			}
		}
		lex.Scan()
	}
	return nil
}

func parseBody(lex *lexer, fs map[string]reflect.Value) error {
	for t := lex.token; t != leftSquareBracket && t != scanner.EOF; t = lex.Scan() {
		f, ok := fs[lex.Text()]
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
			return fmt.Errorf("option %q not recognized", lex.Text())
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
	z.Set(v.Addr())
	return nil
}

func parseSimple(lex *lexer, f reflect.Value) error {
	v := lex.Text()
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
