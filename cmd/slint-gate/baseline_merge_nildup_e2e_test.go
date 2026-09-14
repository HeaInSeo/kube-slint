package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRawSummary writes a summary with hand-built Results (allowing nil/skipped and
// deliberately-duplicate IDs) to exercise merge behavior end-to-end.
func writeRawSummary(t *testing.T, dir, name string, results []summary.SLIResult, skipped []string) string {
	t.Helper()
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       results,
		Reliability:   &summary.Reliability{CollectionStatus: "Complete", SkippedSLIs: skipped},
	}
	path := filepath.Join(dir, name)
	require.NoError(t, summary.WriteFile(path, s))
	return path
}

// End-to-end write->reload: a baseline holding an SLI as nil/skipped, merged with a
// current numeric for it, must reload with that ID exactly once (issue #18).
func TestRunBaselineMerge_NilBaselineSLI_WriteReload_NoDuplicate(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir) // requires reconcile_total_delta >= 1
	five := 5.0
	baseline := writeRawSummary(t, dir, "baseline.json", []summary.SLIResult{
		{ID: "reconcile_total_delta", Value: &five, Status: summary.StatusPass},
		{ID: "m", Value: nil, Status: summary.StatusSkip},
	}, []string{"m"})
	cur := writeRawSummary(t, dir, "summary.json", []summary.SLIResult{
		{ID: "reconcile_total_delta", Value: &five, Status: summary.StatusPass},
		{ID: "m", Value: &five, Status: summary.StatusPass},
	}, nil)

	var err error
	_ = captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.NoError(t, err)

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	assert.Equal(t, 1, countID(merged, "m"), "append-new-only must not re-append an existing nil/skipped SLI (issue #18)")
	// append-new-only left the skipped record unchanged (honest, still nil + skipped).
	for _, r := range merged.Results {
		if r.ID == "m" {
			assert.Nil(t, r.Value, "append-new-only must not silently fill a nil baseline value")
		}
	}
	assert.True(t, idInSkipped(merged, "m"), "skip marker must be retained")
}

// Fail closed: refuse to merge onto an already-corrupt duplicate-ID baseline rather
// than silently repair it.
func TestRunBaselineMerge_DuplicateIDBaseline_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	five := 5.0
	baseline := writeRawSummary(t, dir, "baseline.json", []summary.SLIResult{
		{ID: "reconcile_total_delta", Value: &five, Status: summary.StatusPass},
		{ID: "reconcile_total_delta", Value: &five, Status: summary.StatusPass}, // pre-existing corrupt duplicate
	}, nil)
	cur := writeRawSummary(t, dir, "summary.json", []summary.SLIResult{
		{ID: "reconcile_total_delta", Value: &five, Status: summary.StatusPass},
	}, nil)

	var err error
	_ = captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.Error(t, err, "a duplicate-ID baseline must fail closed, not be silently merged")
	assert.Contains(t, err.Error(), "reconcile_total_delta")
}
