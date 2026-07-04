package report

import (
	"encoding/json"
	"os"
)

// WriteJSON writes v as indented JSON to path. On encode failure the
// partially-written file is removed rather than left truncated/corrupt.
func WriteJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}
