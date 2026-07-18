package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

func TestWriteOut_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := writeOut(&buf, formatJSON, domain.Health{Status: "up", Service: "svc"}); err != nil {
		t.Fatalf("writeOut: %v", err)
	}
	var h domain.Health
	if err := json.Unmarshal(buf.Bytes(), &h); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if h.Status != "up" || h.Service != "svc" {
		t.Errorf("decoded = %+v", h)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		t.Error("JSON output should end with a newline")
	}
}

func TestWriteOut_Compact(t *testing.T) {
	var buf bytes.Buffer
	if err := writeOut(&buf, formatCompact, domain.Health{Status: "up"}); err != nil {
		t.Fatalf("writeOut: %v", err)
	}
	var h domain.Health
	if err := json.Unmarshal(buf.Bytes(), &h); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if h.Status != "up" {
		t.Errorf("decoded = %+v", h)
	}
}

func TestWriteOut_Nil(t *testing.T) {
	var buf bytes.Buffer
	if err := writeOut(&buf, formatJSON, nil); err != nil {
		t.Fatalf("writeOut nil: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("nil output = %q, want empty", buf.String())
	}
}
