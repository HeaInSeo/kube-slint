package main

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// countID returns how many times an SLI ID appears in a summary's Results.
func countID(s summary.Summary, id string) int {
	n := 0
	for _, r := range s.Results {
		if r.ID == id {
			n++
		}
	}
	return n
}

func idInSkipped(s summary.Summary, id string) bool {
	if s.Reliability == nil {
		return false
	}
	for _, x := range s.Reliability.SkippedSLIs {
		if x == id {
			return true
		}
	}
	return false
}

// Issue #18 reproduction: a baseline that HOLDS an SLI as nil/skipped must not be
// re-appended as "new" when the current summary supplies a numeric value for it —
// that produces a duplicate-ID baseline. Before the fix this fails (m appears twice).
func TestMerge_NilBaselineSLI_NotDuplicated_AppendNewOnly(t *testing.T) {
	nilBaseline := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		Results:       []summary.SLIResult{{ID: "m", Value: nil, Status: summary.StatusSkip}},
		Reliability:   &summary.Reliability{CollectionStatus: "Complete", SkippedSLIs: []string{"m"}},
	}
	v := 5.0
	cur := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		Results:       []summary.SLIResult{{ID: "m", Value: &v, Status: summary.StatusPass}},
		Reliability:   &summary.Reliability{CollectionStatus: "Complete"},
	}
	appended, updated, _ := computeMergePlan("append-new-only", nilBaseline, cur, map[string]string{})
	applyMergePlan(&nilBaseline, appended, updated, cur, "append-new-only")
	if got := countID(nilBaseline, "m"); got != 1 {
		t.Fatalf("SLI m must appear exactly once after merge; got %d (duplicate-ID baseline, issue #18)", got)
	}
}

func fptr(v float64) *float64 { return &v }

func mSkippedBaseline() summary.Summary {
	return summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		Results:       []summary.SLIResult{{ID: "m", Value: nil, Status: summary.StatusSkip}, {ID: "keep", Value: fptr(3), Status: summary.StatusPass}},
		Reliability:   &summary.Reliability{CollectionStatus: "Complete", SkippedSLIs: []string{"m"}},
	}
}

func mNumericCurrent() summary.Summary {
	return summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		Results:       []summary.SLIResult{{ID: "m", Value: fptr(5), Status: summary.StatusPass}, {ID: "keep", Value: fptr(3), Status: summary.StatusPass}},
		Reliability:   &summary.Reliability{CollectionStatus: "Complete"},
	}
}

// force-replace: an existing nil/skipped SLI supplied a current numeric is replaced
// IN PLACE (one record), and its stale skip marker is reconciled away.
func TestMerge_NilBaselineSLI_ForceReplace_ReplacesInPlace(t *testing.T) {
	base, cur := mSkippedBaseline(), mNumericCurrent()
	appended, updated, rejected := computeMergePlan("force-replace", base, cur, map[string]string{})
	applyMergePlan(&base, appended, updated, cur, "force-replace")
	if got := countID(base, "m"); got != 1 {
		t.Fatalf("m must appear once; got %d", got)
	}
	for _, r := range base.Results {
		if r.ID == "m" && (r.Value == nil || *r.Value != 5) {
			t.Fatalf("force-replace must replace nil m with current numeric 5; got %v", r.Value)
		}
	}
	if idInSkipped(base, "m") {
		t.Fatalf("force-replace must drop the stale skip marker for a replaced record")
	}
	if idInSkipped(base, "keep") {
		t.Fatalf("a numeric, never-skipped SLI must not appear in the skipped set")
	}
	if len(rejected) != 0 {
		t.Fatalf("no rejections expected; got %v", rejected)
	}
}

// review-existing: a nil old value cannot prove a numeric improvement — reject, never
// guess or silently fill; the skip marker is retained.
func TestMerge_NilBaselineSLI_ReviewExisting_RejectsNoGuess(t *testing.T) {
	base, cur := mSkippedBaseline(), mNumericCurrent()
	appended, updated, rejected := computeMergePlan("review-existing", base, cur, map[string]string{"m": "lower"})
	applyMergePlan(&base, appended, updated, cur, "review-existing")
	if got := countID(base, "m"); got != 1 {
		t.Fatalf("m must appear once; got %d", got)
	}
	for _, r := range base.Results {
		if r.ID == "m" && r.Value != nil {
			t.Fatalf("review-existing must NOT fill a nil baseline; got %v", *r.Value)
		}
	}
	if !idInSkipped(base, "m") {
		t.Fatalf("skip marker must be retained when the record was not replaced")
	}
	if len(updated) != 0 {
		t.Fatalf("no updates expected; got %v", updated)
	}
	if len(rejected) == 0 {
		t.Fatalf("review-existing must report the un-mergeable nil-baseline change as rejected")
	}
}

// append-new-only: an existing nil/skipped SLI is not new and is left untouched.
func TestMerge_NilBaselineSLI_AppendNewOnly_LeavesSkipped(t *testing.T) {
	base, cur := mSkippedBaseline(), mNumericCurrent()
	appended, updated, rejected := computeMergePlan("append-new-only", base, cur, map[string]string{})
	applyMergePlan(&base, appended, updated, cur, "append-new-only")
	if got := countID(base, "m"); got != 1 {
		t.Fatalf("m must appear once; got %d", got)
	}
	for _, r := range base.Results {
		if r.ID == "m" && r.Value != nil {
			t.Fatalf("append-new-only must not fill a nil baseline; got %v", *r.Value)
		}
	}
	if !idInSkipped(base, "m") {
		t.Fatalf("skip marker must be retained")
	}
	if len(updated) != 0 || len(rejected) != 0 {
		t.Fatalf("append-new-only leaves the skipped record untouched; updated=%v rejected=%v", updated, rejected)
	}
}

// A genuinely new ID still appends in every mode.
func TestMerge_GenuinelyNewID_StillAppends(t *testing.T) {
	for _, mode := range []string{"append-new-only", "review-existing", "force-replace"} {
		base := summary.Summary{SchemaVersion: summary.SchemaVersion, Results: []summary.SLIResult{{ID: "keep", Value: fptr(3), Status: summary.StatusPass}}, Reliability: &summary.Reliability{CollectionStatus: "Complete"}}
		cur := summary.Summary{SchemaVersion: summary.SchemaVersion, Results: []summary.SLIResult{{ID: "keep", Value: fptr(3), Status: summary.StatusPass}, {ID: "newid", Value: fptr(7), Status: summary.StatusPass}}, Reliability: &summary.Reliability{CollectionStatus: "Complete"}}
		appended, updated, _ := computeMergePlan(mode, base, cur, map[string]string{})
		applyMergePlan(&base, appended, updated, cur, mode)
		if countID(base, "newid") != 1 {
			t.Fatalf("mode %s: genuinely new id must append exactly once; got %d", mode, countID(base, "newid"))
		}
	}
}

// force-replace applied twice is idempotent: no duplicate ID accrues on repeat.
func TestMerge_NilBaselineSLI_ForceReplace_Idempotent(t *testing.T) {
	base, cur := mSkippedBaseline(), mNumericCurrent()
	appended, updated, _ := computeMergePlan("force-replace", base, cur, map[string]string{})
	applyMergePlan(&base, appended, updated, cur, "force-replace")
	appended2, updated2, _ := computeMergePlan("force-replace", base, cur, map[string]string{})
	applyMergePlan(&base, appended2, updated2, cur, "force-replace")
	if got := countID(base, "m"); got != 1 {
		t.Fatalf("repeat merge must stay idempotent; m count = %d", got)
	}
	if len(appended2) != 0 {
		t.Fatalf("repeat merge must append nothing; got %v", appended2)
	}
}
