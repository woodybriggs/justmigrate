package diff

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"woodybriggs/justmigrate/core/ast"
)

type Diff struct{}

type Edit interface {
	edit()
	String() string
}

type EditRemoveTable struct {
	*ast.CreateTable
}

func (edit *EditRemoveTable) edit() {}
func (edit *EditRemoveTable) String() string {
	return fmt.Sprintf("remove table: \"%s\"", edit.TableIdentifier.ObjectName.Text)
}

type EditAddTable struct {
	*ast.CreateTable
}

func (edit *EditAddTable) edit() {}
func (edit *EditAddTable) String() string {
	return fmt.Sprintf("add table: \"%s\"", edit.TableIdentifier.ObjectName.Text)
}

type EditModifyTable struct {
	Target *ast.CreateTable
	Edits  []Edit
}

func (edit *EditModifyTable) edit() {}
func (edit *EditModifyTable) String() string {
	builder := strings.Builder{}

	fmt.Fprintf(&builder, "modify table: \"%s\"\n", edit.Target.TableIdentifier.ObjectName.Text)
	for _, edit := range edit.Edits {
		builder.WriteString(edit.String())
	}

	return builder.String()
}

type EditRemoveColumn struct {
	ast.ColumnDefinition
}

func (edit *EditRemoveColumn) edit() {}
func (edit *EditRemoveColumn) String() string {
	return fmt.Sprintf("remove column: \"%s\"\n", edit.ColumnName.Text)
}

type EditAddColumn struct {
	ast.ColumnDefinition
}

func (edit *EditAddColumn) edit() {}
func (edit *EditAddColumn) String() string {
	return fmt.Sprintf("add column: \"%s\"\n", edit.ColumnName.Text)
}

type EditModifyColumn struct {
	Target *ast.ColumnDefinition
	Edits  []Edit
}

func (edit *EditModifyColumn) edit() {}
func (edit *EditModifyColumn) String() string {
	builder := strings.Builder{}

	fmt.Fprintf(&builder, "modify column: \"%s\"\n", edit.Target.ColumnName.Text)
	for _, edit := range edit.Edits {
		builder.WriteString(edit.String())
	}

	return builder.String()
}

type EditRemoveTableConstraint struct {
	ast.TableConstraint
}

func (edit *EditRemoveTableConstraint) edit() {}
func (edit *EditRemoveTableConstraint) String() string {
	return fmt.Sprintf("remove table constraint: \"%T\"\n", edit.TableConstraint)
}

type EditAddTableConstraint struct {
	ast.TableConstraint
}

func (edit *EditAddTableConstraint) edit() {}
func (edit *EditAddTableConstraint) String() string {
	return fmt.Sprintf("add table constraint: \"%T\"\n", edit.TableConstraint)
}

type EditChangeColumnType struct {
	From ast.TypeName
	To   ast.TypeName
}

func (edit *EditChangeColumnType) edit() {}
func (edit *EditChangeColumnType) String() string {
	return fmt.Sprintf("change column type: from %s to %s\n", edit.From.TypeName.Text, edit.To.TypeName.Text)
}

type EditRemoveColumnConstraint struct {
	ast.ColumnConstraint
}

func (edit *EditRemoveColumnConstraint) edit() {}
func (edit *EditRemoveColumnConstraint) String() string {
	return fmt.Sprintf("remove column constraint: \"%T\"\n", edit.ColumnConstraint)
}

type EditAddColumnConstraint struct {
	ast.ColumnConstraint
}

func (edit *EditAddColumnConstraint) edit() {}
func (edit *EditAddColumnConstraint) String() string {
	return fmt.Sprintf("add column constraint: \"%T\"\n", edit.ColumnConstraint)
}

type EditModifyColumnConstraint struct {
	Target ast.ColumnConstraint
	Edits  []Edit
}

func (edit *EditModifyColumnConstraint) String() string {
	builder := strings.Builder{}

	fmt.Fprintf(&builder, "modify column constraint: \"%T\"\n", edit.Target)
	for _, edit := range edit.Edits {
		builder.WriteString(edit.String())
	}

	return builder.String()
}

func (edit *EditModifyColumnConstraint) edit() {}

type EditModifyTableConstraint struct {
	Target ast.TableConstraint
	Edits  []Edit
}

func (edit *EditModifyTableConstraint) edit() {}
func (edit *EditModifyTableConstraint) String() string {
	builder := strings.Builder{}

	builder.WriteString(fmt.Sprintf("modify table constraint: \"%T\"\n", edit.Target))
	for _, edit := range edit.Edits {
		builder.WriteString(edit.String())
	}

	return builder.String()
}

type pair[T any] struct {
	A T
	B T
}

type pairs[T any] []pair[T]

func intersection[T any](a, b []T, equal func(x, y T) bool) (result pairs[T]) {
	seen := make(map[int]struct{})

	result = make([]pair[T], 0, min(len(a), len(b)))

	for i, x := range a {
		if _, included := seen[i]; included {
			continue
		}

		for _, y := range b {
			if equal(x, y) {
				result = append(result, pair[T]{
					A: x,
					B: y,
				})
				seen[i] = struct{}{}
				break
			}
		}
	}

	return result
}

func symmetricDifference[T any](a, b []T, predicate func(a, b T) bool) (left, right []T) {
	matchedA := make([]bool, len(a))
	matchedB := make([]bool, len(b))

	for i, x := range a {
		if matchedA[i] {
			continue
		}
		for j, y := range b {
			if matchedB[j] {
				continue
			}
			if predicate(x, y) {
				matchedA[i] = true
				matchedB[j] = true
				break
			}
		}
	}

	// Collect unmatched
	for i, x := range a {
		if !matchedA[i] {
			left = append(left, x)
		}
	}

	for j, y := range b {
		if !matchedB[j] {
			right = append(right, y)
		}
	}

	return left, right
}

func mapOver[T, U any](seq iter.Seq[T], mapfn func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		seq(func(v T) bool {
			return yield(mapfn(v))
		})
	}
}

func filter[T any](seq iter.Seq[T], predicate func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		seq(func(v T) bool {
			if !predicate(v) {
				return true // Skip this item, but tell upstream to keep going
			}
			return yield(v) // Emit item; if downstream stops, stop upstream
		})
	}
}

func filterThenMap[T any, U any](seq iter.Seq[T], filterMap func(T) (U, bool)) iter.Seq[U] {
	return func(yield func(U) bool) {
		seq(func(v T) bool {
			u, ok := filterMap(v)
			if !ok {
				return true
			}
			return yield(u)
		})
	}
}

var (
	ErrArgumentMismatch error = errors.New("arguments a and b do not match")
)

func filterForCreateTable(value ast.Statement) (*ast.CreateTable, bool) {
	result, ok := value.(*ast.CreateTable)
	return result, ok
}

func isSameCreateTable(a, b *ast.CreateTable) bool {
	return a.TableIdentifier.Eq(b.TableIdentifier)
}

func (diff *Diff) DiffSchema(a, b []ast.Statement) ([]Edit, error) {
	edits := []Edit{}

	// Compare all create table statements
	{
		a := slices.Collect(filterThenMap(slices.Values(a), filterForCreateTable))
		b := slices.Collect(filterThenMap(slices.Values(b), filterForCreateTable))

		removedTables, addedTables := symmetricDifference(a, b, isSameCreateTable)
		maybeModifiedTables := intersection(a, b, isSameCreateTable)

		for _, removedTable := range removedTables {
			edits = append(edits, &EditRemoveTable{removedTable})
		}

		for _, addedTable := range addedTables {
			edits = append(edits, &EditAddTable{addedTable})
		}

		for _, pair := range maybeModifiedTables {
			edit := diff.DiffCreateTable(pair.A, pair.B)
			if edit != nil {
				edits = append(edits, edit)
			}
		}
	}

	return edits, nil
}

func isSameColumnDefinition(a, b ast.ColumnDefinition) bool {
	return a.ColumnName.Eq(&b.ColumnName)
}

func isSameTableConstraint(a, b ast.TableConstraint) bool {
	switch a := a.(type) {
	case *ast.TableConstraint_PrimaryKey:
		_, ok := b.(*ast.TableConstraint_PrimaryKey)
		if !ok {
			return false
		}
		return true
	case *ast.TableConstraint_ForeignKey:
		b, ok := b.(*ast.TableConstraint_ForeignKey)
		if !ok {
			return false
		}
		return a.Eq(b)
	default:
		return false
	}
}

func (diff *Diff) DiffCreateTable(a, b *ast.CreateTable) Edit {
	edits := []Edit{}

	// Compare column definitions
	{

		a := a.TableDefinition.ColumnDefinitions
		b := b.TableDefinition.ColumnDefinitions

		removedColumns, addedColumns := symmetricDifference(a, b, isSameColumnDefinition)
		maybeModifiedColumns := intersection(a, b, isSameColumnDefinition)

		for _, removedColumn := range removedColumns {
			edits = append(edits, &EditRemoveColumn{removedColumn})
		}

		for _, addedColumn := range addedColumns {
			edits = append(edits, &EditAddColumn{addedColumn})
		}

		for _, pair := range maybeModifiedColumns {
			edit := diff.DiffColumnDefinition(pair.A, pair.B)
			if edit != nil {
				edits = append(edits, edit)
			}
		}
	}

	// Compare table constraints
	{
		a := a.TableDefinition.TableConstraints
		b := b.TableDefinition.TableConstraints

		removedConstraints, addedConstraints := symmetricDifference(a, b, isSameTableConstraint)
		maybeModifiedConstraints := intersection(a, b, isSameTableConstraint)

		for _, removedConstraints := range removedConstraints {
			edits = append(edits, &EditRemoveTableConstraint{removedConstraints})
		}

		for _, addedConstratint := range addedConstraints {
			edits = append(edits, &EditAddTableConstraint{addedConstratint})
		}

		for _, pair := range maybeModifiedConstraints {
			edit := diff.DiffTableConstraint(pair.A, pair.B)
			if edit != nil {
				edits = append(edits, edit)
			}
		}
	}

	if len(edits) > 0 {
		return &EditModifyTable{
			Target: a,
			Edits:  edits,
		}
	}

	return nil
}

func (diff *Diff) DiffColumnDefinition(a, b ast.ColumnDefinition) Edit {
	edits := []Edit{}

	if len(edits) == 0 {
		return nil
	}

	return &EditModifyColumn{
		Target: &a,
		Edits:  edits,
	}
}

func (diff *Diff) DiffTableConstraint(a, b ast.TableConstraint) Edit {
	edits := []Edit{}

	if len(edits) == 0 {
		return nil
	}

	return &EditModifyTableConstraint{
		Target: a,
		Edits:  edits,
	}
}
