package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCIGithubActions_WithBaseline(t *testing.T) {
	var err error
	stdout := captureStdout(t, func() {
		err = runCIGithubActions([]string{
			"--summary", "artifacts/sli-summary.json",
			"--policy", ".slint/policy.yaml",
			"--baseline", "docs/baselines/my-service-sli-summary.json",
		})
	})
	require.NoError(t, err)
	assert.Contains(t, stdout, "baseline: docs/baselines/my-service-sli-summary.json")
	assert.Contains(t, stdout, "measurement-summary: artifacts/sli-summary.json")
	assert.Contains(t, stdout, "policy: .slint/policy.yaml")
	assert.Contains(t, stdout, "exit-on: FAIL_OR_NOGRADE")
}

func TestRunCIGithubActions_WithoutBaseline(t *testing.T) {
	var err error
	stdout := captureStdout(t, func() {
		err = runCIGithubActions(nil)
	})
	require.NoError(t, err)
	assert.NotContains(t, stdout, "baseline:")
}

func TestRunCIGithubActions_CustomActionRef(t *testing.T) {
	var err error
	stdout := captureStdout(t, func() {
		err = runCIGithubActions([]string{"--action-ref", "v9.9.9"})
	})
	require.NoError(t, err)
	assert.Contains(t, stdout, "slint-gate@v9.9.9")
}

func TestRunCIGithubActions_CustomExitOnMode(t *testing.T) {
	var err error
	stdout := captureStdout(t, func() {
		err = runCIGithubActions([]string{"--exit-on-mode", "FAIL_OR_WARN"})
	})
	require.NoError(t, err)
	assert.Contains(t, stdout, "exit-on: FAIL_OR_WARN")
}

func TestRunCIGithubActions_InvalidExitOnMode(t *testing.T) {
	err := runCIGithubActions([]string{"--exit-on-mode", "BOGUS"})
	assert.Error(t, err)
}
