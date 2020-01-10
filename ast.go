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
	comment Token
	key     Token
	value   Node
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

type Table struct {
	key   Token
	nodes []Node
}

func (t *Table) String() string {
	return t.key.Literal
}

func (t *Table) Pos() Position {
	return t.key.Pos
}
