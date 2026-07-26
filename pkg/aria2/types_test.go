package aria2

import (
	"encoding/json"
	"testing"
)

func TestOptionsMarshalCumulativeValues(t *testing.T) {
	opts := Options{
		OptDir:    "/tmp",
		OptHeader: "Authorization: Bearer token\nX-Trace: abc123",
	}
	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	headers, ok := got[OptHeader].([]any)
	if !ok || len(headers) != 2 {
		t.Fatalf("header = %#v, want a two-item array", got[OptHeader])
	}
	if got[OptDir] != "/tmp" {
		t.Fatalf("dir = %#v", got[OptDir])
	}
}

func TestOptionsMarshalSingleHeaderAsString(t *testing.T) {
	data, err := json.Marshal(Options{OptHeader: "Accept: application/json"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got[OptHeader] != "Accept: application/json" {
		t.Fatalf("header = %#v", got[OptHeader])
	}
}
