package schema

import (
	"errors"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
	"woodybriggs/justmigrate/frontend/token"
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
	column *Column,
	astConstraints []ast.ColumnConstraint,
) error {

	errs := []error{}
	{
		for i := range astConstraints {
			err := column.AddConstraint(astConstraints[i])
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
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
	localConstraints := ColumnConstraints{}

	switch constraint := constraint.(type) {
	// generated columns are not allowed to have default constraints
	case *ast.ColumnConstraint_Default:
		return report.NewReport("invalid column definition").
			WithLocation(constraint.DefaultKeyword.FileLoc).
			WithMessage("generated column can not have a default value").
			WithLabels(
				report.LabelFromExpr(constraint.Default, "this default value"),
			)
	// generated columns are not allowed to be part of primary keys
	case *ast.ColumnConstraint_PrimaryKey:
		return report.NewReport("invalid column definition").
			WithLocation(constraint.PrimaryKeyword.FileLoc).
			WithMessage("generated column can not be part of a primary key constraint").
			WithLabels(
				report.LabelFromKeyword(constraint.PrimaryKeyword, "this primary key is not allowed"),
			)
	case *ast.ColumnConstraint_Check:
		if err := CheckFromColumnConstraintAst(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_Collate:
		if err := CollateConstraintFromColumnConstraintAst(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_NotNull:
		if err := NotNullFromColumnConstraint(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_Unique:
		if err := UniqueFromColumnConstraintAst(c.GetName(), &localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_ForeignKey:
		if err := ForeignKeyFromColumnConstraintAst(c.GetName(), &localConstraints, constraint); err != nil {
			return err
		}
	}

	c.ColumnConstraints = localConstraints

	return nil
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

func (c *GeneratedColumn) Clone() *GeneratedColumn {
	clone := *c
	return &clone
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

	if err := GeneratedColumnConstraintsFromAst(&localCol, colDef.ColumnConstraints); err != nil {
		errs = append(errs, err)
	}

	joinedErrs := errors.Join(errs[0], errs[1])
	if joinedErrs == nil {
		*column = localCol
	}

	return joinedErrs
}

func GeneratedColumnConstraintsFromAst(
	column *GeneratedColumn,
	astConstraints []ast.ColumnConstraint,
) error {
	errs := []error{}
	for i := range astConstraints {
		err := column.AddConstraint(astConstraints[i])
		if err != nil {
			errs = append(errs, err)
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

func (c *Column) SetName(newName *ast.Identifier) {
	c.ColumnDefinition.ColumnName = *newName
	c.Name = newName.String()
}

func (c *Column) GetConstraints() ColumnConstraints {
	return c.ColumnConstraints
}

func (c *Column) AddConstraint(constraint ast.ColumnConstraint) error {

	localConstraints := ColumnConstraints{}

	switch constraint := constraint.(type) {
	case *ast.ColumnConstraint_Generated:
		panic("we should not be here, we should already know if a column is generated and use GeneratedColumnFromColumnConstraintAst")
	case *ast.ColumnConstraint_Check:
		if err := CheckFromColumnConstraintAst(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_Collate:
		if err := CollateConstraintFromColumnConstraintAst(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_NotNull:
		if err := NotNullFromColumnConstraint(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_Default:
		if err := DefaultFromColumnConstraint(&localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_Unique:
		if err := UniqueFromColumnConstraintAst(c.GetName(), &localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_PrimaryKey:
		if err := PrimaryKeyFromColumnConstraintAst(c, &localConstraints, constraint); err != nil {
			return err
		}
	case *ast.ColumnConstraint_ForeignKey:
		if err := ForeignKeyFromColumnConstraintAst(c.GetName(), &localConstraints, constraint); err != nil {
			return err
		}
	}

	c.ColumnConstraints = localConstraints

	return nil
}

func (c *Column) DropConstraint(constraint ast.ColumnConstraint) error {
	switch constraint.(type) {
	case *ast.ColumnConstraint_Generated:
		panic("we should not be here, we should already know if a column is generated and use GeneratedColumnFromColumnConstraintAst")
	case *ast.ColumnConstraint_Check:
		c.ColumnConstraints.Check = nil
	case *ast.ColumnConstraint_Collate:
		c.ColumnConstraints.Collate = nil
	case *ast.ColumnConstraint_NotNull:
		c.ColumnConstraints.NotNull = nil
	case *ast.ColumnConstraint_Default:
		c.ColumnConstraints.Default = nil
	case *ast.ColumnConstraint_Unique:
		c.ColumnConstraints.Unique = nil
	case *ast.ColumnConstraint_PrimaryKey:
		c.ColumnConstraints.PK = nil
	case *ast.ColumnConstraint_ForeignKey:
		c.ColumnConstraints.FK = nil
	}
	return nil
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

func (c *Column) Clone() *Column {
	clone := *c
	return &clone
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

	if err := ColumnConstraintsFromAst(&localCol, colDef.ColumnConstraints); err != nil {
		errs = append(errs, err)
	}

	joinedErrs := errors.Join(errs...)
	if joinedErrs == nil {
		*column = localCol
	}

	return joinedErrs
}

func ColumnDefinitionAstFromColumn(column ColumnLike) *ast.ColumnDefinition {

	columnConstraints := ColumnConstraintsAstFromColumnConstraints(column.GetConstraints())

	return ast.MakeColumnDefinition(
		*column.GetName(),
		column.GetType().TypeName,
		columnConstraints,
	)
}

func ColumnConstraintsAstFromColumnConstraints(constraints ColumnConstraints) []ast.ColumnConstraint {
	result := []ast.ColumnConstraint{}

	var constraintName *ast.ConstraintName = nil

	if constraints.NotNull != nil {

		if constraints.NotNull.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              constraintName.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintNotNull(
			constraintName,
			ast.Keyword(token.Keyword("NOT")),
			ast.Keyword(token.Keyword("NULL")),
			ConflictClauseAstFromConflictAction(constraints.NotNull.ConflictClause),
		))

	}

	if constraints.Default != nil {
		if constraints.Default.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.Default.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintDefault(
			constraintName,
			ast.Keyword(token.Keyword("DEFAULT")),
			constraints.Default.Expr,
		))
	}

	if constraints.Unique != nil {
		if constraints.Unique.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.Unique.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintUnique(
			constraintName,
			ast.Keyword(token.Keyword("UNIQUE")),
			ConflictClauseAstFromConflictAction(constraints.Unique.ConflictAction),
		))
	}

	if constraints.Collate != nil {
		if constraints.Collate.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.Collate.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintCollate(
			constraintName,
			ast.MakeCollation(
				ast.Keyword(token.Keyword("COLLATE")),
				*constraints.Collate.Collate,
			),
		))
	}

	if constraints.PK != nil {
		if constraints.PK.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.PK.Name,
			}
		}

		// TODO(woody): we don't store autoincrement for sqlite anywhere
		var autoIncrement *ast.Keyword = nil

		result = append(result, ast.MakeColumnConstraintPrimaryKey(
			constraintName,
			ast.Keyword(token.Keyword("PRIMARY")),
			ast.Keyword(token.Keyword("KEY")),
			OrderKeywordAstFromOrder(constraints.PK.Order),
			ConflictClauseAstFromConflictAction(constraints.PK.ConflictClause),
			autoIncrement,
		))
	}

	if constraints.FK != nil {
		if constraints.FK.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.FK.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintForeignKey(
			constraintName,
			*ForeignKeyClauseAstFromForeignKey(constraints.FK),
		))
	}

	if constraints.Check != nil {
		if constraints.Check.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *constraints.Check.Name,
			}
		}

		result = append(result, ast.MakeColumnConstraintCheck(
			constraintName,
			ast.Keyword(token.Keyword("CHECK")),
			constraints.Check.Expr,
		))
	}

	return result
}

func ConflictClauseAstFromConflictAction(action ConflictAction) *ast.ConflictClause {

	var actionKeyword *ast.Keyword = nil

	switch action {
	case ConflictActionAbort:
		actionKeyword = ast.MakeKeyword(token.Keyword("abort"))
	case ConflictActionFail:
		actionKeyword = ast.MakeKeyword(token.Keyword("fail"))
	case ConflictActionReplace:
		actionKeyword = ast.MakeKeyword(token.Keyword("replace"))
	case ConflictActionRollback:
		actionKeyword = ast.MakeKeyword(token.Keyword("rollback"))
	case ConflictActionIgnore:
		actionKeyword = ast.MakeKeyword(token.Keyword("ignore"))
	default:
		return nil
	}

	if actionKeyword == nil {
		return nil
	}

	return ast.MakeConflictClause(
		ast.Keyword(token.Keyword("ON")),
		ast.Keyword(token.Keyword("CONFLICT")),
		*actionKeyword,
	)
}

func OrderKeywordAstFromOrder(order Order) *ast.Keyword {
	switch order {
	case OrderAscending:
		return ast.MakeKeyword(token.Keyword("ASC"))
	case OrderDescending:
		return ast.MakeKeyword(token.Keyword("DESC"))
	default:
		return nil
	}
}
