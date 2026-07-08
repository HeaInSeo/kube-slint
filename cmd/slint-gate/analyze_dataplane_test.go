package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane/service"
	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidDataplaneFailOn(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"none", true},
		{"error", true},
		{"warning", true},
		{"", false},
		{"FAIL", false},
		{"errors", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isValidDataplaneFailOn(tc.value), "value=%q", tc.value)
	}
}

func TestShouldFailOnDataplane(t *testing.T) {
	cases := []struct {
		name   string
		s      report.Summary
		failOn string
		want   bool
	}{
		{"none never fails", report.Summary{ErrorCount: 5, WarningCount: 5}, "none", false},
		{"error threshold, errors present", report.Summary{ErrorCount: 1}, "error", true},
		{"error threshold, only warnings", report.Summary{WarningCount: 1}, "error", false},
		{"warning threshold, only warnings", report.Summary{WarningCount: 1}, "warning", true},
		{"warning threshold, errors present", report.Summary{ErrorCount: 1}, "warning", true},
		{"error threshold, clean", report.Summary{}, "error", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shouldFailOnDataplane(tc.s, tc.failOn))
		})
	}
}

// TestResolveDataplaneSeverityThreshold is a regression test for a finding
// from pre-release-adversarial-review (2026-07-08): this subcommand's
// --fail-on collided in name (not meaning) with the gate command's
// deprecated --fail-on/--exit-on pair. --severity-threshold is the new
// preferred name; --fail-on keeps working as a deprecated alias.
// (Structurally similar to TestResolveExitOn in main_test.go — see the
// dupl exemption there.)
//
//nolint:dupl
func TestResolveDataplaneSeverityThreshold(t *testing.T) {
	cases := []struct {
		name                    string
		thresholdSet, failOnSet bool
		thresholdVal, failOnVal string
		wantResolved            string
		wantDeprecated          bool
	}{
		{
			name:         "neither set defaults to error, no deprecation warning",
			thresholdSet: false, thresholdVal: "",
			failOnSet: false, failOnVal: "",
			wantResolved: "error", wantDeprecated: false,
		},
		{
			name:         "only --fail-on set is honored with deprecation warning",
			thresholdSet: false, thresholdVal: "",
			failOnSet: true, failOnVal: "warning",
			wantResolved: "warning", wantDeprecated: true,
		},
		{
			name:         "only --severity-threshold set is honored with no deprecation warning",
			thresholdSet: true, thresholdVal: "none",
			failOnSet: false, failOnVal: "",
			wantResolved: "none", wantDeprecated: false,
		},
		{
			name:         "both set: --severity-threshold wins, no deprecation warning",
			thresholdSet: true, thresholdVal: "error",
			failOnSet: true, failOnVal: "warning",
			wantResolved: "error", wantDeprecated: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, deprecated := resolveDataplaneSeverityThreshold(tc.thresholdSet, tc.thresholdVal, tc.failOnSet, tc.failOnVal)
			assert.Equal(t, tc.wantResolved, resolved)
			assert.Equal(t, tc.wantDeprecated, deprecated)
		})
	}
}

func TestAnalyzeDataplane_EndToEnd_JSONAndSARIFWritten(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "examples", "kind-hello-operator", "manifests")
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "report.json")
	sarifPath := filepath.Join(dir, "report.sarif")

	rep, warnings, err := service.Analyze(fixtureDir, Version)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	require.NoError(t, report.WriteJSON(jsonPath, rep))
	require.NoError(t, report.WriteSARIF(sarifPath, rep))

	jsonData, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	var gotReport report.Report
	require.NoError(t, json.Unmarshal(jsonData, &gotReport))
	assert.Equal(t, 0, gotReport.Summary.ErrorCount)
	assert.Equal(t, 3, gotReport.Summary.WarningCount)

	sarifData, err := os.ReadFile(sarifPath)
	require.NoError(t, err)
	assert.Contains(t, string(sarifData), "sarif-schema-2.1.0")
	assert.Contains(t, string(sarifData), "KSL-DP-004")
}
