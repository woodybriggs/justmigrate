package main

import "fmt"

type TokenKind int

var tokenKindDebugString = map[TokenKind]string{
	0:                               "eof",
	'(':                             "l-paren",
	')':                             "r-paren",
	',':                             "comma",
	'.':                             "period",
	';':                             "semi-colon",
	'=':                             "equal",
	'+':                             "plus",
	'-':                             "minus",
	'`':                             "backtic",
	'>':                             "greater-than",
	'<':                             "less-than",
	'!':                             "not",
	TokenKind_neq:                   "not-equal",
	TokenKind_gte:                   "greater-than-equal",
	TokenKind_lte:                   "less-than-equal",
	TokenKind_Identifier:            "identifier",
	TokenKind_DecimalNumericLiteral: "decimal-numeric-literal",
	TokenKind_HexNumericLiteral:     "hex-numeric-literal",
	TokenKind_BinaryNumericLiteral:  "binary-numeric-literal",
	TokenKind_OctalNumericLiteral:   "octal-numeric-literal",
	TokenKind_StringLiteral:         "string-literal",
}

func (k TokenKind) DebugString() string {
	if val, ok := tokenKindDebugString[k]; ok {
		return val
	}
	if str, ok := keywordIndex.GetKey(k); ok {
		return str
	}
	return "unknown-kind"
}

const (
	TokenKindOffset_ASCII    TokenKind = 0
	TokenKindOffset_Atoms    TokenKind = 257
	TokenKindOffset_Keywords TokenKind = 400
)

const (
	TokenKind_Error     TokenKind = -1
	TokenKind_EOF       TokenKind = TokenKindOffset_ASCII
	TokenKind_LParen    TokenKind = '('
	TokenKind_RParen    TokenKind = ')'
	TokenKind_Comma     TokenKind = ','
	TokenKind_Period    TokenKind = '.'
	TokenKind_SemiColon TokenKind = ';'
	TokenKind_Plus      TokenKind = '+'
	TokenKind_Minus     TokenKind = '-'
	TokenKind_Backtic   TokenKind = '`'
	TokenKind_gt        TokenKind = '>'
	TokenKind_lt        TokenKind = '<'
	TokenKind_not       TokenKind = '!'
)

const (
	TokenKind_Identifier TokenKind = iota + 1 + TokenKindOffset_Atoms
	TokenKind_neq
	TokenKind_gte
	TokenKind_lte
	TokenKind_DecimalNumericLiteral
	TokenKind_HexNumericLiteral
	TokenKind_BinaryNumericLiteral
	TokenKind_OctalNumericLiteral
	TokenKind_StringLiteral
	TokenKind_Comment
	TokenKind_Raw
)

const (
	TokenKind_Keyword_CREATE TokenKind = iota + 1 + TokenKindOffset_Keywords
	TokenKind_Keyword_SELECT
	TokenKind_Keyword_DELETE
	TokenKind_Keyword_UPDATE
	TokenKind_Keyword_EXPLAIN
	TokenKind_Keyword_QUERY
	TokenKind_Keyword_PLAN
	TokenKind_Keyword_TEMPORARY
	TokenKind_Keyword_VIRTUAL
	TokenKind_Keyword_STORED
	TokenKind_Keyword_INDEX
	TokenKind_Keyword_TABLE
	TokenKind_Keyword_VIEW
	TokenKind_Keyword_TRIGGER
	TokenKind_Keyword_ALWAYS
	TokenKind_Keyword_AS
	TokenKind_Keyword_IF
	TokenKind_Keyword_NOT
	TokenKind_Keyword_EXISTS
	TokenKind_Keyword_NULL
	TokenKind_Keyword_PRAMGA
	TokenKind_Keyword_BEGIN
	TokenKind_Keyword_TRANSACTION
	TokenKind_Keyword_COMMIT
	TokenKind_Keyword_USING

	TokenKind_Keyword_CONSTRAINT
	TokenKind_Keyword_PRIMARY
	TokenKind_Keyword_FOREIGN
	TokenKind_Keyword_KEY
	TokenKind_Keyword_AUTOINCREMENT
	TokenKind_Keyword_UNIQUE
	TokenKind_Keyword_CHECK
	TokenKind_Keyword_DEFAULT
	TokenKind_Keyword_COLLATE
	TokenKind_Keyword_REFERENCES
	TokenKind_Keyword_GENERATED

	TokenKind_Keyword_ASC
	TokenKind_Keyword_DESC
	TokenKind_Keyword_ON
	TokenKind_Keyword_CONFLICT
	TokenKind_Keyword_MATCH
	TokenKind_Keyword_DEFERRABLE

	TokenKind_Keyword_ROLLBACK
	TokenKind_Keyword_ABORT
	TokenKind_Keyword_FAIL
	TokenKind_Keyword_IGNORE
	TokenKind_Keyword_REPLACE

	TokenKind_Keyword_TRUE
	TokenKind_Keyword_FALSE

	TokenKind_Keyword_IN

	TokenKind_Keyword_CASCADE
	TokenKind_Keyword_RESTRICT
	TokenKind_Keyword_NO
	TokenKind_Keyword_SET
	TokenKind_Keyword_ACTION

	TokenKind_Keyword_INITIALLY
	TokenKind_Keyword_IMMEDIATE
	TokenKind_Keyword_DEFERRED

	TokenKind_Keyword_STRICT
	TokenKind_Keyword_WITHOUT
	TokenKind_Keyword_ROWID

	TokenKind_Keyword_CASE
	TokenKind_Keyword_WHEN
	TokenKind_Keyword_THEN
	TokenKind_Keyword_ELSE
	TokenKind_Keyword_END
	TokenKind_Keyword_WHERE
)

const (
	Keyword_PRAGMA        string = "pragma"
	Keyword_CREATE        string = "create"
	Keyword_TEMP          string = "temp"
	Keyword_TEMPORARY     string = "temporary"
	Keyword_TABLE         string = "table"
	Keyword_INDEX         string = "index"
	Keyword_VIEW          string = "view"
	Keyword_TRIGGER       string = "trigger"
	Keyword_AS            string = "as"
	Keyword_IF            string = "if"
	Keyword_NOT           string = "not"
	Keyword_EXISTS        string = "exists"
	Keyword_NULL          string = "null"
	Keyword_CONSTRAINT    string = "constraint"
	Keyword_PRIMARY       string = "primary"
	Keyword_FOREIGN       string = "foreign"
	Keyword_KEY           string = "key"
	Keyword_UNIQUE        string = "unique"
	Keyword_CHECK         string = "check"
	Keyword_DEFAULT       string = "default"
	Keyword_COLLATE       string = "collate"
	Keyword_REFERENCES    string = "references"
	Keyword_GENERATED     string = "generated"
	Keyword_ASC           string = "asc"
	Keyword_DESC          string = "desc"
	Keyword_ON            string = "on"
	Keyword_CONFLICT      string = "conflict"
	Keyword_ROLLBACK      string = "rollback"
	Keyword_ABORT         string = "abort"
	Keyword_FAIL          string = "fail"
	Keyword_IGNORE        string = "ignore"
	Keyword_REPLACE       string = "replace"
	Keyword_EXPLAIN       string = "explain"
	Keyword_QUERY         string = "query"
	Keyword_PLAN          string = "plan"
	Keyword_BEGIN         string = "begin"
	Keyword_COMMIT        string = "commit"
	Keyword_TRANSACTION   string = "transaction"
	Keyword_AUTOINCREMENT string = "autoincrement"
	Keyword_TRUE          string = "true"
	Keyword_FALSE         string = "false"
	Keyword_IN            string = "in"
	Keyword_ALWAYS        string = "always"
	Keyword_STORED        string = "stored"
	Keyword_VIRTUAL       string = "virtual"
	Keyword_MATCH         string = "match"
	Keyword_DEFERRABLE    string = "deferrable"
	Keyword_DELETE        string = "delete"
	Keyword_UPDATE        string = "update"
	Keyword_CASCADE       string = "cascade"
	Keyword_RESTRICT      string = "restrict"
	Keyword_NO            string = "no"
	Keyword_SET           string = "set"
	Keyword_ACTION        string = "action"
	Keyword_INITIALLY     string = "initially"
	Keyword_IMMEDIATE     string = "immediate"
	Keyword_DEFERRED      string = "deferred"
	Keyword_STRICT        string = "strict"
	Keyword_WITHOUT       string = "without"
	Keyword_ROWID         string = "rowid"
	Keyword_CASE          string = "case"
	Keyword_WHEN          string = "when"
	Keyword_THEN          string = "then"
	Keyword_ELSE          string = "else"
	Keyword_END           string = "end"
	Keyword_USING         string = "using"
	Keyword_WHERE         string = "where"
	Keyword_SELECT        string = "select"
)

type MapIndex[TKey comparable, TVal comparable] struct {
	kv map[TKey]TVal
	vk map[TVal]TKey
}

func NewIndex[TKey comparable, TVal comparable]() *MapIndex[TKey, TVal] {
	return &MapIndex[TKey, TVal]{
		kv: map[TKey]TVal{},
		vk: map[TVal]TKey{},
	}
}

func (i *MapIndex[TKey, TVal]) Add(key TKey, value TVal) *MapIndex[TKey, TVal] {
	i.kv[key] = value
	i.vk[value] = key
	return i
}

func (i *MapIndex[TKey, TVal]) GetValue(key TKey) (TVal, bool) {
	res, ok := i.kv[key]
	return res, ok
}

func (i *MapIndex[TKey, TVal]) GetKey(value TVal) (TKey, bool) {
	res, ok := i.vk[value]
	return res, ok
}

var keywordIndex = NewIndex[string, TokenKind]().
	Add(Keyword_SELECT, TokenKind_Keyword_SELECT).
	Add(Keyword_CREATE, TokenKind_Keyword_CREATE).
	Add(Keyword_TEMP, TokenKind_Keyword_TEMPORARY).
	Add(Keyword_TEMPORARY, TokenKind_Keyword_TEMPORARY).
	Add(Keyword_TABLE, TokenKind_Keyword_TABLE).
	Add(Keyword_INDEX, TokenKind_Keyword_INDEX).
	Add(Keyword_VIEW, TokenKind_Keyword_VIEW).
	Add(Keyword_TRIGGER, TokenKind_Keyword_TRIGGER).
	Add(Keyword_AS, TokenKind_Keyword_AS).
	Add(Keyword_IF, TokenKind_Keyword_IF).
	Add(Keyword_NOT, TokenKind_Keyword_NOT).
	Add(Keyword_EXISTS, TokenKind_Keyword_EXISTS).
	Add(Keyword_NULL, TokenKind_Keyword_NULL).
	Add(Keyword_CONSTRAINT, TokenKind_Keyword_CONSTRAINT).
	Add(Keyword_PRIMARY, TokenKind_Keyword_PRIMARY).
	Add(Keyword_FOREIGN, TokenKind_Keyword_FOREIGN).
	Add(Keyword_KEY, TokenKind_Keyword_KEY).
	Add(Keyword_UNIQUE, TokenKind_Keyword_UNIQUE).
	Add(Keyword_CHECK, TokenKind_Keyword_CHECK).
	Add(Keyword_DEFAULT, TokenKind_Keyword_DEFAULT).
	Add(Keyword_COLLATE, TokenKind_Keyword_COLLATE).
	Add(Keyword_REFERENCES, TokenKind_Keyword_REFERENCES).
	Add(Keyword_GENERATED, TokenKind_Keyword_GENERATED).
	Add(Keyword_ASC, TokenKind_Keyword_ASC).
	Add(Keyword_DESC, TokenKind_Keyword_DESC).
	Add(Keyword_ON, TokenKind_Keyword_ON).
	Add(Keyword_CONFLICT, TokenKind_Keyword_CONFLICT).
	Add(Keyword_ROLLBACK, TokenKind_Keyword_ROLLBACK).
	Add(Keyword_ABORT, TokenKind_Keyword_ABORT).
	Add(Keyword_FAIL, TokenKind_Keyword_FAIL).
	Add(Keyword_IGNORE, TokenKind_Keyword_IGNORE).
	Add(Keyword_REPLACE, TokenKind_Keyword_REPLACE).
	Add(Keyword_PRAGMA, TokenKind_Keyword_PRAMGA).
	Add(Keyword_BEGIN, TokenKind_Keyword_BEGIN).
	Add(Keyword_TRANSACTION, TokenKind_Keyword_TRANSACTION).
	Add(Keyword_COMMIT, TokenKind_Keyword_COMMIT).
	Add(Keyword_AUTOINCREMENT, TokenKind_Keyword_AUTOINCREMENT).
	Add(Keyword_TRUE, TokenKind_Keyword_TRUE).
	Add(Keyword_FALSE, TokenKind_Keyword_FALSE).
	Add(Keyword_IN, TokenKind_Keyword_IN).
	Add(Keyword_ALWAYS, TokenKind_Keyword_ALWAYS).
	Add(Keyword_VIRTUAL, TokenKind_Keyword_VIRTUAL).
	Add(Keyword_STORED, TokenKind_Keyword_STORED).
	Add(Keyword_MATCH, TokenKind_Keyword_MATCH).
	Add(Keyword_DEFERRABLE, TokenKind_Keyword_DEFERRABLE).
	Add(Keyword_DELETE, TokenKind_Keyword_DELETE).
	Add(Keyword_UPDATE, TokenKind_Keyword_UPDATE).
	Add(Keyword_CASCADE, TokenKind_Keyword_CASCADE).
	Add(Keyword_RESTRICT, TokenKind_Keyword_RESTRICT).
	Add(Keyword_NO, TokenKind_Keyword_NO).
	Add(Keyword_SET, TokenKind_Keyword_SET).
	Add(Keyword_ACTION, TokenKind_Keyword_ACTION).
	Add(Keyword_INITIALLY, TokenKind_Keyword_INITIALLY).
	Add(Keyword_IMMEDIATE, TokenKind_Keyword_IMMEDIATE).
	Add(Keyword_DEFERRED, TokenKind_Keyword_DEFERRED).
	Add(Keyword_STRICT, TokenKind_Keyword_STRICT).
	Add(Keyword_WITHOUT, TokenKind_Keyword_WITHOUT).
	Add(Keyword_ROWID, TokenKind_Keyword_ROWID).
	Add(Keyword_CASE, TokenKind_Keyword_CASE).
	Add(Keyword_WHEN, TokenKind_Keyword_WHEN).
	Add(Keyword_THEN, TokenKind_Keyword_THEN).
	Add(Keyword_ELSE, TokenKind_Keyword_ELSE).
	Add(Keyword_END, TokenKind_Keyword_END).
	Add(Keyword_USING, TokenKind_Keyword_USING).
	Add(Keyword_WHERE, TokenKind_Keyword_WHERE)

var constaintKeywords = map[TokenKind]bool{
	TokenKind_Keyword_CONSTRAINT: true,
	TokenKind_Keyword_PRIMARY:    true,
	TokenKind_Keyword_FOREIGN:    true,
	TokenKind_Keyword_UNIQUE:     true,
	TokenKind_Keyword_CHECK:      true,
	TokenKind_Keyword_DEFAULT:    true,
	TokenKind_Keyword_COLLATE:    true,
	TokenKind_Keyword_REFERENCES: true,
	TokenKind_Keyword_GENERATED:  true,
}

type TextRange struct {
	Start int
	End   int
}

type Token struct {
	Text        string
	SourceRange TextRange
	Kind        TokenKind

	FileLoc Location

	LeadingTrivia  string
	TrailingTrivia string
}

func (t Token) String() string {
	return fmt.Sprintf("%s%s%s", t.LeadingTrivia, t.Text, t.TrailingTrivia)
}

func (t Token) DebugString() string {
	if str, ok := keywordIndex.GetKey(t.Kind); ok {
		return str
	}
	if str, ok := tokenKindDebugString[t.Kind]; ok {
		return str
	}
	return t.Text
}
