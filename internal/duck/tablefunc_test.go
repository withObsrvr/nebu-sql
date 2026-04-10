package duck

import (
	"reflect"
	"testing"
)

func TestBuildColumns(t *testing.T) {
	got := buildColumns([]string{"meta", "transfer", "_schema", "event_type", "fee"})
	want := []string{"_schema", "_nebu_version", "event_type", "meta", "transfer", "fee"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildColumns() = %v, want %v", got, want)
	}
}
