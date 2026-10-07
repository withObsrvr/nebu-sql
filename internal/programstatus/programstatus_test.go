package programstatus

import (
	"bytes"
	"testing"
)

func TestWrite(t *testing.T) {
	var output bytes.Buffer
	if err := Write(&output, Working, "Executing query"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	want := "\x1b]7501;state=working:app=nebu-sql:msg=RXhlY3V0aW5nIHF1ZXJ5\x1b\\"
	if output.String() != want {
		t.Fatalf("Write() = %q, want %q", output.String(), want)
	}
}

func TestSanitizeMessage(t *testing.T) {
	if got := SanitizeMessage("bad\nmessage"); got != "bad message" {
		t.Fatalf("SanitizeMessage() = %q, want bad message", got)
	}
}
