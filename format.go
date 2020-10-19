package toml

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

type FormatRule func(*Formatter) error

func WithTab(tab int) FormatRule {
	return func(ft *Formatter) error {
		if tab >= 1 {
			ft.withTab = strings.Repeat(" ", tab)
		}
		return nil
	}
}

func WithEmpty(with bool) FormatRule {
	return func(ft *Formatter) error {
		ft.withEmpty = with
		return nil
	}
}

func WithNest(with bool) FormatRule {
	return func(ft *Formatter) error {
		ft.withNest = with
		return nil
	}
}

func WithComment(with bool) FormatRule {
	return func(ft *Formatter) error {
		ft.withComment = with
		return nil
	}
}

func WithFloat(format string) FormatRule {
	return func(ft *Formatter) error {
		var spec byte
		switch strings.ToLower(format) {
		case "e":
			spec = 'e'
		case "", "f":
			spec = 'f'
		case "g":
			spec = 'g'
		default:
			return fmt.Errorf("%s: unsupported specifier", format)
		}
		ft.floatconv = formatFloat(spec)
		return nil
	}
}

func WithNumber(format string) FormatRule {
	return func(ft *Formatter) error {
		var (
			base   int
			prefix string
		)
		switch strings.ToLower(format) {
		case "x":
			base, prefix = 16, "0x"
		case "o":
			base, prefix = 8, "0o"
		case "b":
			base, prefix = 2, "0b"
		case "", "d":
			base = 10
		default:
			return fmt.Errorf("%s: unsupported base", format)
		}
		ft.intconv = formatInteger(base, prefix)
		return nil
	}
}

func WithTime(utc bool, millis int) FormatRule {
	return func(ft *Formatter) error {
		return nil
	}
}

type Formatter struct {
	doc    Node
	level  int
	writer *bufio.Writer

	floatconv func(string) (string, error)
	intconv   func(string) (string, error)
	timeconv  func(string) (string, error)

	withTab     string
	withEmpty   bool
	withNest    bool
	withComment bool
}

func NewFormatter(doc string, rules ...FormatRule) (Formatter, error) {
	identity := func(str string) (string, error) {
		return str, nil
	}
	f := Formatter{
		floatconv:   identity,
		intconv:     identity,
		timeconv:    identity,
		withTab:     "\t",
		withEmpty:   false,
		withNest:    false,
		withComment: true,
	}

	buf, err := ioutil.ReadFile(doc)
	if err != nil {
		return f, err
	}
	f.doc, err = Parse(bytes.NewReader(buf))
	if err != nil {
		return f, err
	}
	for _, rfn := range rules {
		if err := rfn(&f); err != nil {
			return f, err
		}
	}
	return f, nil
}

func (f Formatter) Format(w io.Writer) error {
	f.writer = bufio.NewWriter(w)
	root, ok := f.doc.(*Table)
	if !ok {
		return fmt.Errorf("document not parsed properly")
	}
	if err := f.formatTable(root, nil); err != nil {
		return err
	}
	return f.writer.Flush()
}

func (f Formatter) formatTable(curr *Table, paths []string) error {
	options := curr.listOptions()
	if f.withEmpty || len(options) > 0 {
		f.formatHeader(curr, paths)
		err := f.formatOptions(options)
		if err != nil {
			return nil
		}
		fmt.Fprintln(f.writer)
	}
	if !curr.isRoot() && curr.kind.isContainer() {
		paths = append(paths, curr.key.Literal)
	}
	for _, next := range curr.listTables() {
		if err := f.formatTable(next, paths); err != nil {
			return err
		}
	}
	return nil
}

func (f Formatter) formatOptions(options []*Option) error {
	length := longestKey(options)
	for _, o := range options {
		f.formatComment(o.comment.pre, "\n", true)
		if _, err := fmt.Fprintf(f.writer, "%-*s = ", length, o.key.Literal); err != nil {
			return err
		}
		if err := f.formatValue(o.value); err != nil {
			return err
		}
		f.formatComment(o.comment.post, "", false)
		fmt.Fprintln(f.writer)
	}
	return nil
}

func (f Formatter) formatValue(n Node) error {
	if n == nil {
		return nil
	}
	var err error
	switch n := n.(type) {
	case *Literal:
		pat := "%s"
		if n.token.Type == TokString {
			pat = "\"%s\""
		}
		str, err := f.convertValue(n.token)
		if err != nil {
			return err
		}
		fmt.Fprintf(f.writer, pat, str)
	case *Array:
		err = f.formatArray(n)
	case *Table:
		err = f.formatInline(n)
	default:
		err = fmt.Errorf("unexpected value type %T", n)
	}
	return err
}

func (f Formatter) convertValue(tok Token) (string, error) {
	switch tok.Type {
	default:
		return tok.Literal, nil
	case TokInteger:
		return f.intconv(tok.Literal)
	case TokFloat:
		return f.floatconv(tok.Literal)
	}
}

func (f Formatter) formatArray(a *Array) error {
	retr := func(n Node) comment {
		var c comment
		switch n := n.(type) {
		case *Literal:
			c = n.comment
		case *Array:
			c = n.comment
		case *Table:
			c = n.comment
		}
		return c
	}
	multi := a.isMultiline()
	fmt.Fprint(f.writer, "[")
	for i, n := range a.nodes {
		if multi {
			fmt.Fprint(f.writer, "\n\t")
		} else if !multi && i > 0 {
			fmt.Fprint(f.writer, " ")
		}
		com := retr(n)
		f.formatComment(com.pre, "\n\t", true)
		if err := f.formatValue(n); err != nil {
			return err
		}
		if i < len(a.nodes)-1 || multi {
			fmt.Fprint(f.writer, ",")
		}
		f.formatComment(com.post, "", false)
	}
	if multi {
		fmt.Fprintln(f.writer)
	}
	fmt.Fprint(f.writer, "]")
	return nil
}

func (f Formatter) formatInline(t *Table) error {
	fmt.Fprint(f.writer, "{")
	for i, o := range t.listOptions() {
		if i > 0 {
			fmt.Fprint(f.writer, ", ")
		}
		fmt.Fprintf(f.writer, "%s = ", o.key.Literal)
		if err := f.formatValue(o.value); err != nil {
			return err
		}
	}
	fmt.Fprint(f.writer, "}")
	return nil
}

func (f Formatter) formatHeader(curr *Table, paths []string) error {
	if curr.isRoot() {
		return nil
	}
	var pat string
	switch curr.kind {
	case tableRegular:
		pat = "[%s]"
	case tableArray:
	case tableItem:
		pat = "[[%s]]"
	}
	if curr.kind != tableItem {
		paths = append(paths, curr.key.Literal)
	}
	f.formatComment(curr.comment.pre, "\n", true)
	fmt.Fprintf(f.writer, pat, strings.Join(paths, "."))
	f.formatComment(curr.comment.post, "", false)
	fmt.Fprintln(f.writer)
	return nil
}

func (f Formatter) formatComment(comment, eol string, pre bool) error {
	if !f.withComment || comment == "" {
		return nil
	}
	var (
		scan = bufio.NewScanner(strings.NewReader(comment))
		pat  = " # %s"
	)
	if pre {
		pat = strings.TrimSpace(pat)
	}
	for scan.Scan() {
		fmt.Fprintf(f.writer, pat, scan.Text())
		if eol != "" {
			fmt.Fprint(f.writer, eol)
		}
	}
	return nil
}

func longestKey(options []*Option) int {
	var length int
	for _, o := range options {
		n := len(o.key.Literal)
		if length == 0 || length < n {
			length = n
		}
	}
	return length
}

func formatFloat(specifier byte) func(string) (string, error) {
	return func(str string) (string, error) {
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(f, specifier, -1, 64), nil
	}
}

func formatInteger(base int, prefix string) func(string) (string, error) {
	return func(str string) (string, error) {
		n, err := strconv.ParseInt(str, 0, 64)
		if err != nil {
			return "", err
		}
		str = strconv.FormatInt(n, base)
		return prefix + str, nil
	}
}
