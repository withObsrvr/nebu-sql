package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	_ "github.com/duckdb/duckdb-go/v2"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/withObsrvr/nebu-sql/internal/duck"
	"github.com/withObsrvr/nebu-sql/internal/version"
)

func Run(args []string) error {
	fs := flag.NewFlagSet("nebu-sql", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var query string
	var file string
	var jsonOutput bool
	var showVersion bool
	fs.StringVar(&query, "c", "", "SQL query to execute")
	fs.StringVar(&file, "file", "", "Path to a .sql file to execute")
	fs.BoolVar(&jsonOutput, "json", false, "Emit results as newline-delimited JSON objects")
	fs.BoolVar(&showVersion, "version", false, "Print nebu-sql version")

	if err := fs.Parse(args); err != nil {
		return usageError(err)
	}
	if showVersion {
		fmt.Println(version.String())
		return nil
	}
	if strings.TrimSpace(query) == "" && strings.TrimSpace(file) == "" {
		return usageError(errors.New("either -c or --file is required"))
	}
	if strings.TrimSpace(query) != "" && strings.TrimSpace(file) != "" {
		return usageError(errors.New("use either -c or --file, not both"))
	}
	if strings.TrimSpace(file) != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read query file %q: %w", file, err)
		}
		query = string(b)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open duckdb connection: %w", err)
	}
	defer conn.Close()

	if err := duck.RegisterNebuTableFunction(conn); err != nil {
		return fmt.Errorf("register nebu() table function: %w", err)
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if isCanceled(ctx, err) {
			return nil
		}
		return fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()

	if err := printRows(rows, jsonOutput); err != nil {
		if isCanceled(ctx, err) {
			return nil
		}
		return err
	}
	return nil
}

func usageError(err error) error {
	return fmt.Errorf("%w\n\nusage:\n  nebu-sql -c \"SELECT ...\"\n  nebu-sql --file query.sql\n  nebu-sql --version\n\noptions:\n  --json      emit newline-delimited JSON rows\n  --version   print version", err)
}

func printRows(rows *sql.Rows, jsonOutput bool) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("read columns: %w", err)
	}

	values := make([]any, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		for rows.Next() {
			if err := rows.Scan(scanArgs...); err != nil {
				return fmt.Errorf("scan row: %w", err)
			}
			obj := make(map[string]any, len(cols))
			for i, col := range cols {
				obj[col] = normalizedValue(values[i])
			}
			if err := enc.Encode(obj); err != nil {
				return fmt.Errorf("encode row as json: %w", err)
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate rows: %w", err)
		}
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(cols, "\t"))
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}
		parts := make([]string, len(values))
		for i, v := range values {
			parts[i] = formatValue(v)
		}
		fmt.Fprintln(tw, strings.Join(parts, "\t"))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}

func normalizedValue(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}

func isCanceled(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "context canceled") || strings.Contains(msg, "signal: killed")
}

var _ = duckdb.RegisterTableUDF[duckdb.RowTableFunction]
