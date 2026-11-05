// Package postgres is the PostgreSQL connection pool of the application and its schema migrations.
package postgres

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"od/internal/config"
)

// QueryTimeout is the longest a single query may take, so a hung database cannot hang the application
const QueryTimeout = 30 * time.Second

type DB struct {
	pool *pgxpool.Pool
}

// Connect creates the pool. Connections are opened lazily, so this succeeds even before the
// database exists (see EnsureDatabase).
func Connect(ctx context.Context, cfg config.Database) (*DB, error) {
	pool, err := pgxpool.New(ctx, cfg.URL(cfg.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, QueryTimeout)
}

// Query runs a SELECT (or a statement with RETURNING) and returns every row as a map of column
// name to value. A NULL column is nil in the map; an empty result is an empty slice, not nil.
func (db *DB) Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// QueryInto runs a query and scans every row into the slice of structs that target points to
// (var rows []userRow; db.QueryInto(ctx, &rows, ...)). Columns are matched to fields by the db
// tag or the lower-cased field name, also inside embedded structs; columns without a field are
// ignored and nullable columns need pointer fields. An empty result leaves an empty slice, not nil.
func (db *DB) QueryInto(ctx context.Context, target any, sql string, args ...any) error {
	sliceValue := reflect.ValueOf(target).Elem()
	elementType := sliceValue.Type().Elem()

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	sliceValue.Set(reflect.MakeSlice(sliceValue.Type(), 0, 0))
	columns := rows.FieldDescriptions()
	for rows.Next() {
		element := reflect.New(elementType).Elem()
		fieldsByColumn := structFieldsByColumn(element)
		destinations := make([]any, len(columns))
		for index, column := range columns {
			if field, found := fieldsByColumn[column.Name]; found {
				destinations[index] = field.Addr().Interface()
			} else {
				destinations[index] = new(any)
			}
		}
		if err := rows.Scan(destinations...); err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, element))
	}
	return rows.Err()
}

func structFieldsByColumn(structValue reflect.Value) map[string]reflect.Value {
	fields := map[string]reflect.Value{}
	structType := structValue.Type()
	for index := 0; index < structType.NumField(); index++ {
		fieldType := structType.Field(index)
		if fieldType.Anonymous && fieldType.Type.Kind() == reflect.Struct {
			maps.Copy(fields, structFieldsByColumn(structValue.Field(index)))
			continue
		}
		if !fieldType.IsExported() {
			continue
		}
		column := fieldType.Tag.Get("db")
		if column == "-" {
			continue
		}
		if column == "" {
			column = strings.ToLower(fieldType.Name)
		}
		fields[column] = structValue.Field(index)
	}
	return fields
}

// Exec runs an INSERT / UPDATE / DELETE / DDL statement and returns the number of affected rows.
// Without arguments, several statements separated by semicolons are allowed.
func (db *DB) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	commandTag, err := db.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// CountRow receives the result of "SELECT COUNT(*) AS total ..."
type CountRow struct {
	Total int `db:"total"`
}
