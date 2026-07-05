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

const minimalPassPolicy = `schema_version: "slint.policy.v1"
thresholds:
  - name: "reconcile_min"
    metric: "reconcile_total_delta"
    operator: ">="
    value: 1
promote_to_fail:
  - "threshold_miss"
`

func writeBaselinePolicy(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimalPassPolicy), 0o644))
	return path
}

func TestRunBaselineApprove_PassIsApproved(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)

	v := 5.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Config:        summary.RunConfig{EvidencePaths: map[string]string{"logs": "/tmp/local/logs.txt"}},
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	out := filepath.Join(dir, "baseline.json")
	var err error
	stdout := captureStdout(t, func() {
		err = runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out})
	})
	require.NoError(t, err)
	assert.Contains(t, stdout, "Approved.")
	assert.FileExists(t, out)

	written, err := summary.LoadFile(out)
	require.NoError(t, err)
	assert.Nil(t, written.Config.EvidencePaths, "EvidencePaths must be cleared from the written baseline")
}

func TestRunBaselineApprove_FailIsRejected(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)

	v := 0.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	out := filepath.Join(dir, "baseline.json")
	err := runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out})
	require.Error(t, err)
	assert.NoFileExists(t, out)
}

func TestRunBaselineApprove_Fail_ForceDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)

	v := 0.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	out := filepath.Join(dir, "baseline.json")
	err := runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out, "--force"})
	require.Error(t, err, "--force must not launder a FAIL result into an approved baseline")
	assert.NoFileExists(t, out)
}

func TestRunBaselineApprove_NoGradeIsRejected(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)
	out := filepath.Join(dir, "baseline.json")

	err := runBaselineApprove([]string{"--summary", filepath.Join(dir, "nonexistent.json"), "--policy", policyPath, "--output", out})
	require.Error(t, err)
	assert.NoFileExists(t, out)
}

func TestRunBaselineApprove_WarnRequiresAllowFlag(t *testing.T) {
	dir := t.TempDir()
	// No thresholds/promote_to_fail at all -> regression enabled with no baseline
	// produces a first-run WARN (BASELINE_ABSENT_FIRST_RUN), never PASS/FAIL.
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(`schema_version: "slint.policy.v1"
regression:
  enabled: true
  tolerance_percent: 5
`), 0o644))

	v := 1.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	out := filepath.Join(dir, "baseline.json")
	err := runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out})
	require.Error(t, err, "WARN must not be approved without --allow-warn")
	assert.NoFileExists(t, out)

	err = runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out, "--allow-warn"})
	require.NoError(t, err)
	assert.FileExists(t, out)
}

func TestRunBaselineApprove_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)

	v := 5.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	out := filepath.Join(dir, "baseline.json")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Equal(t, "existing content", string(data))

	err = runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath, "--output", out, "--force"})
	require.NoError(t, err)
	data, readErr = os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "schemaVersion")
}

func TestRunBaselineApprove_RequiresOutput(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)

	v := 5.0
	s := summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results:       []summary.SLIResult{{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass}},
	}
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, summary.WriteFile(summaryPath, s))

	err := runBaselineApprove([]string{"--summary", summaryPath, "--policy", policyPath})
	assert.Error(t, err)
}
