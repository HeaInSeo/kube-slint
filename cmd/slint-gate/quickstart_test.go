package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunQuickstart_NothingSetUp_SuggestsInit(t *testing.T) {
	dir := t.TempDir()
	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{
			"--policy", filepath.Join(dir, "policy.yaml"),
			"--summary", filepath.Join(dir, "summary.json"),
		})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate init --profile kubebuilder-operator")
}

func TestRunQuickstart_PolicyPresent_SummaryMissing_SuggestsRunningE2E(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{
			"--policy", policy,
			"--summary", filepath.Join(dir, "summary.json"),
		})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "run your E2E test")
}

func TestRunQuickstart_SummaryInvalid_SuggestsInspect(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(summaryPath, []byte("not json {{{"), 0o644))

	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{"--policy", policy, "--summary", summaryPath})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate inspect --summary "+summaryPath)
}

func TestRunQuickstart_MissingPolicy_SuggestsRecommendPolicy(t *testing.T) {
	dir := t.TempDir()
	summary := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{"--policy", filepath.Join(dir, "policy.yaml"), "--summary", summary})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate recommend-policy")
}

func TestRunQuickstart_Pass_NoBaseline_SuggestsBaselineApprove(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	summary := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})

	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{"--policy", policy, "--summary", summary})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Gate:        PASS")
	assert.Contains(t, out, "slint-gate baseline approve")
}

func TestRunQuickstart_BaselinePresent_SuggestsCISnippet(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	summary := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})
	baseline := writeDiffSummary(t, dir, "baseline.json", map[string]float64{"reconcile_total_delta": 5})

	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{"--policy", policy, "--summary", summary, "--baseline", baseline})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate ci github-actions")
}

func TestRunQuickstart_BaselineGivenButMissing_StillSuggestsApprove_NoCrash(t *testing.T) {
	dir := t.TempDir()
	policy := writeBaselinePolicy(t, dir)
	summary := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})
	baselinePath := filepath.Join(dir, "nonexistent-baseline.json")

	var err error
	out := captureStdout(t, func() {
		err = runQuickstart([]string{"--policy", policy, "--summary", summary, "--baseline", baselinePath})
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Baseline:    ✗ not found")
	assert.Contains(t, out, "slint-gate baseline approve")
}
