package toml

import (
	"fmt"
	"sort"
	"strings"
)

func Dump(n Node) {
	dumpNode(n, 0)
}

func dumpNode(n Node, level int) {
	switch x := n.(type) {
	case *Option:
		value := dumpLiteral(x.value)
		fmt.Printf("%soption(pos: %s, key: %s, value: %s),", strings.Repeat(" ", level*2), x.Pos(), x.key.Literal, value)
		fmt.Println()
	case *Table:
		if x.kind == tableArray {
			fmt.Printf("%sarray{", strings.Repeat(" ", level*2))
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s},", strings.Repeat(" ", level*2))
			fmt.Println()
		} else {
			label := x.key.Literal
			if label == "" {
				label = "default"
			}
			fmt.Printf("%stable[label=%s, kind=%s, pos= %s]{", strings.Repeat(" ", level*2), label, x.kind, x.Pos())
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s},", strings.Repeat(" ", level*2))
			fmt.Println()
		}
	}
}

func dumpLiteral(n Node) string {
	switch x := n.(type) {
	case *Literal:
		str := tokenString(x.token.Type)
		return fmt.Sprintf("%s(%s)", str, x.token.Literal)
	case *Array:
		var b strings.Builder
		b.WriteString("array")
		b.WriteRune(lsquare)
		for _, n := range x.nodes {
			b.WriteString(dumpLiteral(n))
			b.WriteRune(comma)
			b.WriteRune(space)
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
				b.WriteString(dumpLiteral(o.value))
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
