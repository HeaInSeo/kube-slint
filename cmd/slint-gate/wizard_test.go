package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWizard_RejectsNonInteractiveStdin(t *testing.T) {
	// The test process's stdin is not a TTY (it's whatever `go test` wired
	// up - a pipe or /dev/null), so this exercises the real guard, not a
	// fake. This is the concrete fix for the risk the wizard was previously
	// deferred over: it must never block forever waiting on input that will
	// never arrive under CI/piped invocation.
	dir := t.TempDir()
	err := runWizard([]string{
		"--policy", filepath.Join(dir, "policy.yaml"),
		"--summary", filepath.Join(dir, "summary.json"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TTY")
	assert.Contains(t, err.Error(), "quickstart")
}

func TestRunWizardLoop_NothingSetUp_RunsInit_ThenStopsAtE2E(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, ".slint", "policy.yaml")
	summaryPath := filepath.Join(dir, "artifacts", "sli-summary.json")

	// confirm init? (blank -> default yes), namespace (skip), service (skip)
	stdin := strings.NewReader("\n\n\n")

	var err error
	out := captureStdout(t, func() {
		err = runWizardLoop(stdin, policyPath, summaryPath, "")
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate init --profile kubebuilder-operator")
	assert.Contains(t, out, "run your E2E test")

	_, statErr := os.Stat(policyPath)
	assert.NoError(t, statErr, "wizard should have written the policy file via runInit")
}

func TestRunWizardLoop_UserDeclines_StopsWithoutActing(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.yaml")
	summaryPath := filepath.Join(dir, "summary.json")

	stdin := strings.NewReader("n\n")

	var err error
	out := captureStdout(t, func() {
		err = runWizardLoop(stdin, policyPath, summaryPath, "")
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate init")

	_, statErr := os.Stat(policyPath)
	assert.True(t, os.IsNotExist(statErr), "declining the prompt must not run init")
}

func TestRunWizardLoop_ApproveBaseline_ThenWiresCI(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir) // reconcile_total_delta >= 1
	summaryPath := writeDiffSummary(t, dir, "summary.json", map[string]float64{"reconcile_total_delta": 5})
	baselineOut := filepath.Join(dir, "baseline.json")

	// stepApproveBaseline: confirm "y", then the baseline output path.
	// stepWireCI (next loop iteration): confirm "y" to print the snippet.
	stdin := strings.NewReader("y\n" + baselineOut + "\ny\n")

	var err error
	out := captureStdout(t, func() {
		err = runWizardLoop(stdin, policyPath, summaryPath, "")
	})
	require.NoError(t, err)
	assert.Contains(t, out, "Approved.")
	assert.Contains(t, out, "uses: HeaInSeo/kube-slint/.github/actions/slint-gate")
	assert.Contains(t, out, "Onboarding loop complete.")

	_, statErr := os.Stat(baselineOut)
	assert.NoError(t, statErr, "wizard should have written the approved baseline")
}

func TestRunWizardLoop_InvalidSummary_RunsInspectThenStops(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeBaselinePolicy(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(summaryPath, []byte("not json {{{"), 0o644))

	stdin := strings.NewReader("y\n")

	var err error
	out := captureStdout(t, func() {
		err = runWizardLoop(stdin, policyPath, summaryPath, "")
	})
	require.NoError(t, err)
	assert.Contains(t, out, "slint-gate inspect --summary "+summaryPath)
}
