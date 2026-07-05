package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeInspectSummary(t *testing.T, dir string, ids []string, collectionStatus string) string {
	t.Helper()
	results := make([]summary.SLIResult, 0, len(ids))
	for _, id := range ids {
		v := 0.0
		results = append(results, summary.SLIResult{ID: id, Value: &v, Status: summary.StatusPass})
	}
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       results,
	}
	if collectionStatus != "" {
		s.Reliability = &summary.Reliability{CollectionStatus: collectionStatus}
	}
	path := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(path, s))
	return path
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, _ := r.Read(tmp)
		if n == 0 {
			break
		}
		buf.Write(tmp[:n])
	}
	return buf.String()
}

func TestRunInspect_FullyMeasured(t *testing.T) {
	dir := t.TempDir()
	path := writeInspectSummary(t, dir, []string{
		"reconcile_total_delta", "reconcile_error_delta", "workqueue_depth_end",
		"rest_client_5xx_delta", "rest_client_429_delta", "workqueue_retries_total_delta",
		"reconcile_success_delta", "workqueue_adds_total_delta", "rest_client_requests_total_delta",
	}, "Complete")

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", path})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "reconcile_total_delta")
	assert.Contains(t, out, "Missing profile SLIs:\n  (none)")
	assert.Contains(t, out, "Threshold policy: ready")
	assert.Contains(t, out, "Baseline approval: ready")
	assert.Contains(t, out, "Measurement confidence: complete")
}

func TestRunInspect_PartiallyMeasured(t *testing.T) {
	dir := t.TempDir()
	path := writeInspectSummary(t, dir, []string{"reconcile_total_delta", "reconcile_error_delta"}, "Partial")

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", path})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "workqueue_depth_end")
	assert.Contains(t, out, "missing metric")
	assert.Contains(t, out, "Recommendation: keep this SLI commented out for now.")
}

func TestRunInspect_EmptySummary(t *testing.T) {
	dir := t.TempDir()
	path := writeInspectSummary(t, dir, nil, "")

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", path})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Threshold policy: not ready")
	assert.Contains(t, out, "Baseline approval: not ready")
}

func TestRunInspect_InformationalTier_MeasuredShowsInformationalWording(t *testing.T) {
	dir := t.TempDir()
	path := writeInspectSummary(t, dir, []string{"reconcile_success_delta"}, "Complete")

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", path})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "reconcile_success_delta")
	assert.Contains(t, out, "measured, informational only (no default threshold)")
}

func TestRunInspect_InvalidSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schemaVersion":"slo.v99","generatedAt":"2026-01-01T00:00:00Z","results":[]}`), 0o644))

	err := runInspect([]string{"--summary", path})
	assert.Error(t, err)
}

func TestRunInspect_MissingFile(t *testing.T) {
	err := runInspect([]string{"--summary", "/nonexistent/summary.json"})
	assert.Error(t, err)
}

func TestRunInspect_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(path, []byte("not json {{{"), 0o644))

	err := runInspect([]string{"--summary", path})
	assert.Error(t, err)
}
