package cli

import (
	"strings"
	"testing"
)

// FuzzResolveName fuzzes the --data JSON parsing path in resolveName. The
// data == "-" stdin branch is skipped via t.Skip to keep fuzzing hermetic
// (no stdin reads). The function must never return both a name and an error,
// and must never panic.
func FuzzResolveName(f *testing.F) {
	cases := []string{
		`{"name":"from data"}`, `{"name":""}`, `{}`, `null`, `42`,
		`{not json}`, `{"name":null}`, `{"name":"\u0000"}`,
		`{"name":"` + strings.Repeat("a", 10000) + `"}`,
		`{"name":"\xff\xfe\xfd"}`, `{"name":"unicode \u00e9"}`,
		`{"extra":"field","name":"ok"}`,
		"", "  ", `[]"name":"x"`,
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, data string) {
		if data == "-" {
			t.Skip("stdin branch is non-hermetic")
		}
		name, err := resolveName("", data)
		if err != nil && name != "" {
			t.Fatalf("resolveName returned both name %q and error %v", name, err)
		}
	})
}
