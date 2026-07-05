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
	assert.NotContains(t, body, "Not yet promoted")
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
