package summary_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

func TestResultValues_OmitsNilValues(t *testing.T) {
	a, b := 1.5, 2.5
	s := summary.Summary{
		Results: []summary.SLIResult{
			{ID: "measured_a", Value: &a},
			{ID: "measured_b", Value: &b},
			{ID: "skipped_no_value", Value: nil},
		},
	}

	got := s.ResultValues()

	if len(got) != 2 {
		t.Fatalf("expected 2 entries (nil-value result omitted), got %d: %v", len(got), got)
	}
	if got["measured_a"] != 1.5 || got["measured_b"] != 2.5 {
		t.Fatalf("unexpected values: %v", got)
	}
	if _, ok := got["skipped_no_value"]; ok {
		t.Fatal("skipped_no_value should not appear in ResultValues()")
	}
}

func TestResultValues_EmptySummary(t *testing.T) {
	got := summary.Summary{}.ResultValues()
	if len(got) != 0 {
		t.Fatalf("expected empty map for a Summary with no results, got %v", got)
	}
}
