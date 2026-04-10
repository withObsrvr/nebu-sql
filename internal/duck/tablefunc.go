package duck

import (
	"context"
	"database/sql"
	"fmt"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/withObsrvr/nebu-sql/internal/processor"
)

type nebuSource struct {
	stream      *processor.Stream
	varcharType duckdb.TypeInfo
	columns     []string
}

func RegisterNebuTableFunction(conn *sql.Conn) error {
	varcharType, err := duckdb.NewTypeInfo(duckdb.TYPE_VARCHAR)
	if err != nil {
		return fmt.Errorf("create varchar type info: %w", err)
	}
	bigintType, err := duckdb.NewTypeInfo(duckdb.TYPE_BIGINT)
	if err != nil {
		return fmt.Errorf("create bigint type info: %w", err)
	}

	udf := duckdb.RowTableFunction{
		Config: duckdb.TableFunctionConfig{
			Arguments: []duckdb.TypeInfo{varcharType},
			NamedArguments: map[string]duckdb.TypeInfo{
				"start": bigintType,
				"stop":  bigintType,
			},
		},
		BindArgumentsContext: func(ctx context.Context, named map[string]any, args ...any) (duckdb.RowTableSource, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("nebu() expects a processor name as its first argument")
			}
			processorName, ok := args[0].(string)
			if !ok || processorName == "" {
				return nil, fmt.Errorf("nebu() first argument must be a non-empty processor name")
			}
			start, ok := named["start"].(int64)
			if !ok || start <= 0 {
				return nil, fmt.Errorf("nebu() requires named argument start > 0")
			}
			stop, ok := named["stop"].(int64)
			if !ok || stop <= 0 {
				return nil, fmt.Errorf("nebu() requires named argument stop > 0")
			}
			if stop < start {
				return nil, fmt.Errorf("nebu() requires stop >= start")
			}

			path, err := processor.LookPath(processorName)
			if err != nil {
				return nil, err
			}
			doc, err := processor.Describe(path)
			if err != nil {
				return nil, err
			}
			schema, err := processor.ExtractSchemaInfo(doc)
			if err != nil {
				return nil, err
			}
			stream, err := processor.StartRange(ctx, path, start, stop, schema.VariantFields)
			if err != nil {
				return nil, err
			}
			columns := buildColumns(schema.Fields)
			return &nebuSource{
				stream:      stream,
				varcharType: varcharType,
				columns:     columns,
			}, nil
		},
	}

	return duckdb.RegisterTableUDF(conn, "nebu", udf)
}

func (s *nebuSource) ColumnInfos() []duckdb.ColumnInfo {
	infos := make([]duckdb.ColumnInfo, 0, len(s.columns))
	for _, name := range s.columns {
		infos = append(infos, duckdb.ColumnInfo{Name: name, T: s.varcharType})
	}
	return infos
}

func (s *nebuSource) Init() {}

func (s *nebuSource) FillRow(row duckdb.Row) (bool, error) {
	rec, ok, err := s.stream.Next()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	for i, name := range s.columns {
		value := rec.Values[name]
		if name == "event_type" && rec.EventType != "" {
			value = rec.EventType
		}
		if err := duckdb.SetRowValue(row, i, value); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *nebuSource) Cardinality() *duckdb.CardinalityInfo {
	return nil
}

func buildColumns(fields []string) []string {
	columns := []string{"_schema", "_nebu_version", "event_type"}
	seen := map[string]struct{}{
		"_schema":       {},
		"_nebu_version": {},
		"event_type":    {},
	}
	for _, name := range fields {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		columns = append(columns, name)
	}
	return columns
}
