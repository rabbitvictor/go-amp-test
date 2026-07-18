package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// outputFormat enumerates the supported --format values.
type outputFormat string

const (
	formatJSON    outputFormat = "json"    // pretty-printed, 2-space indent
	formatCompact outputFormat = "compact" // no whitespace
)

// writeOut prints v to w in the requested format. nil values print nothing.
func writeOut(w io.Writer, format outputFormat, v any) error {
	if v == nil {
		return nil
	}
	switch format {
	case formatCompact:
		return json.NewEncoder(w).Encode(v)
	case formatJSON:
		fallthrough
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}
		_, err = w.Write(append(b, '\n'))
		return err
	}
}
