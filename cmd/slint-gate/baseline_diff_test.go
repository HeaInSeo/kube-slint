package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeDiffSummary(t *testing.T, dir, name string, values map[string]float64) string {
	t.Helper()
	results := make([]summary.SLIResult, 0, len(values))
	for id, v := range values {
		results = append(results, summary.SLIResult{ID: id, Value: &v, Status: summary.StatusPass})
	}
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       results,
		Reliability:   &summary.Reliability{CollectionStatus: "Complete"},
	}
	path := filepath.Join(dir, name)
	require.NoError(t, summary.WriteFile(path, s))
	return path
}

func writeDiffPolicy(t *testing.T, dir, metric, operator string) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	body := "schema_version: \"slint.policy.v1\"\nthresholds:\n  - name: \"t\"\n    metric: \"" + metric + "\"\n    operator: \"" + operator + "\"\n    value: 0\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestRunBaselineDiff_Identical_OK(t *testing.T) {
	dir := t.TempDir()
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur, "--policy", filepath.Join(dir, "nonexistent-policy.yaml")})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Result:\n  OK")
}

func TestRunBaselineDiff_NewSLI(t *testing.T) {
	dir := t.TempDir()
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5, "webhook_request_5xx_delta": 0})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "webhook_request_5xx_delta")
	// A purely-new SLI (nothing changed or missing) needs no review -- new
	// SLIs are always safe to append, per the append-new-only merge policy.
	assert.Contains(t, out, "Result:\n  OK")
}

func TestRunBaselineDiff_Worsened_LabeledWeakens(t *testing.T) {
	dir := t.TempDir()
	policy := writeDiffPolicy(t, dir, "workqueue_depth_end", "<=")
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"workqueue_depth_end": 0})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"workqueue_depth_end": 3})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "weakens the known-good baseline")
	assert.Contains(t, out, "REVIEW_REQUIRED")
}

func TestRunBaselineDiff_Improved_LabeledImproves(t *testing.T) {
	dir := t.TempDir()
	policy := writeDiffPolicy(t, dir, "workqueue_depth_end", "<=")
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"workqueue_depth_end": 3})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"workqueue_depth_end": 0})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur, "--policy", policy})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "baseline merge --mode review-existing")
}

func TestRunBaselineDiff_MissingFromSummary(t *testing.T) {
	dir := t.TempDir()
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5, "workqueue_depth_end": 0})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Removed or missing SLIs:\n  workqueue_depth_end")
	assert.Contains(t, out, "mark stale, do not delete automatically")
	assert.Contains(t, out, "REVIEW_REQUIRED")
}

func TestRunBaselineDiff_SchemaVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})
	badPath := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(badPath, []byte(`{"schemaVersion":"slo.v99","generatedAt":"2026-01-01T00:00:00Z","results":[]}`), 0o644))

	err := runBaselineDiff([]string{"--baseline", baseline, "--summary", badPath})
	assert.Error(t, err)
}

func TestRunBaselineDiff_MissingPolicy_FallsBackNeutral(t *testing.T) {
	dir := t.TempDir()
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"workqueue_depth_end": 0})
	cur := writeDiffSummary(t, dir, "summary.json", map[string]float64{"workqueue_depth_end": 3})

	var err error
	out := captureStdout(t, func() {
		err = runBaselineDiff([]string{"--baseline", baseline, "--summary", cur, "--policy", filepath.Join(dir, "nonexistent-policy.yaml")})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "is unknown")
	assert.NotContains(t, out, "weakens")
	assert.NotContains(t, out, "improves")
}

func TestRunBaselineDiff_RequiresBaseline(t *testing.T) {
	err := runBaselineDiff([]string{"--summary", "artifacts/sli-summary.json"})
	assert.Error(t, err)
}
