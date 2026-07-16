package schema

import (
	"errors"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
)

type Type struct {
	TypeName *ast.TypeName

	Name string
	Args []ast.NumericLiteral
}

func TypeFromAst(typeName *ast.TypeName) (*Type, error) {
	if typeName == nil {
		return nil, nil
	}
	return &Type{
		TypeName: typeName,
		Name:     typeName.Name.Text,
		Args:     []ast.NumericLiteral{typeName.Arg0, typeName.Arg1},
	}, nil
}

type GeneratedColumnConstraints struct {
	Collate string
	NotNull *ast.ColumnConstraint_NotNull
	Checks  *Check
	Unique  *ast.ColumnConstraint_Unique
}

type ColumnConstraints struct {
	Collate *CollateConstraint
	NotNull *NotNull
	Default *Default
	Unique  *Unique
	Check   *Check
	FK      *ForeignKey
	PK      *PrimaryKey
}

func ColumnConstraintsFromAst(
	table *Table,
	column *Column,
	constraints *ColumnConstraints,
	astConstraints []ast.ColumnConstraint,
) error {

	errs := []error{}

	for i := range astConstraints {
		switch constraint := astConstraints[i].(type) {
		case *ast.ColumnConstraint_Generated:
			panic("we should not be here, we should already know if a column is generated and use GeneratedColumnFromColumnConstraintAst")
		case *ast.ColumnConstraint_Check:
			if err := CheckFromColumnConstraintAst(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_Collate:
			if err := CollateConstraintFromColumnConstraintAst(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_NotNull:
			if err := NotNullFromColumnConstraint(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_Default:
			if err := DefaultFromColumnConstraint(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_Unique:
			if err := UniqueFromColumnConstraintAst(column, constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_PrimaryKey:
			if err := PrimaryKeyFromColumnConstraintAst(column, constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_ForeignKey:
			if err := ForeignKeyFromColumnConstraintAst(column, constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

type ColumnLike interface {
	column()
	GetName() *ast.Identifier

	GetConstraints() ColumnConstraints
	Eq(any) bool
}

type GeneratedColumn struct {
	ColumnDefinition *ast.ColumnDefinition

	Name    string
	Type    *Type
	Storage Storage
	ColumnConstraints
}

func (genCol *GeneratedColumn) column() {}

func (c *GeneratedColumn) GetName() *ast.Identifier {
	return &c.ColumnDefinition.ColumnName
}

func (c *GeneratedColumn) GetConstraints() ColumnConstraints {
	return c.ColumnConstraints
}

func (c *GeneratedColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(*GeneratedColumn)
	if !ok {
		return false
	}

	return c.ColumnDefinition.Eq(other.ColumnDefinition)
}

func GeneratedColumnFromAst(table *Table, column *GeneratedColumn, colDef *ast.ColumnDefinition) (err error) {
	column.ColumnDefinition = colDef
	column.Name = colDef.ColumnName.String()
	column.Type, err = TypeFromAst(colDef.TypeName)

	err = GeneratedColumnConstraintsFromAst(table, column, &column.ColumnConstraints, colDef.ColumnConstraints)
	return err
}

func GeneratedColumnConstraintsFromAst(table *Table, column *GeneratedColumn, constraints *ColumnConstraints, astConstraints []ast.ColumnConstraint) error {

	errs := []error{}

	for i := range astConstraints {
		switch constraint := astConstraints[i].(type) {
		// generated columns are not allowed to have default constraints
		case *ast.ColumnConstraint_Default:
			err := report.NewReport("invalid column definition").
				WithLocation(constraint.DefaultKeyword.FileLoc).
				WithMessage("generated column can not have a default value").
				WithLabels(
					report.LabelFromExpr(constraint.Default, "this default value"),
				)
			errs = append(errs, err)
		// generated columns are not allowed to be part of primary keys
		case *ast.ColumnConstraint_PrimaryKey:
			err := report.NewReport("invalid column definition").
				WithLocation(constraint.PrimaryKeyword.FileLoc).
				WithMessage("generated column can not be part of a primary key constraint").
				WithLabels(
					report.LabelFromKeyword(constraint.PrimaryKeyword, "this primary key is not allowed"),
				)
			errs = append(errs, err)
		case *ast.ColumnConstraint_Check:
			if err := CheckFromColumnConstraintAst(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_Collate:
			if err := CollateConstraintFromColumnConstraintAst(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_NotNull:
			if err := NotNullFromColumnConstraint(constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_Unique:
			if err := UniqueFromColumnConstraintAst(column, constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		case *ast.ColumnConstraint_ForeignKey:
			if err := ForeignKeyFromColumnConstraintAst(column, constraints, constraint); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

type Column struct {
	ColumnDefinition *ast.ColumnDefinition

	Name string
	Type *Type
	PK   *PrimaryKey
	ColumnConstraints
}

func (col *Column) column() {}

func (c *Column) GetName() *ast.Identifier {
	return &c.ColumnDefinition.ColumnName
}

func (c *Column) GetConstraints() ColumnConstraints {
	return c.ColumnConstraints
}

func (c *Column) Eq(otherAny any) bool {
	other, ok := otherAny.(*Column)
	if !ok {
		return false
	}

	return c.ColumnDefinition.Eq(other.ColumnDefinition)
}

func ColumnFromAst(table *Table, column *Column, colDef *ast.ColumnDefinition) (err error) {

	column.ColumnDefinition = colDef
	column.Name = colDef.ColumnName.String()
	column.Type, err = TypeFromAst(colDef.TypeName)

	err = ColumnConstraintsFromAst(table, column, &column.ColumnConstraints, colDef.ColumnConstraints)

	return err
}
