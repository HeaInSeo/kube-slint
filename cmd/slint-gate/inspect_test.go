package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

func writeInspectPolicy(t *testing.T, dir string, body string) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// captureStdout redirects os.Stdout to a pipe and returns everything fn()
// wrote to it. The read end is drained concurrently in a goroutine started
// before fn() runs (not after), since os.Pipe()'s write end has a bounded
// kernel buffer (commonly 64KiB on Linux, not a portable guarantee) — code
// under test that writes more than that in one fn() call would otherwise
// block on write() forever, because nothing would be reading yet.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = old
	<-done // wait for the drain goroutine to finish before reading buf

	return buf.String()
}

// TestCaptureStdout_DoesNotDeadlockOnOutputLargerThanPipeBuffer reproduces
// the finding from the second pre-release-adversarial-review pass
// (2026-07-08): captureStdout used to drain the pipe only after fn()
// returned, so writing more than the OS pipe's kernel buffer (commonly
// 64KiB on Linux) in one fn() call would block forever. This writes well
// past that with a bounded per-test timeout, so a regression here fails
// the test instead of hanging the whole `go test` run.
func TestCaptureStdout_DoesNotDeadlockOnOutputLargerThanPipeBuffer(t *testing.T) {
	const totalBytes = 512 * 1024 // 512KiB, comfortably past a 64KiB pipe buffer
	const chunk = "0123456789abcdef\n"

	done := make(chan string, 1)
	go func() {
		out := captureStdout(t, func() {
			written := 0
			for written < totalBytes {
				n, _ := os.Stdout.WriteString(chunk)
				written += n
			}
		})
		done <- out
	}()

	select {
	case out := <-done:
		if len(out) < totalBytes {
			t.Fatalf("expected at least %d bytes captured, got %d", totalBytes, len(out))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("captureStdout deadlocked on output larger than the pipe buffer")
	}
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

func TestRunInspect_PolicyCoverageReportsMeasuredButNotCovered(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{
		"reconcile_total_delta",
		"reconcile_slow_delta",
	}, "Complete")
	policyPath := writeInspectPolicy(t, dir, `schema_version: "slint.policy.v1"
thresholds:
  - name: reconcile_min
    metric: reconcile_total_delta
    operator: ">="
    value: 1
regression:
  enabled: false
`)

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", summaryPath, "--policy", policyPath})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Policy coverage:")
	assert.Contains(t, out, "Measured but not covered by policy:")
	assert.Contains(t, out, "reconcile_slow_delta")
	assert.Contains(t, out, "advisory: add a threshold/regression rule or mark informational")
	assert.Contains(t, out, "Policy-covered but missing from summary: (none)")
}

func TestRunInspect_PolicyCoverageReportsPolicyMetricMissingFromSummary(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")
	policyPath := writeInspectPolicy(t, dir, `schema_version: "slint.policy.v1"
thresholds:
  - name: reconcile_min
    metric: reconcile_total_delta
    operator: ">="
    value: 1
  - name: workqueue_max
    metric: workqueue_depth_end
    operator: "<="
    value: 0
regression:
  enabled: true
`)

	var err error
	out := captureStdout(t, func() {
		err = runInspect([]string{"--summary", summaryPath, "--policy", policyPath})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Policy-covered but missing from summary:")
	assert.Contains(t, out, "workqueue_depth_end")
	assert.Contains(t, out, "Regression coverage: enabled")
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
