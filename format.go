package toml

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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

func WithRaw(with bool) FormatRule {
	return func(ft *Formatter) error {
		ft.withRaw = with
		return nil
	}
}

func WithInline(inline bool) FormatRule {
	return func(ft *Formatter) error {
		ft.withInline = inline
		return nil
	}
}

func WithArray(format string) FormatRule {
	return func(ft *Formatter) error {
		switch strings.ToLower(format) {
		case "", "mixed":
			ft.withArray = arrayMixed
		case "multi":
			ft.withArray = arrayMulti
		case "single":
			ft.withArray = arraySingle
		default:
			return fmt.Errorf("%s: unsupported array format", format)
		}
		return nil
	}
}

func WithTime(millis int, utc bool) FormatRule {
	return func(ft *Formatter) error {
		if millis > 9 {
			millis = 9
		}
		pattern := "2006-01-02 15:04:05"
		if millis > 0 {
			pattern += "." + strings.Repeat("0", millis)
		}
		pattern += "-07:00"
		ft.timeconv = formatTime(pattern, utc)
		return nil
	}
}

func WithFloat(format string, underscore int) FormatRule {
	return func(ft *Formatter) error {
		var spec byte
		switch strings.ToLower(format) {
		case "e", "scientific":
			spec = 'e'
		case "", "f", "float":
			spec = 'f'
		case "g", "auto":
			spec = 'g'
		default:
			return fmt.Errorf("%s: unsupported specifier", format)
		}
		ft.floatconv = formatFloat(spec, underscore)
		return nil
	}
}

func WithNumber(format string, underscore int) FormatRule {
	return func(ft *Formatter) error {
		var (
			base   int
			prefix string
		)
		switch strings.ToLower(format) {
		case "x", "hexa", "hex":
			base, prefix = 16, "0x"
		case "o", "octal", "oct":
			base, prefix = 8, "0o"
		case "b", "binary", "bin":
			base, prefix = 2, "0b"
		case "", "d", "decimal", "dec":
			base = 10
		default:
			return fmt.Errorf("%s: unsupported base", format)
		}
		ft.intconv = formatInteger(base, underscore, prefix)
		return nil
	}
}

func WithEOL(format string) FormatRule {
	return func(ft *Formatter) error {
		switch strings.ToLower(format) {
		case "crlf", "windows":
			ft.withEOL = "\r\n"
		case "lf", "linux", "":
			ft.withEOL = "\n"
		default:
			return fmt.Errorf("%s: unsupported eof", format)
		}
		return nil
	}
}

const (
	arrayMixed int = iota
	arraySingle
	arrayMulti
)

type Formatter struct {
	doc    Node
	writer *bufio.Writer

	floatconv func(string) (string, error)
	intconv   func(string) (string, error)
	timeconv  func(string) (string, error)

	withArray   int
	withInline  bool
	withTab     string
	withEOL     string
	withEmpty   bool
	withComment bool
	withNest    bool
	currLevel   int
	withRaw     bool
}

func NewFormatter(doc string, rules ...FormatRule) (*Formatter, error) {
	identity := func(str string) (string, error) {
		return str, nil
	}
	f := Formatter{
		floatconv:   identity,
		intconv:     identity,
		timeconv:    identity,
		withArray:   arrayMixed,
		withInline:  false,
		withEmpty:   false,
		withNest:    false,
		withComment: true,
		withTab:     "\t",
		withEOL:     "\n",
		withRaw:     false,
	}

	buf, err := ioutil.ReadFile(doc)
	if err != nil {
		return nil, err
	}
	f.doc, err = Parse(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	for _, rfn := range rules {
		if err := rfn(&f); err != nil {
			return nil, err
		}
	}
	return &f, nil
}

func (f *Formatter) Format(w io.Writer) error {
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

func (f *Formatter) formatTable(curr *Table, paths []string) error {
	options := curr.listOptions()
	if f.withEmpty || len(options) > 0 {
		f.formatHeader(curr, paths)
		err := f.formatOptions(options, append(paths, curr.key.Literal))
		if err != nil {
			return nil
		}
		f.endLine()
	}
	if !curr.isRoot() && curr.kind.isContainer() {
		paths = append(paths, curr.key.Literal)
	}
	if f.canNest(curr) {
		f.enterLevel(false)
		defer f.leaveLevel(false)
	}
	for _, next := range curr.listTables() {
		if err := f.formatTable(next, paths); err != nil {
			return err
		}
	}
	return nil
}

func (f *Formatter) formatOptions(options []*Option, paths []string) error {
	type table struct {
		prefix string
		*Table
	}
	var (
		length  = longestKey(options)
		array   int
		inlines []table
	)
	for _, o := range options {
		if i, ok := o.value.(*Table); ok && f.withInline {
			i.kind = tableRegular
			i.key = o.key
			i.comment = o.comment
			inlines = append(inlines, table{Table: i})
			continue
		}
		if i, ok := o.value.(*Array); ok && f.withInline {
			var (
				a Array
				t = Table{kind: tableArray, key: o.key}
			)
			for _, n := range i.nodes {
				if nod, ok := n.(*Table); ok {
					nod.key = o.key
					nod.kind = tableItem
					t.nodes = append(t.nodes, nod)
				} else {
					a.nodes = append(a.nodes, n)
				}
			}
			if !t.isEmpty() {
				sub := table{Table: &t}
				if !a.isEmpty() {
					sub.prefix = fmt.Sprintf("\"#%d\"", array)
					array++
				}
				inlines = append(inlines, sub)
			}
			if a.isEmpty() {
				continue
			}
			o.value = &a
		}
		f.formatComment(o.comment.pre, true)
		f.beginLine()
		f.writeKey(o.key.Literal, length)
		if err := f.formatValue(o.value); err != nil {
			return err
		}
		f.formatComment(o.comment.post, false)
		f.endLine()
	}
	if len(inlines) > 0 {
		f.endLine()
		f.enterLevel(false)
		defer f.leaveLevel(false)
		for _, i := range inlines {
			parents := append([]string{}, paths...)
			if i.prefix != "" {
				parents = append(parents, i.prefix)
			}
			if err := f.formatTable(i.Table, parents); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Formatter) formatValue(n Node) error {
	if n == nil {
		return nil
	}
	var err error
	switch n := n.(type) {
	case *Literal:
		if f.withRaw {
			f.writer.WriteString(n.token.Raw)
			break
		}
		err = f.formatLiteral(n)
	case *Array:
		err = f.formatArray(n)
	case *Table:
		err = f.formatInline(n)
	default:
		err = fmt.Errorf("unexpected value type %T", n)
	}
	return err
}

func (f *Formatter) formatLiteral(i *Literal) error {
	if i.token.isString() {
		f.formatString(i.token)
		return nil
	}
	str, err := f.convertValue(i.token)
	if err == nil {
		f.writer.WriteString(str)
	}
	return err
}

func (f *Formatter) formatString(tok Token) {
	var (
		isMulti bool
		quoting string
		escape  func(rune) (rune, bool)
	)
	switch tok.Type {
	case TokBasic:
		escape = escapeBasic
		quoting = "\""
	case TokBasicMulti:
		escape = escapeMulti
		quoting, isMulti = "\"\"\"", true
	case TokLiteral:
		quoting = "'"
	case TokLiteralMulti:
		quoting, isMulti = "'''", true
	default:
		return
	}
	f.writer.WriteString(quoting)
	if isMulti {
		f.endLine()
	}
	str := escapeString(tok.Literal, isMulti, escape)
	if isMulti && strings.IndexByte(str, newline) < 0 {
		str = textWrap(str)
	}
	f.writer.WriteString(str)
	f.writer.WriteString(quoting)
}

func textWrap(str string) string {
	var (
		scan = bufio.NewScanner(strings.NewReader(str))
		buf  strings.Builder
	)
	const (
		length = 72
		limit  = 8
	)
	scan.Split(func(data []byte, ateof bool) (int, []byte, error) {
		if ateof {
			return len(data), data, bufio.ErrFinalToken
		}
		var i, j int
		for i < length {
			x := bytes.IndexAny(data[i:], " \t.?,!;")
			if x < 0 {
				return 0, nil, nil
			}
			j, i = i, i+x+1
		}
		if i >= length+limit {
			i = j
		}
		return i, data[:i], nil
	})
	scan.Scan()
	for {
		buf.WriteString(scan.Text())
		if !scan.Scan() {
			break
		}
		buf.WriteRune(backslash)
		buf.WriteRune(newline)
	}
	return buf.String()
}

func escapeBasic(r rune) (rune, bool) {
	switch r {
	case backslash:
	case dquote:
	case newline:
		r = 'n'
	case tab:
		r = 't'
	case formfeed:
		r = 'f'
	case backspace:
		r = 'b'
	case carriage:
		r = 'r'
	default:
		return r, false
	}
	return r, true
}

func escapeMulti(r rune) (rune, bool) {
	switch r {
	case backslash:
	case dquote:
	case tab:
		r = 't'
	case formfeed:
		r = 'f'
	case backspace:
		r = 'b'
	case carriage:
		r = 'r'
	default:
		return r, false
	}
	return r, true
}

func escapeString(str string, multi bool, escape func(r rune) (rune, bool)) string {
	if escape == nil {
		return str
	}
	var (
		i  int
		b  strings.Builder
		ok bool
	)
	for i < len(str) {
		char, z := utf8.DecodeRuneInString(str[i:])
		if char == utf8.RuneError {
			break
		}
		if multi && char == dquote {
			i += z
			b.WriteRune(char)
			if char, z = utf8.DecodeRuneInString(str[i:]); char == dquote {
				i += z
				b.WriteRune(char)
				char, z = utf8.DecodeRuneInString(str[i:])
			}
		}
		if char, ok = escape(char); ok {
			b.WriteRune(backslash)
		}
		b.WriteRune(char)
		i += z
	}
	return b.String()
}

func (f *Formatter) convertValue(tok Token) (string, error) {
	switch tok.Type {
	default:
		return tok.Literal, nil
	case TokDatetime:
		return f.timeconv(tok.Literal)
	case TokInteger:
		return f.intconv(tok.Literal)
	case TokFloat:
		return f.floatconv(tok.Literal)
	}
}

func (f *Formatter) formatArray(a *Array) error {
	if len(a.nodes) <= 1 || f.withArray == arraySingle {
		return f.formatArrayLine(a)
	}
	if f.withArray == arrayMulti {
		return f.formatArrayMultiline(a)
	}
	if a.isMultiline() {
		return f.formatArrayMultiline(a)
	}
	return f.formatArrayLine(a)
}

func (f *Formatter) formatArrayMultiline(a *Array) error {
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
	f.enterArray()
	defer func() {
		f.leaveArray()
		f.beginLine()
		f.writer.WriteString("]")
	}()

	f.writer.WriteString("[")
	f.endLine()
	for _, n := range a.nodes {
		com := retr(n)
		f.formatComment(com.pre, true)
		f.beginLine()
		if err := f.formatValue(n); err != nil {
			return err
		}
		f.writer.WriteString(",")
		f.formatComment(com.post, false)
		f.endLine()
	}
	return nil
}

func (f *Formatter) formatArrayLine(a *Array) error {
	f.writer.WriteString("[")
	for i, n := range a.nodes {
		if err := f.formatValue(n); err != nil {
			return err
		}
		if i < len(a.nodes)-1 {
			f.writer.WriteString(", ")
		}
	}
	f.writer.WriteString("]")
	return nil
}

func (f *Formatter) formatInline(t *Table) error {
	defer func(array int) {
		f.withArray = array
	}(f.withArray)
	f.withArray = arraySingle
	f.writer.WriteString("{")
	for i, o := range t.listOptions() {
		if i > 0 {
			f.writer.WriteString(", ")
		}
		f.writeKey(o.key.Literal, 0)
		if err := f.formatValue(o.value); err != nil {
			return err
		}
	}
	f.writer.WriteString("}")
	return nil
}

func (f *Formatter) formatHeader(curr *Table, paths []string) error {
	if curr.isRoot() {
		return nil
	}
	if curr.kind != tableItem {
		paths = append(paths, curr.key.Literal)
	}
	f.formatComment(curr.comment.pre, true)
	switch str := strings.Join(paths, "."); curr.kind {
	case tableRegular, tableImplicit:
		f.writeRegularHeader(str)
	case tableItem:
		f.writeArrayHeader(str)
	default:
		return fmt.Errorf("%s: can not write header for %s", curr.kind, str)
	}
	f.formatComment(curr.comment.post, false)
	f.endLine()
	return nil
}

func (f *Formatter) formatComment(comment string, pre bool) error {
	if !f.withComment || comment == "" {
		return nil
	}
	scan := bufio.NewScanner(strings.NewReader(comment))
	for scan.Scan() {
		f.writeComment(scan.Text(), pre)
	}
	return scan.Err()
}

func (f *Formatter) enterArray() {
	f.enterLevel(true)
}

func (f *Formatter) leaveArray() {
	f.leaveLevel(true)
}

func (f *Formatter) enterLevel(force bool) {
	if f.withNest || force {
		f.currLevel++
	}
}

func (f *Formatter) leaveLevel(force bool) {
	if f.withNest || force {
		f.currLevel--
	}
}

func (f *Formatter) canNest(curr *Table) bool {
	if curr.isRoot() {
		return false
	}
	if curr.kind == tableImplicit && !f.withEmpty {
		return false
	}
	return curr.kind.canNest()
}

func (f *Formatter) writeKey(str string, length int) {
	n, _ := f.writer.WriteString(str)
	if length > 0 {
		f.writer.WriteString(strings.Repeat(" ", length-n))
	}
	f.writer.WriteString(" = ")
}

func (f *Formatter) writeComment(str string, pre bool) {
	if pre {
		f.beginLine()
	} else {
		f.writer.WriteString(" ")
	}
	f.writer.WriteString("# ")
	f.writer.WriteString(str)
	if pre {
		f.endLine()
	}
}

func (f *Formatter) writeRegularHeader(str string) {
	f.beginLine()
	f.writer.WriteString("[")
	f.writer.WriteString(str)
	f.writer.WriteString("]")
}

func (f *Formatter) writeArrayHeader(str string) {
	f.beginLine()
	f.writer.WriteString("[[")
	f.writer.WriteString(str)
	f.writer.WriteString("]]")
}

func (f *Formatter) endLine() {
	f.writer.WriteString(f.withEOL)
}

func (f *Formatter) beginLine() {
	if f.currLevel == 0 {
		return
	}
	f.writer.WriteString(strings.Repeat(f.withTab, f.currLevel))
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

func formatString(str string) string {
	return str
}

func formatTime(pattern string, utc bool) func(string) (string, error) {
	if pattern == "" {
		pattern = time.RFC3339
	}
	return func(str string) (string, error) {
		var (
			when time.Time
			err  error
		)
		for _, pat := range makeAllPatterns() {
			when, err = time.Parse(pat, str)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "", err
		}
		if utc {
			when = when.UTC()
		}
		return when.Format(pattern), nil
	}
}

func formatFloat(specifier byte, underscore int) func(string) (string, error) {
	return func(str string) (string, error) {
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return "", err
		}
		str = strconv.FormatFloat(f, specifier, -1, 64)
		return withUnderscore(str, underscore), nil
	}
}

func formatInteger(base, underscore int, prefix string) func(string) (string, error) {
	return func(str string) (string, error) {
		n, err := strconv.ParseInt(str, 0, 64)
		if err != nil {
			return "", err
		}
		str = strconv.FormatInt(n, base)
		return prefix + withUnderscore(str, underscore), nil
	}
}

func withUnderscore(str string, every int) string {
	if every == 0 || len(str) < every {
		return str
	}
	x := strings.Index(str, ".")
	if x < 0 {
		return insertUnderscore(str, every)
	}
	part := insertUnderscore(str[:x], every) + "."
	str = str[x+1:]
	x = strings.IndexAny(str, "eE")
	if x < 0 {
		return part + insertUnderscore(str, every)
	}
	part += insertUnderscore(str[:x], every)
	part += "e"
	if str[x+1] == '+' || str[x+1] == '-' {
		x++
		part += string(str[x])
	}
	return part + insertUnderscore(str[x+1:], every)
}

func insertUnderscore(str string, every int) string {
	if len(str) <= every {
		return str
	}
	var (
		i   = len(str) % every
		buf bytes.Buffer
	)
	if i > 0 {
		buf.WriteString(str[:i])
		buf.WriteString("_")
	}
	for i < len(str) && i+every < len(str) {
		buf.WriteString(str[i : i+every])
		buf.WriteString("_")
		i += every
	}
	buf.WriteString(str[i:])
	return buf.String()
}
