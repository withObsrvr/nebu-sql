package duck

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestRegisterNebuTableFunction_EndToEndWithFakeProcessor(t *testing.T) {
	tmpDir := t.TempDir()
	processorPath := filepath.Join(tmpDir, "fake-processor")
	script := `#!/bin/sh
set -eu
if [ "${1-}" = "--describe-json" ]; then
  cat <<'JSON'
{"name":"fake-processor","schema":{"id":"nebu.fake.v1","output":{"$defs":{"fake.Event":{"type":"object","properties":{"meta":{"type":"object"},"transfer":{"type":"object"},"fee":{"type":"object"},"contractId":{"type":"string"}},"oneOf":[{"required":["transfer"]},{"required":["fee"]}]}} ,"$ref":"#/$defs/fake.Event"}}}
JSON
  exit 0
fi
cat <<'JSONL'
{"_schema":"nebu.fake.v1","_nebu_version":"test","meta":{"ledgerSequence":1},"transfer":{"assetCode":"USDC"},"contractId":"C1"}
{"_schema":"nebu.fake.v1","_nebu_version":"test","meta":{"ledgerSequence":2},"fee":{"assetCode":"XLM"},"contractId":"C2"}
JSONL
`
	if err := os.WriteFile(processorPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake processor: %v", err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open conn: %v", err)
	}
	defer conn.Close()

	if err := RegisterNebuTableFunction(conn); err != nil {
		t.Fatalf("RegisterNebuTableFunction() error = %v", err)
	}

	rows, err := db.QueryContext(context.Background(), `
		SELECT _schema, _nebu_version, event_type, contractId, transfer, fee
		FROM nebu('fake-processor', start = 1, stop = 2)
		ORDER BY contractId
	`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	type result struct {
		schema     string
		version    string
		eventType  string
		contractID string
		transfer   string
		fee        string
	}
	var got []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.schema, &r.version, &r.eventType, &r.contractID, &r.transfer, &r.fee); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("row count = %d, want 2", len(got))
	}

	if got[0].schema != "nebu.fake.v1" || got[0].version != "test" || got[0].eventType != "transfer" || got[0].contractID != "C1" || got[0].transfer != `{"assetCode":"USDC"}` || got[0].fee != "" {
		t.Fatalf("first row = %+v", got[0])
	}
	if got[1].schema != "nebu.fake.v1" || got[1].version != "test" || got[1].eventType != "fee" || got[1].contractID != "C2" || got[1].transfer != "" || got[1].fee != `{"assetCode":"XLM"}` {
		t.Fatalf("second row = %+v", got[1])
	}
}
