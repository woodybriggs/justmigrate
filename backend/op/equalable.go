package op

func (target TargetSchema) Eq(otherAny any) bool {
	other, ok := otherAny.(TargetSchema)
	if !ok {
		return false
	}

	schemaEq := target.Schema.Eq(other.Schema)
	return schemaEq
}

func (target TargetTable) Eq(otherAny any) bool {
	other, ok := otherAny.(TargetTable)
	if !ok {
		return false
	}

	schemaEq := target.Schema.Eq(other.Schema)
	tableEq := target.Table.Eq(other.Table)
	return schemaEq && tableEq
}

func (target TargetColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(TargetColumn)
	if !ok {
		return false
	}

	schemaEq := target.Schema.Eq(other.Schema)
	tableEq := target.Table.Eq(other.Table)
	colEq := target.Column.Eq(other.Column)
	return schemaEq && tableEq && colEq
}

func (target TargetColumnConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(TargetColumnConstraint)
	if !ok {
		return false
	}

	schemaEq := target.Schema.Eq(other.Schema)
	tableEq := target.Table.Eq(other.Table)
	colEq := target.Column.Eq(other.Column)
	constraintEq := target.Constraint.Eq(other.Constraint)
	return schemaEq && tableEq && colEq && constraintEq
}

func (target TargetTableConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(TargetTableConstraint)
	if !ok {
		return false
	}

	schemaEq := target.Schema.Eq(other.Schema)
	tableEq := target.Table.Eq(other.Table)
	constraintEq := target.Constraint.Eq(other.Constraint)
	return schemaEq && tableEq && constraintEq
}

func (op *DropTable) Eq(otherAny any) bool {
	other, ok := otherAny.(*DropTable)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return true
}

func (op *AddTable) Eq(otherAny any) bool {
	other, ok := otherAny.(*AddTable)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.Table.Eq(other.Table)
}

func (op *RenameTable) Eq(otherAny any) bool {
	other, ok := otherAny.(*RenameTable)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.NewName.Eq(other.NewName)
}

func (op *DropColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(*DropColumn)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return true
}

func (op *AddColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(*AddColumn)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.ColumnDefinition.Eq(other.ColumnDefinition)
}

func (op *RenameColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(*RenameColumn)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.NewName.Eq(other.NewName)
}

func (op *DropColumnConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(*DropColumnConstraint)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return true
}

func (op *AddColumnConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(*AddColumnConstraint)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.Constraint.Eq(other.Constraint)
}

func (op *DropTableConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(*DropTableConstraint)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return true
}

func (op *AddTableConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(*AddTableConstraint)
	if !ok {
		return false
	}

	if !op.Target.Eq(other.Target) {
		return false
	}

	return op.Constraint.Eq(other.Constraint)
}
