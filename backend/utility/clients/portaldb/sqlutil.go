package portaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	portalshared "higress-portal-backend/schema/shared"
)

type execContexter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type queryContexter interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type queryRowContexter interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func IsPostgresDriver(driver string) bool {
	return portalshared.IsPostgresDriver(driver)
}

func Rebind(driver, query string) string {
	if !IsPostgresDriver(driver) {
		return query
	}
	var (
		builder        strings.Builder
		index          int
		inSingleQuotes bool
	)
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			if inSingleQuotes && i+1 < len(query) && query[i+1] == '\'' {
				builder.WriteByte(ch)
				builder.WriteByte(query[i+1])
				i++
				continue
			}
			inSingleQuotes = !inSingleQuotes
			builder.WriteByte(ch)
			continue
		}
		if ch == '?' && !inSingleQuotes {
			index++
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(index))
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func ExecContext(ctx context.Context, execer execContexter, driver, query string, args ...any) (sql.Result, error) {
	return execer.ExecContext(ctx, Rebind(driver, query), args...)
}

func QueryContext(ctx context.Context, queryer queryContexter, driver, query string, args ...any) (*sql.Rows, error) {
	return queryer.QueryContext(ctx, Rebind(driver, query), args...)
}

func QueryRowContext(ctx context.Context, queryer queryRowContexter, driver, query string, args ...any) *sql.Row {
	return queryer.QueryRowContext(ctx, Rebind(driver, query), args...)
}

func AssignValue(driver, column string) string {
	_ = driver
	return fmt.Sprintf("%s = EXCLUDED.%s", column, column)
}

func UpsertClause(driver string, conflictColumns []string, assignments ...string) string {
	_ = driver
	return fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s", strings.Join(conflictColumns, ", "), strings.Join(assignments, ", "))
}

func UpsertAdd(driver, tableName, column string) string {
	_ = driver
	return fmt.Sprintf("%s = %s.%s + EXCLUDED.%s", column, tableName, column, column)
}

func UTCCurrentTimestamp(driver string) string {
	_ = driver
	return "TIMEZONE('UTC', CURRENT_TIMESTAMP)"
}

// InsertReturningID executes an INSERT statement and returns the generated id.
func InsertReturningID(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, driver, query string, args ...any) (int64, error) {
	_ = driver
	var id int64
	err := db.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&id)
	return id, err
}

func CastToText(driver, expr string) string {
	_ = driver
	return fmt.Sprintf("CAST(%s AS TEXT)", expr)
}
