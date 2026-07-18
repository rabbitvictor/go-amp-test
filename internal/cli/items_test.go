package cli

import "testing"

func TestResolveName_NameFlag(t *testing.T) {
	got, err := resolveName("my item", "")
	if err != nil {
		t.Fatalf("resolveName: %v", err)
	}
	if got != "my item" {
		t.Errorf("resolveName = %q, want %q", got, "my item")
	}
}

func TestResolveName_DataJSON(t *testing.T) {
	got, err := resolveName("", `{"name":"from data"}`)
	if err != nil {
		t.Fatalf("resolveName: %v", err)
	}
	if got != "from data" {
		t.Errorf("resolveName = %q, want %q", got, "from data")
	}
}

func TestResolveName_BadJSON(t *testing.T) {
	_, err := resolveName("", `{not json}`)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
