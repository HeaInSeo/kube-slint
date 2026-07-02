package promtext

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseTextToMap_LongMetricLine(t *testing.T) {
	longLabel := strings.Repeat("a", 70*1024)
	raw := fmt.Sprintf("metric_with_long_label{label=%q} 1\n", longLabel)

	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	if got["metric_with_long_label{label=\""+longLabel+"\"}"] != 1 {
		t.Fatalf("long metric line was not parsed")
	}
}
