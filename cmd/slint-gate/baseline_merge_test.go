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

	err := runBaselineMerge([]string{"--baseline", baseline, "--summary", cur, "--policy", policy, "--mode", "force-replace"})
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
