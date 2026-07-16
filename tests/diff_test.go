package tests

import (
	"slices"
	"testing"

	"woodybriggs/justmigrate/backend/diff"
	"woodybriggs/justmigrate/backend/op"
	"woodybriggs/justmigrate/backend/schema"
	"woodybriggs/justmigrate/dialects/sqlite/parser"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/lexer"
	"woodybriggs/justmigrate/frontend/token"
)

var schemaSourceA string = ` 
CREATE TABLE users (
    username TEXT NOT NULL UNIQUE,
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name TEXT NOT NULL, 
    email TEXT,
    phone TEXT
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_amount REAL,
    shipping_address TEXT NOT NULL, 
    CONSTRAINT valid_total CHECK (total_amount >= 0)
);

CREATE TABLE legacy_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    [action] TEXT NOT NULL,
    timestamp TEXT DEFAULT CURRENT_TIMESTAMP
);
`

var schemaSourceB string = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    -- DROP COLUMN CONSTRAINT: Removed 'NOT NULL' from first_name
    first_name TEXT,
    -- NEW COLUMN CONSTRAINT: Added 'UNIQUE NOT NULL' to email
    email TEXT UNIQUE NOT NULL, 
    phone TEXT 
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_amount REAL,
	-- NEW COLUMN: Added 'status'
    status TEXT DEFAULT 'cart',
	-- NEW TABLE CONSTRAINT: Added a composite check constraint
    CONSTRAINT valid_checkout CHECK ((status = 'cart') OR (total_amount > 0)) 
    -- DROP COLUMN: Removed 'shipping_address' (assumed moved to a normalized addresses table)
	-- DROP TABLE CONSTRAINT: Removed 'CONSTRAINT valid_total CHECK (total_amount >= 0)'
);

-- DROP TABLE: 'legacy_audit_logs' has been completely removed from the schema

-- NEW TABLE: Added 'product_reviews'
CREATE TABLE product_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    review_text TEXT
);
`

// expected ops
// drop table legacy_audit_logs
// add table product_reviews
// drop not null column constraint from users.first_name
// add not null column constraint to user.email
// add unique column constraint to users.email
// drop column orders.shipping_address
// add column orders.status
// drop check table constraint valid_total from orders
// add check table constraint valid_checkout to orders

func loadSchema(filename string, src string) (*schema.Schema, error) {
	sourceCode := lexer.SourceCode{
		FileName: filename,
		Raw:      []rune(src),
	}

	l := lexer.NewLexer(sourceCode)
	p := parser.NewSqliteParser(l)
	ast := p.Statements()
	schema, err := schema.SchemaFromAst(ast)

	return schema, err
}

var NotKeyword *ast.Keyword = ast.MakeKeyword(
	token.Token{
		Text: "not",
		Kind: token.TokenKind_Keyword_NOT,
	},
)

var NullKeyword *ast.Keyword = ast.MakeKeyword(
	token.Token{
		Text: "null",
		Kind: token.TokenKind_Keyword_NULL,
	},
)

var UniqueKeyword *ast.Keyword = ast.MakeKeyword(
	token.Token{
		Text: "unique",
		Kind: token.TokenKind_Keyword_UNIQUE,
	},
)

var CheckKeyword *ast.Keyword = ast.MakeKeyword(
	token.Token{
		Text: "check",
		Kind: token.TokenKind_Keyword_CHECK,
	},
)

var DefaultKeyword *ast.Keyword = ast.MakeKeyword(
	token.Token{
		Text: "default",
		Kind: token.TokenKind_Keyword_DEFAULT,
	},
)

var statusColumnDefinition = ast.MakeColumnDefinition(
	*ast.MakeIdentifier(token.Token{Text: "status", Kind: token.TokenKind_Identifier}),
	ast.MakeTypeName(
		*ast.MakeIdentifier(token.Token{Text: "TEXT", Kind: token.TokenKind_Identifier}),
		nil,
		nil,
	),
	[]ast.ColumnConstraint{
		ast.MakeColumnConstraintDefault(
			nil,
			*DefaultKeyword,
			ast.MakeLiteralString(
				token.Token{Text: "cart", Kind: token.TokenKind_StringLiteral},
				"cart",
			),
		),
	},
)

func Test_DiffSchema(t *testing.T) {

	schemaA, err := loadSchema("a", schemaSourceA)
	if err != nil {
		t.Fatalf("failed to load schema a\n%v\n", err)
		t.FailNow()
	}

	schemaB, err := loadSchema("b", schemaSourceB)
	if err != nil {
		t.Fatalf("failed to load schema\n%v\n", err)
		t.FailNow()
	}

	differ := diff.Diff{}

	ops, err := differ.DiffSchema(schemaA, schemaB)
	if err != nil {
		t.Fatalf("failed to diff schema %s", err)
	}

	ordersTable := schemaA.Tables["orders"]
	column := schema.Column{}
	err = schema.ColumnFromAst(ordersTable, &column, statusColumnDefinition)
	if err != nil {
		t.Fatalf("%v", err.Error())
		t.FailNow()
	}

	expectedOps := []op.Op{
		&op.DropTable{
			Target: op.TargetTable{
				Schema: schemaA,
				Table:  schemaA.Tables["legacy_audit_logs"],
			},
		},
		&op.AddTable{
			Target: op.TargetSchema{
				Schema: schemaA,
			},
			Table: schemaB.Tables["product_reviews"],
		},
		&op.DropColumnConstraint{
			Target: op.TargetColumnConstraint{
				Schema: schemaA,
				Table:  schemaA.Tables["users"],
				Column: schemaA.Columns["users"]["first_name"],
				Constraint: ast.MakeColumnConstraintNotNull(
					nil,
					*NotKeyword,
					*NullKeyword,
					nil,
				),
			},
		},
		&op.AddColumnConstraint{
			Target: op.TargetColumn{
				Schema: schemaA,
				Table:  schemaA.Tables["users"],
				Column: schemaA.Columns["users"]["email"],
			},
			Constraint: ast.MakeColumnConstraintNotNull(
				nil,
				*NotKeyword,
				*NullKeyword,
				nil,
			),
		},
		&op.AddColumnConstraint{
			Target: op.TargetColumn{
				Schema: schemaA,
				Table:  schemaA.Tables["users"],
				Column: schemaA.Columns["users"]["email"],
			},
			Constraint: ast.MakeColumnConstraintUnique(
				nil,
				*UniqueKeyword,
				nil,
			),
		},
		&op.DropColumn{
			Target: op.TargetColumn{
				Schema: schemaA,
				Table:  schemaA.Tables["orders"],
				Column: schemaA.Columns["orders"]["shipping_address"],
			},
		},
		&op.AddColumn{
			Target: op.TargetTable{
				Schema: schemaA,
				Table:  schemaA.Tables["orders"],
			},
			ColumnDefinition: &column,
		},
		&op.DropTableConstraint{
			Target: op.TargetTableConstraint{
				Schema: schemaA,
				Table:  schemaA.Tables["orders"],
				Constraint: ast.MakeTableConstraintCheck(
					&ast.ConstraintName{
						Name: *ast.MakeIdentifier(token.Token{Text: "valid_total", Kind: token.TokenKind_Identifier}),
					},
					*CheckKeyword,
					token.Token{Kind: token.TokenKind_LParen},
					ast.MakeBinaryOpExpr(
						ast.MakeIdentifier(
							token.Token{Text: "total_amount", Kind: token.TokenKind_Identifier},
						),
						token.Token{Text: ">=", Kind: token.TokenKind_gte},
						ast.MakeLiteralSignedInteger(
							token.Token{Text: "0", Kind: token.TokenKind_IntegerNumericLiteral},
							0,
						),
					),
					token.Token{Kind: token.TokenKind_LParen},
				),
			},
		},
		&op.AddTableConstraint{
			Target: op.TargetTable{
				Schema: schemaA,
				Table:  schemaA.Tables["orders"],
			},
			Constraint: ast.MakeTableConstraintCheck(
				&ast.ConstraintName{
					Name: *ast.MakeIdentifier(token.Token{Text: "valid_checkout", Kind: token.TokenKind_Identifier}),
				},
				*CheckKeyword,
				token.Token{Kind: token.TokenKind_LParen},
				ast.MakeBinaryOpExpr(
					ast.MakeBinaryOpExpr(
						ast.MakeIdentifier(
							token.Token{Text: "status", Kind: token.TokenKind_Identifier},
						),
						token.Token{Text: "=", Kind: token.TokenKind_eq},
						ast.MakeLiteralString(
							token.Token{Text: "cart", Kind: token.TokenKind_StringLiteral},
							"cart",
						),
					),
					token.Token{Text: "OR", Kind: token.TokenKind_Keyword_OR},
					ast.MakeBinaryOpExpr(
						ast.MakeIdentifier(
							token.Token{Text: "total_amount", Kind: token.TokenKind_Identifier},
						),
						token.Token{Text: ">", Kind: token.TokenKind_gt},
						ast.MakeLiteralSignedInteger(
							token.Token{Text: "0", Kind: token.TokenKind_IntegerNumericLiteral},
							0,
						),
					),
				),
				token.Token{Kind: token.TokenKind_LParen},
			),
		},
	}

	if len(ops) != len(expectedOps) {
		t.Fatalf("expected ops length does not match ops len")
		t.FailNow()
	}

	opIs := func(a op.Op) func(b op.Op) bool {
		return func(b op.Op) bool {
			return a.Eq(b)
		}
	}

	for _, expected := range expectedOps {
		found := slices.ContainsFunc(ops, opIs(expected))

		if !found {
			t.Fatalf("op not found in ops: %T %+v", expected, expected)
		}
	}
}
