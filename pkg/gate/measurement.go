package gate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

func loadMeasurement(path string) (*summary.Summary, string) {
	if path == "" {
		return nil, measMissing
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, measMissing
		}
		// Same reasoning as loadPolicy: an OS/IO failure here is not the
		// same thing as "corrupt content," and diagnose.go's hints for
		// MEASUREMENT_INPUT_CORRUPT only make sense for the latter. There's
		// no warnings-list return path here (unlike loadPolicy), so surface
		// the real error directly.
		fmt.Fprintf(os.Stderr, "slint-gate: could not read %s: %v\n", path, err)
		return nil, measCorrupt
	}
	var s summary.Summary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, measCorrupt
	}
	if err := summary.ValidateSchemaVersion(s); err != nil {
		return nil, measUnsupportedSchema
	}
	// summary.Validate additionally rejects empty/duplicate result IDs, an
	// unrecognized result status, and a missing/zero generatedAt — none of
	// which ValidateSchemaVersion alone catches. Any such artifact is
	// untrustworthy, same as malformed JSON, so it maps to the same
	// measCorrupt/MEASUREMENT_INPUT_CORRUPT outcome.
	if err := summary.Validate(s); err != nil {
		return nil, measCorrupt
	}
	return &s, measOK
}

func resultValueMap(s *summary.Summary) map[string]float64 {
	if s == nil {
		return map[string]float64{}
	}
	return s.ResultValues()
}
