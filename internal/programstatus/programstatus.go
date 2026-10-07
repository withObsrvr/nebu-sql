// Package programstatus emits Program Status Protocol (OSC 7501) reports.
package programstatus

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type State string

const (
	Idle    State = "idle"
	Working State = "working"
	Done    State = "done"
	Error   State = "error"
)

func Write(w io.Writer, state State, message string) error {
	if state != Idle && state != Working && state != Done && state != Error {
		return fmt.Errorf("invalid program status state %q", state)
	}
	if len(message) > 2048 {
		return fmt.Errorf("program status message exceeds 2048 bytes")
	}
	for _, r := range message {
		if unicode.IsControl(r) {
			return fmt.Errorf("program status message contains a control character")
		}
	}
	body := "state=" + string(state) + ":app=nebu-sql"
	if message != "" {
		body += ":msg=" + base64.StdEncoding.EncodeToString([]byte(message))
	}
	sequence := "\x1b]7501;" + body + "\x1b\\"
	if len(sequence) > 4096 {
		return fmt.Errorf("program status report exceeds 4096 bytes")
	}
	_, err := io.WriteString(w, sequence)
	return err
}

func SanitizeMessage(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	if len(value) > 2048 {
		return strings.ToValidUTF8(value[:2048], "")
	}
	return value
}
