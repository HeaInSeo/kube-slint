package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRecommendPolicy_FullyMeasured_AllActive(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{
		"reconcile_total_delta", "reconcile_error_delta", "workqueue_depth_end",
		"rest_client_5xx_delta", "rest_client_429_delta", "workqueue_retries_total_delta",
	}, "Complete")
	out := filepath.Join(dir, "policy.yaml")

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--output", out})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	body := string(data)
	for _, id := range []string{
		"reconcile_total_delta", "reconcile_error_delta", "workqueue_depth_end",
		"rest_client_5xx_delta", "rest_client_429_delta", "workqueue_retries_total_delta",
	} {
		assert.Contains(t, body, `metric: "`+id+`"`)
	}
	assert.Contains(t, body, "promote_to_fail:")
	// The 3 informational-tier candidates are always commented out (by
	// design, regardless of measurement), so "Not promoted" still appears --
	// but none of the 6 gateable candidates above should be inside it.
	assert.Contains(t, body, "Not promoted to an active rule")
}

func TestRunRecommendPolicy_PartiallyMeasured_MissingCommented(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta", "reconcile_error_delta"}, "Partial")
	out := filepath.Join(dir, "policy.yaml")

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--output", out})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, `metric: "reconcile_total_delta"`)
	assert.NotContains(t, body, `metric: "workqueue_depth_end"`)
	assert.Contains(t, body, "# - workqueue_depth_end")
}

func TestRunRecommendPolicy_UnknownProfile(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--profile", "bogus", "--dry-run"})
	assert.Error(t, err)
}

func TestRunRecommendPolicy_UnknownStrictness(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--strictness", "overkill", "--dry-run"})
	assert.Error(t, err)
}

func TestRunRecommendPolicy_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")
	out := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--output", out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Equal(t, "existing content", string(data), "existing file must not be touched without --force")
}

func TestRunRecommendPolicy_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")
	out := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runRecommendPolicy([]string{"--summary", summaryPath, "--output", out, "--force"})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "schema_version")
}

func TestRunRecommendPolicy_DryRun_NoFileWritten(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_total_delta"}, "Complete")
	out := filepath.Join(dir, "policy.yaml")

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--output", out, "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, "schema_version")
	assert.NoFileExists(t, out)
}

func TestRunRecommendPolicy_Strict_NoisyHasNoRelaxNote(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"rest_client_429_delta"}, "Complete")

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--strictness", "strict", "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, `metric: "rest_client_429_delta"`)
	assert.NotContains(t, stdout, "CI-environment sensitive")
}

func TestRunRecommendPolicy_Conservative_NoisyHasRelaxNote(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"rest_client_429_delta"}, "Complete")

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--strictness", "conservative", "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, `metric: "rest_client_429_delta"`)
	assert.Contains(t, stdout, "CI-environment sensitive")
}

func TestRunRecommendPolicy_Informational_NeverActive_AnyStrictness(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_success_delta"}, "Complete")

	for _, s := range []string{"strict", "conservative", "lenient"} {
		var capturedErr error
		stdout := captureStdout(t, func() {
			capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--strictness", s, "--dry-run"})
		})
		require.NoError(t, capturedErr)
		assert.NotContains(t, stdout, `metric: "reconcile_success_delta"`, "strictness=%s", s)
		assert.Contains(t, stdout, "no default threshold is recommended", "strictness=%s", s)
	}
}

func TestRunRecommendPolicy_EmitsStrictCoverageInformationalDefaults(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"reconcile_success_delta"}, "Complete")

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, "coverage:")
	assert.Contains(t, stdout, "  required: true")
	assert.Contains(t, stdout, `    - "reconcile_success_delta"`)
	assert.Contains(t, stdout, `    - "workqueue_adds_total_delta"`)
	assert.Contains(t, stdout, `    - "rest_client_requests_total_delta"`)
	assert.Contains(t, stdout, `  - "coverage_gap"`)
}

func TestRunRecommendPolicy_ProfileFile_DrivesOutput(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"custom_metric_delta"}, "Complete")
	profilePath := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(`schema_version: "slint.profile.v1"
name: "custom"
candidates:
  - id: "custom_metric_delta"
    operator: "=="
    value: 0
    tier: "core"
    reason: "custom candidate"
`), 0o644))

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--profile-file", profilePath, "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, `metric: "custom_metric_delta"`)
	assert.Contains(t, stdout, "# Profile:      custom")
}

func TestRunRecommendPolicy_MismatchedValue_ShowsWarning(t *testing.T) {
	dir := t.TempDir()
	// workqueue_depth_end's default rule is <= 0; measuring 3 violates it.
	summaryPath := writeDiffSummary(t, dir, "summary.json", map[string]float64{"workqueue_depth_end": 3})

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.Contains(t, stdout, "⚠ measured value (3) does not satisfy this default threshold")
}

func TestRunRecommendPolicy_MatchingValue_NoWarning(t *testing.T) {
	dir := t.TempDir()
	// workqueue_depth_end's default rule is <= 0; measuring 0 satisfies it.
	summaryPath := writeDiffSummary(t, dir, "summary.json", map[string]float64{"workqueue_depth_end": 0})

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.NotContains(t, stdout, "⚠")
}

func TestRunRecommendPolicy_Lenient_NoisyIsCommentedOut(t *testing.T) {
	dir := t.TempDir()
	summaryPath := writeInspectSummary(t, dir, []string{"rest_client_429_delta"}, "Complete")

	var capturedErr error
	stdout := captureStdout(t, func() {
		capturedErr = runRecommendPolicy([]string{"--summary", summaryPath, "--strictness", "lenient", "--dry-run"})
	})
	require.NoError(t, capturedErr)
	assert.NotContains(t, stdout, `metric: "rest_client_429_delta"`)
	assert.Contains(t, stdout, "# - rest_client_429_delta")
}
