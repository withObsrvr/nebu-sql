package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractSchemaInfo_ResolvesRefAndVariants(t *testing.T) {
	doc := map[string]any{
		"schema": map[string]any{
			"output": map[string]any{
				"$defs": map[string]any{
					"example.Event": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"meta":     map[string]any{"type": "object"},
							"transfer": map[string]any{"type": "object"},
							"fee":      map[string]any{"type": "object"},
						},
						"oneOf": []any{
							map[string]any{"required": []any{"transfer"}},
							map[string]any{"required": []any{"fee"}},
						},
					},
				},
				"$ref": "#/$defs/example.Event",
			},
		},
	}

	info, err := ExtractSchemaInfo(doc)
	if err != nil {
		t.Fatalf("ExtractSchemaInfo() error = %v", err)
	}

	wantFields := []string{"fee", "meta", "transfer"}
	if !reflect.DeepEqual(info.Fields, wantFields) {
		t.Fatalf("Fields = %v, want %v", info.Fields, wantFields)
	}

	wantVariants := []string{"fee", "transfer"}
	if !reflect.DeepEqual(info.VariantFields, wantVariants) {
		t.Fatalf("VariantFields = %v, want %v", info.VariantFields, wantVariants)
	}
}

func TestExtractSchemaInfo_DirectProperties(t *testing.T) {
	doc := map[string]any{
		"schema": map[string]any{
			"output": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"contractId": map[string]any{"type": "string"},
					"eventType":  map[string]any{"type": "string"},
					"type":       map[string]any{"type": "string"},
				},
			},
		},
	}

	info, err := ExtractSchemaInfo(doc)
	if err != nil {
		t.Fatalf("ExtractSchemaInfo() error = %v", err)
	}

	wantFields := []string{"contractId", "eventType", "type"}
	if !reflect.DeepEqual(info.Fields, wantFields) {
		t.Fatalf("Fields = %v, want %v", info.Fields, wantFields)
	}
	if len(info.VariantFields) != 0 {
		t.Fatalf("VariantFields = %v, want empty", info.VariantFields)
	}
}

func TestParseRow_NormalizesValuesAndDetectsEventType(t *testing.T) {
	line := []byte(`{"_schema":"nebu.token-transfer.v1","_nebu_version":"0.6.3","meta":{"ledgerSequence":60200000},"fee":{"assetCode":"XLM","amount":"100"},"ok":true,"count":42}`)

	row, err := parseRow(line, map[string]struct{}{"fee": {}, "transfer": {}})
	if err != nil {
		t.Fatalf("parseRow() error = %v", err)
	}

	if row.EventType != "fee" {
		t.Fatalf("EventType = %q, want fee", row.EventType)
	}
	if got := row.Values["_schema"]; got != "nebu.token-transfer.v1" {
		t.Fatalf("_schema = %q", got)
	}
	if got := row.Values["ok"]; got != "true" {
		t.Fatalf("ok = %q, want true", got)
	}
	if got := row.Values["count"]; got != "42" {
		t.Fatalf("count = %q, want 42", got)
	}
	if got := row.Values["meta"]; got != `{"ledgerSequence":60200000}` {
		t.Fatalf("meta = %q", got)
	}
}

func TestNormalizeValue_PreservesLargeIntegersAndDecimals(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "large integer", raw: json.RawMessage(`9007199254740993`), want: "9007199254740993"},
		{name: "decimal", raw: json.RawMessage(`12.34`), want: "12.34"},
		{name: "string", raw: json.RawMessage(`"hello"`), want: "hello"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeValue(tc.raw); got != tc.want {
				t.Fatalf("normalizeValue(%s) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestDetectEventType_PrefersExplicitFields(t *testing.T) {
	raw := map[string]json.RawMessage{
		"eventType": json.RawMessage(`"swap"`),
		"fee":       json.RawMessage(`{"assetCode":"XLM"}`),
	}

	got := detectEventType(raw, map[string]struct{}{"fee": {}})
	if got != "swap" {
		t.Fatalf("detectEventType() = %q, want swap", got)
	}
}

func TestDescribe_UnsupportedDescribeJSON(t *testing.T) {
	path := writeExecutable(t, "no-describe", `#!/bin/sh
set -eu
if [ "${1-}" = "--describe-json" ]; then
  echo 'unknown flag: --describe-json' >&2
  exit 1
fi
`)

	_, err := Describe(path)
	if err == nil {
		t.Fatal("Describe() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "does not support --describe-json") {
		t.Fatalf("Describe() error = %v", err)
	}
}

func TestDescribe_MalformedJSON(t *testing.T) {
	path := writeExecutable(t, "bad-describe", `#!/bin/sh
set -eu
if [ "${1-}" = "--describe-json" ]; then
  echo '{'
  exit 0
fi
`)

	_, err := Describe(path)
	if err == nil {
		t.Fatal("Describe() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse ") {
		t.Fatalf("Describe() error = %v", err)
	}
}

func TestExtractSchemaInfo_UnsupportedRef(t *testing.T) {
	doc := map[string]any{
		"schema": map[string]any{
			"output": map[string]any{
				"$ref": "https://example.com/schema.json",
			},
		},
	}

	_, err := ExtractSchemaInfo(doc)
	if err == nil {
		t.Fatal("ExtractSchemaInfo() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unsupported schema ref") {
		t.Fatalf("ExtractSchemaInfo() error = %v", err)
	}
}

func writeExecutable(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}
