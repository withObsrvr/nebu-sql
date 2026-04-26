package processor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SchemaInfo struct {
	Fields        []string
	VariantFields []string
}

type Row struct {
	Values    map[string]string
	EventType string
}

type Stream struct {
	ctx          context.Context
	cmd          *exec.Cmd
	scanner      *bufio.Scanner
	stderr       bytes.Buffer
	waitDone     bool
	variantHints map[string]struct{}

	mu        sync.Mutex
	inNext    bool
	idleSince time.Time
}

var idleReapGrace = 250 * time.Millisecond
var idleReapPoll = 50 * time.Millisecond

func LookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("processor %q not found on PATH; try: nebu install %s", name, name)
	}
	return path, nil
}

func Describe(path string) (map[string]any, error) {
	cmd := exec.Command(path, "--describe-json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := string(bytes.TrimSpace(out))
		if strings.Contains(trimmed, "unknown flag: --describe-json") {
			return nil, fmt.Errorf("processor %q does not support --describe-json; reinstall or update it with `nebu install %s`", path, filepath.Base(path))
		}
		return nil, fmt.Errorf("run %s --describe-json: %w\n%s", path, err, bytes.TrimSpace(out))
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse %s --describe-json output: %w", path, err)
	}
	return doc, nil
}

// ExtractSchemaInfo reads the constrained schema shape emitted by nebu
// processors today. It supports either direct top-level properties or a
// top-level $ref into schema.output.$defs, and uses oneOf.required entries to
// infer variant fields such as transfer/fee row shapes.
func ExtractSchemaInfo(doc map[string]any) (*SchemaInfo, error) {
	root, ok := doc["schema"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("describe output missing schema object")
	}
	output, ok := root["output"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("describe output missing schema.output object")
	}
	resolved, err := resolveSchemaNode(output, output)
	if err != nil {
		return nil, err
	}
	properties, _ := resolved["properties"].(map[string]any)
	fields := make([]string, 0, len(properties))
	for name := range properties {
		fields = append(fields, name)
	}
	sort.Strings(fields)
	variantFields := extractVariantFields(resolved)
	return &SchemaInfo{Fields: fields, VariantFields: variantFields}, nil
}

func StartRange(ctx context.Context, path string, start, stop int64, variantFields []string) (*Stream, error) {
	args := []string{
		"--start-ledger", strconv.FormatInt(start, 10),
		"--end-ledger", strconv.FormatInt(stop, 10),
		"-q",
	}
	cmd := exec.CommandContext(ctx, path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout for %s: %w", path, err)
	}
	stream := &Stream{ctx: ctx, cmd: cmd, variantHints: make(map[string]struct{}, len(variantFields))}
	for _, name := range variantFields {
		stream.variantHints[name] = struct{}{}
	}
	cmd.Stderr = &stream.stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", path, err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	stream.scanner = scanner
	go stream.reapIfAbandoned()
	return stream, nil
}

func (s *Stream) Next() (*Row, bool, error) {
	s.markNextStart()
	defer s.markNextDone()

	if s.scanner.Scan() {
		line := append([]byte(nil), s.scanner.Bytes()...)
		row, err := parseRow(line, s.variantHints)
		if err != nil {
			return nil, false, err
		}
		return row, true, nil
	}
	if err := s.scanner.Err(); err != nil {
		_ = s.wait()
		return nil, false, fmt.Errorf("scan processor output: %w", err)
	}
	if err := s.wait(); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

func (s *Stream) Close() error {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	s.mu.Lock()
	waitDone := s.waitDone
	s.mu.Unlock()
	if waitDone {
		return nil
	}

	_ = s.cmd.Process.Kill()
	return s.wait()
}

func (s *Stream) wait() error {
	s.mu.Lock()
	if s.waitDone {
		s.mu.Unlock()
		return nil
	}
	s.waitDone = true
	s.mu.Unlock()

	if err := s.cmd.Wait(); err != nil {
		if s.ctx != nil && s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		stderr := bytes.TrimSpace(s.stderr.Bytes())
		if len(stderr) > 0 {
			return fmt.Errorf("processor exited with error: %w\n%s", err, stderr)
		}
		return fmt.Errorf("processor exited with error: %w", err)
	}
	return nil
}

func (s *Stream) markNextStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inNext = true
	s.idleSince = time.Time{}
}

func (s *Stream) markNextDone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inNext = false
	s.idleSince = time.Now()
}

func (s *Stream) reapIfAbandoned() {
	ticker := time.NewTicker(idleReapPoll)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		waitDone := s.waitDone
		inNext := s.inNext
		idleSince := s.idleSince
		s.mu.Unlock()

		if waitDone {
			return
		}
		if inNext || idleSince.IsZero() {
			continue
		}
		if time.Since(idleSince) < idleReapGrace {
			continue
		}
		_ = s.Close()
		return
	}
}

func parseRow(line []byte, variantHints map[string]struct{}) (*Row, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("parse JSONL event: %w", err)
	}
	values := make(map[string]string, len(raw))
	for key, value := range raw {
		values[key] = normalizeValue(value)
	}
	row := &Row{Values: values}
	row.EventType = detectEventType(raw, variantHints)
	return row, nil
}

func detectEventType(raw map[string]json.RawMessage, variantHints map[string]struct{}) string {
	if s := jsonString(raw["event_type"]); s != "" {
		return s
	}
	if s := jsonString(raw["eventType"]); s != "" {
		return s
	}
	for key := range raw {
		if _, ok := variantHints[key]; ok && len(raw[key]) > 0 && string(raw[key]) != "null" {
			return key
		}
	}
	return ""
}

func resolveSchemaNode(node map[string]any, root map[string]any) (map[string]any, error) {
	ref, _ := node["$ref"].(string)
	if ref == "" {
		return node, nil
	}
	if !strings.HasPrefix(ref, "#/$defs/") {
		return nil, fmt.Errorf("unsupported schema ref %q", ref)
	}
	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema output missing $defs for ref %q", ref)
	}
	name := strings.TrimPrefix(ref, "#/$defs/")
	resolvedAny, ok := defs[name]
	if !ok {
		return nil, fmt.Errorf("schema ref %q not found", ref)
	}
	resolved, ok := resolvedAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema ref %q is not an object", ref)
	}
	return resolved, nil
}

func extractVariantFields(node map[string]any) []string {
	oneOf, _ := node["oneOf"].([]any)
	var fields []string
	seen := map[string]struct{}{}
	for _, entry := range oneOf {
		obj, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		required, _ := obj["required"].([]any)
		if len(required) != 1 {
			continue
		}
		name, ok := required[0].(string)
		if !ok || name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func jsonCompact(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err == nil {
		return buf.String()
	}
	return string(raw)
}

func normalizeValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return jsonCompact(raw)
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return jsonCompact(raw)
	}
}
