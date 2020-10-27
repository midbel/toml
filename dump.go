package toml

import (
	"fmt"
	"sort"
	"strings"
)

// Dump the given Node to stdout.
func Dump(n Node) {
	dumpNode(n, 0)
}

func dumpNode(n Node, level int) {
	space := strings.Repeat(" ", level*2)
	switch x := n.(type) {
	case *Option:
		value := dumpLiteral(x.value, level+2)
		fmt.Printf("%soption(pos: %s, key: %s, value: %s),", space, x.Pos(), x.key.Literal, value)
		fmt.Println()
	case *Table:
		if x.kind == tableArray {
			fmt.Printf("%sarray{", space)
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s},", space)
			fmt.Println()
		} else {
			label := x.key.Literal
			if label == "" {
				label = "default"
			}
			fmt.Printf("%stable[label=%s, kind=%s, pos= %s]{", space, label, x.kind, x.Pos())
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s},", space)
			fmt.Println()
		}
	}
}

func dumpLiteral(n Node, level int) string {
	switch x := n.(type) {
	case *Literal:
		return x.token.String()
	case *Array:
		var b strings.Builder
		b.WriteString("array")
		b.WriteRune(lsquare)
		for i, n := range x.nodes {
			if i > 0 {
				b.WriteRune(comma)
				b.WriteRune(space)
			}
			b.WriteString(dumpLiteral(n, level))
		}
		b.WriteRune(rsquare)
		return b.String()
	case *Table:
		var b strings.Builder
		b.WriteString("inline")
		b.WriteRune(lcurly)
		for _, n := range x.nodes {
			o, ok := n.(*Option)
			if !ok {
				b.WriteString("???")
			} else {
				b.WriteString(o.key.Literal)
				b.WriteRune(equal)
				b.WriteString(dumpLiteral(o.value, level))
			}
			b.WriteRune(comma)
			b.WriteRune(space)
		}
		b.WriteRune(rcurly)
		return b.String()
	default:
		return "???"
	}
}

func sortNodes(nodes []Node) []Node {
	ns := make([]Node, len(nodes))
	copy(ns, nodes)

	sort.Slice(ns, func(i, j int) bool {
		pi, pj := ns[i].Pos(), ns[j].Pos()
		return pi.Line < pj.Line
	})
	return ns
}
