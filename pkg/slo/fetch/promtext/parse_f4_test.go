package promtext

import (
	"strings"
	"testing"
)

// These cover F4: strings.Fields-based splitting used to break on label
// values containing whitespace, since it would split the label block
// itself into multiple tokens and hand a truncated fragment to
// strconv.ParseFloat instead of the real value.

func TestParseTextToMap_LabelValueWithSpace(t *testing.T) {
	raw := `http_request_duration{path="/foo bar",method="GET"} 42` + "\n"

	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	want := `http_request_duration{method="GET",path="/foo bar"}`
	if got[want] != 42 {
		t.Fatalf("expected %q = 42, got map: %v", want, got)
	}
}

func TestParseTextToMap_MultipleLabelValuesWithSpaces(t *testing.T) {
	raw := `metric_a{label="value one"} 1` + "\n" +
		`metric_b{label="value two", other="x y z"} 2` + "\n"

	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 parsed series, got %d: %v", len(got), got)
	}
}

func TestParseTextToMap_LabelValueWithEscapedQuoteAndSpace(t *testing.T) {
	raw := `metric{label="say \"hi\" to me"} 7` + "\n"

	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 parsed series, got %d: %v", len(got), got)
	}
	for _, v := range got {
		if v != 7 {
			t.Fatalf("expected value 7, got %v", v)
		}
	}
}

func TestParseTextToMap_NoLabelsStillWorks(t *testing.T) {
	raw := "bare_metric 3\n"
	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	if got["bare_metric"] != 3 {
		t.Fatalf("expected bare_metric = 3, got %v", got)
	}
}

func TestParseTextToMap_ValueWithTrailingTimestampStillWorks(t *testing.T) {
	raw := `metric{a="b"} 5 1620000000000` + "\n"
	got, err := ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMap returned error: %v", err)
	}
	if got[`metric{a="b"}`] != 5 {
		t.Fatalf("expected metric{a=\"b\"} = 5, got %v", got)
	}
}

func TestSplitMetricLine_UnterminatedQuoteIsSkipped(t *testing.T) {
	// An unterminated label value (no closing '"') should be skipped, not
	// misparsed, same as the pre-F4 behavior for other malformed lines.
	_, _, ok := splitMetricLine(`metric{a="unterminated 1`)
	if ok {
		t.Fatal("expected splitMetricLine to report ok=false for an unterminated quoted value")
	}
}
