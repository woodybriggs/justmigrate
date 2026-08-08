package op

import "fmt"

func (t TargetSchema) String() string {
	return t.Schema.Name
}

func (t TargetTable) String() string {
	return t.Schema.Name + "." + t.Table.Name
}

func (t TargetTableConstraint) String() string {
	return t.Schema.Name + "." + t.Table.Name + ".(" + fmt.Sprintf("%T", t.Constraint) + ")"
}

func (t TargetColumn) String() string {
	return t.Schema.Name + "." + t.Table.Name + "." + t.Column.GetName().Text
}

func (t TargetColumnConstraint) String() string {
	return t.Schema.Name + "." + t.Table.Name + "." + t.Column.GetName().Text + ".(" + fmt.Sprintf("%T", t.Constraint) + ")"
}

func (t TargetIndex) String() string {
	return t.Schema.Name + "." + t.Table.Name + "." + t.Index.Name
}

func (o MigrateData) String() string {
	return fmt.Sprintf("%T", o)
}
