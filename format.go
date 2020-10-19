package toml

import (
  "bufio"
	"fmt"
	"io"
	"strings"
)

func Format(r io.Reader, w io.Writer) error {
  ws := bufio.NewWriter(w)
  defer ws.Flush()
	doc, err := Parse(r)
	if err != nil {
		return err
	}
	t, ok := doc.(*Table)
	if !ok {
		return fmt.Errorf("document not parsed properly")
	}
	return formatTable(t, nil, ws)
}

func formatTable(t *Table, parents []string, w io.Writer) error {
	options := t.listOptions()
	if len(options) > 0 {
		formatHeader(t, parents, w)
		err := formatOptions(options, w)
		if err != nil {
			return nil
		}
		fmt.Fprintln(w)
	}
	if !t.isRoot() && t.kind.isContainer() {
		parents = append(parents, t.key.Literal)
	}
	for _, t := range t.listTables() {
		if err := formatTable(t, parents, w); err != nil {
			return err
		}
	}
	return nil
}

func formatHeader(t *Table, parents []string, w io.Writer) error {
	if t.isRoot() {
		return nil
	}
  formatComment(t.comment.pre, "\n", w)
	var pattern string
	switch t.kind {
	case tableRegular:
		pattern = "[%s]"
	case tableArray:
	case tableItem:
		pattern = "[[%s]]"
	}
	if t.kind != tableItem {
		parents = append(parents, t.key.Literal)
	}
	fmt.Fprintf(w, pattern, strings.Join(parents, "."))
  formatComment(t.comment.post, "", w)
	fmt.Fprintln(w)
	return nil
}

func formatOptions(options []*Option, w io.Writer) error {
	length := longestKey(options)
	for _, o := range options {
    formatComment(o.comment.pre, "\n", w)
		if _, err := fmt.Fprintf(w, "%-*s = ", length, o.key.Literal); err != nil {
			return err
		}
		if err := formatValue(o.value, w); err != nil {
			return err
		}
    formatComment(o.comment.post, "", w)
		fmt.Fprintln(w)
	}
	return nil
}

func formatValue(n Node, w io.Writer) error {
	if n == nil {
		return nil
	}
	var err error
	switch n := n.(type) {
	case *Literal:
		pattern := "%s"
		if n.token.Type == TokString {
			pattern = "\"%s\""
		}
		fmt.Fprintf(w, pattern, n.token.Literal)
	case *Array:
		formatArray(n, w)
	case *Table:
		formatInline(n, w)
	default:
		err = fmt.Errorf("unexpected value type %T", n)
	}
	return err
}

func formatInline(t *Table, w io.Writer) {
	fmt.Fprint(w, "{")
	for i, o := range t.listOptions() {
		if i > 0 {
			fmt.Fprint(w, ", ")
		}
		fmt.Fprintf(w, "%s = ", o.key.Literal)
		formatValue(o.value, w)
	}
	fmt.Fprint(w, "}")
}

func formatArray(a *Array, w io.Writer) {
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
	fmt.Fprint(w, "[")
	for i, n := range a.nodes {
		if multi {
			fmt.Fprint(w, "\n\t")
		} else if !multi && i > 0 {
      fmt.Fprint(w, " ")
    }
		com := retr(n)
    formatComment(com.pre, "\n\t", w)
		formatValue(n, w)
		if i < len(a.nodes)-1 || multi {
			fmt.Fprint(w, ",")
		}
    formatComment(com.post, "", w)
	}
	if multi {
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, "]")
}

func formatComment(comment, eol string, w io.Writer) {
  if comment == "" {
    return
  }
  s := bufio.NewScanner(strings.NewReader(comment))
  for s.Scan() {
    fmt.Fprintf(w, " # %s", s.Text())
    if eol != "" {
      fmt.Fprint(w, eol)
    }
  }
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
