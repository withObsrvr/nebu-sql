package version

import "testing"

func TestString_UsesInjectedValue(t *testing.T) {
	old := Value
	Value = "v1.2.3"
	defer func() { Value = old }()

	if got := String(); got != "v1.2.3" {
		t.Fatalf("String() = %q, want %q", got, "v1.2.3")
	}
}
