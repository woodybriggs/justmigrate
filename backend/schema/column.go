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

func (typ *Type) Eq(otherAny any) bool {
	other, ok := otherAny.(*Type)
	if !ok {
		return false
	}

	return typ.TypeName.Eq(other.TypeName)
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
	column ColumnLike,
	constraints *ColumnConstraints,
	astConstraints []ast.ColumnConstraint,
) error {

	errs := []error{}
	localConstraints := ColumnConstraints{}
	{
		constraints := &localConstraints
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
				if err := UniqueFromColumnConstraintAst(column.GetName(), constraints, constraint); err != nil {
					errs = append(errs, err)
				}
			case *ast.ColumnConstraint_PrimaryKey:
				if err := PrimaryKeyFromColumnConstraintAst(column, constraints, constraint); err != nil {
					errs = append(errs, err)
				}
			case *ast.ColumnConstraint_ForeignKey:
				if err := ForeignKeyFromColumnConstraintAst(column.GetName(), constraints, constraint); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	joinedErrs := errors.Join(errs...)
	if joinedErrs == nil {
		*constraints = localConstraints
	}

	return joinedErrs
}

type ColumnLike interface {
	column()
	GetName() *ast.Identifier
	SetName(*ast.Identifier)
	GetConstraints() ColumnConstraints
	AddConstraint(constraint ast.ColumnConstraint) error
	DropConstraint(constraint ast.ColumnConstraint) error
	GetType() *Type
	SetType(*ast.TypeName) error
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

func (c *GeneratedColumn) SetName(newName *ast.Identifier) {
	c.ColumnDefinition.ColumnName = *newName
	c.Name = newName.String()
}

func (c *GeneratedColumn) GetConstraints() ColumnConstraints {
	return c.ColumnConstraints
}

func (c *GeneratedColumn) AddConstraint(constraint ast.ColumnConstraint) error {
	panic("TODO(woody): not implemented")
}

func (c *GeneratedColumn) DropConstraint(constraint ast.ColumnConstraint) error {
	panic("TODO(woody): not implemented")
}

func (c *GeneratedColumn) GetType() *Type {
	return c.Type
}

func (c *GeneratedColumn) SetType(newTyp *ast.TypeName) error {
	typ, err := TypeFromAst(newTyp)
	if err != nil {
		return err
	}
	c.Type = typ
	return nil
}

func (c *GeneratedColumn) Eq(otherAny any) bool {
	other, ok := otherAny.(*GeneratedColumn)
	if !ok {
		return false
	}

	return c.ColumnDefinition.Eq(other.ColumnDefinition)
}

func GeneratedColumnFromAst(column *GeneratedColumn, colDef *ast.ColumnDefinition) error {

	if column == nil {
		panic("GeneratedColumnFromAst: column must not be nil")
	}

	errs := make([]error, 0, 2)
	localCol := GeneratedColumn{
		ColumnDefinition: colDef,
		Name:             colDef.ColumnName.String(),
	}

	if typ, err := TypeFromAst(colDef.TypeName); err != nil {
		errs = append(errs, err)
	} else {
		localCol.Type = typ
	}

	if err := GeneratedColumnConstraintsFromAst(&colDef.ColumnName, &localCol.ColumnConstraints, colDef.ColumnConstraints); err != nil {
		errs = append(errs, err)
	}

	joinedErrs := errors.Join(errs[0], errs[1])
	if joinedErrs == nil {
		*column = localCol
	}

	return joinedErrs
}

func GeneratedColumnConstraintsFromAst(colName *ast.Identifier, constraints *ColumnConstraints, astConstraints []ast.ColumnConstraint) error {

	errs := []error{}
	localConstraints := ColumnConstraints{}

	{
		constraints := &localConstraints
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
				if err := UniqueFromColumnConstraintAst(colName, constraints, constraint); err != nil {
					errs = append(errs, err)
				}
			case *ast.ColumnConstraint_ForeignKey:
				if err := ForeignKeyFromColumnConstraintAst(colName, constraints, constraint); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	joinedErrs := errors.Join(errs...)
	if joinedErrs == nil {
		*constraints = localConstraints
	}

	return joinedErrs
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

func (c *Column) SetName(newName *ast.Identifier) {
	c.ColumnDefinition.ColumnName = *newName
	c.Name = newName.String()
}

func (c *Column) GetConstraints() ColumnConstraints {
	return c.ColumnConstraints
}

func (c *Column) AddConstraint(constraint ast.ColumnConstraint) error {
	panic("TODO(woody): not implemented")
}

func (c *Column) DropConstraint(constraint ast.ColumnConstraint) error {
	panic("TODO(woody): not implemented")
}

func (c *Column) GetType() *Type {
	return c.Type
}

func (c *Column) SetType(newTyp *ast.TypeName) error {
	typ, err := TypeFromAst(newTyp)
	if err != nil {
		return err
	}
	c.Type = typ
	return nil
}

func (c *Column) Eq(otherAny any) bool {
	other, ok := otherAny.(*Column)
	if !ok {
		return false
	}

	return c.ColumnDefinition.Eq(other.ColumnDefinition)
}

func ColumnFromAst(column *Column, colDef *ast.ColumnDefinition) error {

	localCol := Column{
		ColumnDefinition: colDef,
		Name:             colDef.ColumnName.String(),
	}

	errs := make([]error, 0, 2)
	if typ, err := TypeFromAst(colDef.TypeName); err != nil {
		errs = append(errs, err)
	} else {
		localCol.Type = typ
	}

	if err := ColumnConstraintsFromAst(&localCol, &localCol.ColumnConstraints, colDef.ColumnConstraints); err != nil {
		errs = append(errs, err)
	}

	joinedErrs := errors.Join(errs...)
	if joinedErrs == nil {
		*column = localCol
	}

	return joinedErrs
}
