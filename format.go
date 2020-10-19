package toml

import (
	"fmt"
	"io"
	"strings"
)

func Format(r io.Reader, w io.Writer) error {
	doc, err := Parse(r)
	if err != nil {
		return err
	}
	t, ok := doc.(*Table)
	if !ok {
		return fmt.Errorf("document not parsed properly")
	}
	return formatTable(t, nil, w)
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
	if t.comment.pre != "" {
		for _, str := range strings.Split(t.comment.pre, "\n") {
			fmt.Fprintf(w, "# %s\n", str)
		}
	}
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
	if t.comment.post != "" {
		fmt.Fprintf(w, " # %s", t.comment.post)
	}
	fmt.Fprintln(w)
	return nil
}

func formatOptions(options []*Option, w io.Writer) error {
	length := longestKey(options)
	for _, o := range options {
		if o.comment.pre != "" {
			for _, str := range strings.Split(o.comment.pre, "\n") {
				fmt.Fprintf(w, "# %s\n", str)
			}
		}
		if _, err := fmt.Fprintf(w, "%-*s = ", length, o.key.Literal); err != nil {
			return err
		}
		if err := formatValue(o.value, w); err != nil {
			return err
		}
		if o.comment.post != "" {
			fmt.Fprintf(w, " # %s", o.comment.post)
		}
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
	fmt.Fprint(w, "[")
	for _, n := range a.nodes {
		formatValue(n, w)
		fmt.Fprint(w, ", ")
	}
	fmt.Fprint(w, "]")
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
