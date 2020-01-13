package toml

type Node interface {
	Pos() Position
}

type Literal struct {
	token Token
}

func (l *Literal) Pos() Position {
	return l.token.Pos
}

type Option struct {
	key   Token
	value Node
}

func (o *Option) Pos() Position {
	return o.key.Pos
}

type Array struct {
	pos   Position
	nodes []Node
}

func (a *Array) Pos() Position {
	return a.pos
}

type tableType int

const (
	typeAbstract tableType = iota
	typeRegular
	typeInline
	typeArray
	typeItem
)

type Table struct {
	pos  Position
	key  string
	kind tableType

	nodes []Node
}

func createTable(key Token, kind tableType) *Table {
	return createTableWithValues(key, kind, nil)
}

func createTableWithValues(key Token, kind tableType, ns []Node) *Table {
	return &Table{
		pos:   key.Pos,
		key:   key.Literal,
		kind:  kind,
		nodes: ns,
	}
}

func (t *Table) Pos() Position {
	return t.pos
}

func (t *Table) isDefined() bool {
	return t.kind != typeAbstract
}

func (t *Table) isFrozen() bool {
	return false
}
