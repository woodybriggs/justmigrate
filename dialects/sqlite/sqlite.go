package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type Sqlite struct {
	FileName string
	*sql.DB
}

func (sqlite *Sqlite) Url() string {
	return sqlite.FileName
}

type schemaRow struct {
	Type      sql.NullString
	Name      sql.NullString
	TableName sql.NullString
	RootPage  sql.NullInt64
	Sql       sql.NullString
}

func (sqlite *Sqlite) ExportDataDefinitions() (string, error) {
	builder := strings.Builder{}

	rows, err := sqlite.Query("select type, name, tbl_name, rootpage, sql from sqlite_schema;")
	if err != nil {
		return "", err
	}

	ok := rows.Next()
	for ok {
		row := &schemaRow{}
		err := rows.Scan(&row.Type, &row.Name, &row.TableName, &row.RootPage, &row.Sql)
		if err != nil {
			log.Panicln(err)
		}

		if row.Sql.Valid {
			builder.WriteString("/* ")
			builder.WriteString(fmt.Sprintf("%s: %s", row.Type.String, row.Name.String))
			builder.WriteString(" */\n")
			builder.WriteString(row.Sql.String)
			builder.WriteRune(';')
			builder.WriteRune('\n')
			builder.WriteRune('\n')
		}

		ok = rows.Next()
	}

	return builder.String(), nil
}
