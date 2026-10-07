package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestParseArgs_Query(t *testing.T) {
	cfg, err := parseArgs([]string{"-c", "select 1"}, func(string) ([]byte, error) {
		t.Fatal("readFile should not be called")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if cfg.Query != "select 1" {
		t.Fatalf("Query = %q, want %q", cfg.Query, "select 1")
	}
	if cfg.JSONOutput {
		t.Fatal("JSONOutput = true, want false")
	}
	if cfg.ShowVersion {
		t.Fatal("ShowVersion = true, want false")
	}
}

func TestParseArgs_JSONOutput(t *testing.T) {
	cfg, err := parseArgs([]string{"--json", "-c", "select 1"}, func(string) ([]byte, error) {
		t.Fatal("readFile should not be called")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if cfg.Query != "select 1" {
		t.Fatalf("Query = %q, want %q", cfg.Query, "select 1")
	}
	if !cfg.JSONOutput {
		t.Fatal("JSONOutput = false, want true")
	}
	if cfg.ShowVersion {
		t.Fatal("ShowVersion = true, want false")
	}
}

func TestParseArgs_File(t *testing.T) {
	cfg, err := parseArgs([]string{"--file", "q.sql"}, func(path string) ([]byte, error) {
		if path != "q.sql" {
			t.Fatalf("readFile path = %q, want q.sql", path)
		}
		return []byte("select 42"), nil
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if cfg.Query != "select 42" {
		t.Fatalf("Query = %q, want %q", cfg.Query, "select 42")
	}
}

func TestParseArgs_QueryAndFileConflict(t *testing.T) {
	_, err := parseArgs([]string{"-c", "select 1", "--file", "q.sql"}, func(string) ([]byte, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("parseArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "use either -c or --file, not both") {
		t.Fatalf("parseArgs() error = %v", err)
	}
}

func TestParseArgs_RequiresQuerySource(t *testing.T) {
	_, err := parseArgs(nil, func(string) ([]byte, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("parseArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "either -c or --file is required") {
		t.Fatalf("parseArgs() error = %v", err)
	}
}

func TestParseArgs_VersionDoesNotRequireQuery(t *testing.T) {
	cfg, err := parseArgs([]string{"--version"}, func(string) ([]byte, error) {
		t.Fatal("readFile should not be called")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if !cfg.ShowVersion {
		t.Fatal("ShowVersion = false, want true")
	}
}

func TestParseArgs_FileReadError(t *testing.T) {
	_, err := parseArgs([]string{"--file", "q.sql"}, func(string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if err == nil {
		t.Fatal("parseArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `read query file "q.sql"`) {
		t.Fatalf("parseArgs() error = %v", err)
	}
}

func TestParseArgs_InvalidFlag(t *testing.T) {
	_, err := parseArgs([]string{"--wat"}, func(string) ([]byte, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("parseArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("parseArgs() error = %v", err)
	}
}

func TestPrintRows_TableOutput(t *testing.T) {
	rows := queryRows(t, `SELECT 1 AS a, 'x' AS b`)
	defer rows.Close()

	var buf bytes.Buffer
	if err := printRows(&buf, rows, false); err != nil {
		t.Fatalf("printRows() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Fatalf("output missing headers: %q", out)
	}
	if !strings.Contains(out, "1") || !strings.Contains(out, "x") {
		t.Fatalf("output missing row values: %q", out)
	}
}

func TestPrintRows_JSONOutput(t *testing.T) {
	rows := queryRows(t, `SELECT 1 AS a, 'x' AS b`)
	defer rows.Close()

	var buf bytes.Buffer
	if err := printRows(&buf, rows, true); err != nil {
		t.Fatalf("printRows() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got["a"] != float64(1) {
		t.Fatalf("a = %#v, want 1", got["a"])
	}
	if got["b"] != "x" {
		t.Fatalf("b = %#v, want x", got["b"])
	}
}

func TestRunCLI_VersionOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := runCLI([]string{"--version"}, &buf, func(string) ([]byte, error) {
		t.Fatal("readFile should not be called")
		return nil, nil
	}); err != nil {
		t.Fatalf("runCLI() error = %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got == "" {
		t.Fatal("version output is empty")
	}
}

func TestPrintRows_NullFormatting(t *testing.T) {
	rows := queryRows(t, `SELECT NULL AS a`)
	defer rows.Close()

	var buf bytes.Buffer
	if err := printRows(&buf, rows, false); err != nil {
		t.Fatalf("printRows() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a") || !strings.Contains(out, "NULL") {
		t.Fatalf("output = %q, want header and NULL", out)
	}
}

func TestIsCanceled(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	liveCtx := context.Background()

	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{name: "context canceled", ctx: canceledCtx, err: context.Canceled, want: true},
		{name: "wrapped context canceled", ctx: liveCtx, err: errors.New("context canceled"), want: true},
		{name: "signal killed", ctx: liveCtx, err: errors.New("processor exited with error: signal: killed"), want: true},
		{name: "other", ctx: liveCtx, err: errors.New("boom"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isCanceled(tc.ctx, tc.err)
			if got != tc.want {
				t.Fatalf("isCanceled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func queryRows(t *testing.T, query string) *sql.Rows {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	return rows
}

func TestParseArgsProgramStatus(t *testing.T) {
	cfg, err := parseArgs([]string{"--program-status", "always", "-c", "select 1"}, func(string) ([]byte, error) {
		return nil, errors.New("unexpected read")
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if cfg.ProgramStatus != "always" {
		t.Fatalf("ProgramStatus = %q, want always", cfg.ProgramStatus)
	}

	_, err = parseArgs([]string{"--program-status", "sometimes", "-c", "select 1"}, func(string) ([]byte, error) {
		return nil, errors.New("unexpected read")
	})
	if err == nil || !strings.Contains(err.Error(), "expected auto, always, or never") {
		t.Fatalf("parseArgs() error = %v, want invalid program status", err)
	}
}

func TestRunCLIProgramStatus(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLIWithStatus(
		[]string{"--program-status", "always", "-c", "select 1"},
		&stdout,
		&stderr,
		func() bool { return false },
		func(string) ([]byte, error) { return nil, errors.New("unexpected read") },
	)
	if err != nil {
		t.Fatalf("runCLIWithStatus() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "state=working") || !strings.Contains(stderr.String(), "state=done") {
		t.Fatalf("status output = %q, want working and done", stderr.String())
	}
}

func TestReportQueryResult(t *testing.T) {
	tests := []struct {
		name      string
		canceled  bool
		err       error
		wantState string
	}{
		{name: "success", wantState: "state=done"},
		{name: "error", err: errors.New("query failed"), wantState: "state=error"},
		{name: "canceled", canceled: true, err: context.Canceled, wantState: "state=idle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.canceled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			var output bytes.Buffer
			status := queryStatus{mode: "always", writer: &output, isTerminal: func() bool { return false }}

			reportQueryResult(ctx, status, tt.err)
			if !strings.Contains(output.String(), tt.wantState) {
				t.Fatalf("status output = %q, want %s", output.String(), tt.wantState)
			}
		})
	}
}

func TestTerminalFileRejectsNonTerminalCharacterDevice(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = devNull.Close() })
	if terminalFile(devNull) {
		t.Fatal("os.DevNull reported as a terminal")
	}
}
