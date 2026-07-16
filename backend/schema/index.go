package schema

import "slices"

type Index struct {
	Name    string
	Table   *Table
	Columns []ColumnLike
}

func (index *Index) Eq(otherAny any) bool {
	other, ok := otherAny.(*Index)
	if !ok {
		return false
	}

	nameEq := index.Name == other.Name
	tableEq := index.Table.Eq(other.Table)
	colsEq := slices.EqualFunc(index.Columns, other.Columns, func(a, b ColumnLike) bool {
		return a.Eq(b)
	})

	return nameEq && tableEq && colsEq
}
