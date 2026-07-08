package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBaselineMerge_AppendsNewSLI(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir) // requires reconcile_total_delta >= 1
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "webhook_request_5xx_delta": 0})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "webhook_request_5xx_delta = 0")
	assert.Contains(t, out, "Result:\n  MERGED\n")

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	ids := make([]string, 0, len(merged.Results))
	for _, r := range merged.Results {
		ids = append(ids, r.ID)
	}
	assert.Contains(t, ids, "webhook_request_5xx_delta")
	assert.Contains(t, ids, "reconcile_total_delta")
}

func TestRunBaselineMerge_RejectsWorsenedExisting_BaselineValueUnchanged(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir) // only checks reconcile_total_delta
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 0})
	// workqueue_depth_end worsens (0 -> 3) but the policy doesn't check it, so the
	// overall gate still PASSes -- exercising the append-new-only rejection path.
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 3})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "workqueue_depth_end: current summary has 3, baseline has 0")
	assert.Contains(t, out, "Result:\n  MERGED_WITH_REJECTIONS")

	merged, loadErr := summary.LoadFile(baseline)
	require.NoError(t, loadErr)
	for _, r := range merged.Results {
		if r.ID == "workqueue_depth_end" {
			assert.Equal(t, 0.0, *r.Value, "append-new-only must not overwrite the existing baseline value")
		}
	}
}

func TestRunBaselineMerge_MissingFromSummary_LeftInPlace(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 0})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	require.NoError(t, err)

	merged, loadErr := summary.LoadFile(baseline)
	require.NoError(t, loadErr)
	found := false
	for _, r := range merged.Results {
		if r.ID == "workqueue_depth_end" {
			found = true
		}
	}
	assert.True(t, found, "an SLI missing from the current summary must not be deleted from the baseline")
}

func TestRunBaselineMerge_RejectsWhenSummaryFailsPolicy(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir) // requires reconcile_total_delta >= 1
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 0})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	require.Error(t, err)

	merged, loadErr := summary.LoadFile(baseline)
	require.NoError(t, loadErr)
	assert.Len(t, merged.Results, 1, "baseline must be untouched when the current summary fails its policy")
}

func TestRunBaselineMerge_RequiresExistingBaseline(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	err := runBaselineMerge([]string{"--baseline", filepath.Join(dir, "nonexistent.json"), "--summary", cur, "--policy", policy})
	assert.Error(t, err)
}

func TestRunBaselineMerge_RejectsUnsupportedMode(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "bogus-mode"})
	assert.Error(t, err)
}

func TestRunBaselineMerge_OutputDefaultsToBaseline(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "new_metric_delta": 1})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	require.NoError(t, err)

	data, readErr := os.ReadFile(baseline)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "new_metric_delta")
}

func TestRunBaselineMerge_RefusesDifferentOutputOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})
	out := filepath.Join(dir, "other-baseline.json")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--output", out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Equal(t, "existing content", string(data), "existing file must not be touched without --force")
}

func TestRunBaselineMerge_ForceOverwritesDifferentOutput(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "new_metric_delta": 1})
	out := filepath.Join(dir, "other-baseline.json")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--output", out, "--force"})
	require.NoError(t, err)

	data, readErr := os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "new_metric_delta")
}

// writeMergeReviewPolicy writes a policy with two thresholds: reconcile_total_delta
// (higher-is-better, >=1) and workqueue_depth_end (lower-is-better, <=100 — loose
// enough that any test value in this file satisfies it, so gate.Evaluate PASSes
// regardless of the specific workqueue_depth_end values under test).
func writeMergeReviewPolicy(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	body := `schema_version: "slint.policy.v1"
thresholds:
  - name: "reconcile_min"
    metric: "reconcile_total_delta"
    operator: ">="
    value: 1
  - name: "workqueue_max"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 100
promote_to_fail:
  - "threshold_miss"
`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// Structurally similar to the RejectsRegression/ForceReplace cases below by
// necessity (same arrange/act/assert shape for a merge-mode outcome); not
// worth extracting a shared table-driven helper for 3 cases.
//
//nolint:dupl
func TestRunBaselineMerge_ReviewExisting_UpdatesConfirmedImprovement(t *testing.T) {
	dir := t.TempDir()
	policy := writeMergeReviewPolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 10})
	// workqueue_depth_end is lower-is-better; 10 -> 3 is a confirmed improvement.
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 3})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "review-existing"})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "workqueue_depth_end: 10 → 3")
	assert.Contains(t, out, "Result:\n  MERGED\n")

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	for _, r := range merged.Results {
		if r.ID == "workqueue_depth_end" {
			assert.Equal(t, 3.0, *r.Value, "review-existing should apply a confirmed improvement")
		}
	}
}

//nolint:dupl // see comment on TestRunBaselineMerge_ReviewExisting_UpdatesConfirmedImprovement
func TestRunBaselineMerge_ReviewExisting_RejectsRegression(t *testing.T) {
	dir := t.TempDir()
	policy := writeMergeReviewPolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 10})
	// 10 -> 50 is worse for a lower-is-better metric, even though it still
	// satisfies the (loose) <=100 threshold and the overall gate PASSes.
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 50})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "review-existing"})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "workqueue_depth_end: current summary has 50, baseline has 10")
	assert.Contains(t, out, "Result:\n  MERGED_WITH_REJECTIONS")

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	for _, r := range merged.Results {
		if r.ID == "workqueue_depth_end" {
			assert.Equal(t, 10.0, *r.Value, "review-existing must not apply a regression")
		}
	}
}

func TestRunBaselineMerge_ReviewExisting_RejectsUnknownDirection(t *testing.T) {
	dir := t.TempDir()
	policy := writeMergeReviewPolicy(t, dir)
	// webhook_request_5xx_delta has no threshold rule in the policy, so its
	// improve/weaken direction is unknown - review-existing must not guess.
	// workqueue_depth_end is included unchanged just to satisfy the policy's
	// threshold rule for it so the overall gate PASSes.
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 1, "webhook_request_5xx_delta": 1})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 1, "webhook_request_5xx_delta": 0})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "review-existing"})
	require.NoError(t, err)

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	for _, r := range merged.Results {
		if r.ID == "webhook_request_5xx_delta" {
			assert.Equal(t, 1.0, *r.Value, "an unrecognized direction must never be treated as an improvement")
		}
	}
}

//nolint:dupl // see comment on TestRunBaselineMerge_ReviewExisting_UpdatesConfirmedImprovement
func TestRunBaselineMerge_ForceReplace_OverwritesUnconditionally(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir) // only checks reconcile_total_delta
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 10})
	// 10 -> 999 is unambiguously worse for workqueue_depth_end, but
	// force-replace applies it anyway (no direction check, no gate rule for it).
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 999})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "force-replace"})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "workqueue_depth_end: 10 → 999")
	assert.Contains(t, out, "Result:\n  MERGED\n")

	merged, err := summary.LoadFile(baseline)
	require.NoError(t, err)
	for _, r := range merged.Results {
		if r.ID == "workqueue_depth_end" {
			assert.Equal(t, 999.0, *r.Value, "force-replace must overwrite regardless of direction")
		}
	}
}

func TestRunBaselineMerge_InPlaceMergeNeverNeedsForce(t *testing.T) {
	// Regression guard: the default in-place merge (no --output) must keep
	// working without --force, since overwriting the baseline it just read
	// from is the whole point of merge.
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "new_metric_delta": 1})

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--output", baseline})
	require.NoError(t, err)
}
