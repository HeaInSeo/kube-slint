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
	assert.Equal(t, 1, gotReport.Summary.ErrorCount)
	assert.Equal(t, 3, gotReport.Summary.WarningCount)

	sarifData, err := os.ReadFile(sarifPath)
	require.NoError(t, err)
	assert.Contains(t, string(sarifData), "sarif-schema-2.1.0")
	assert.Contains(t, string(sarifData), "KSL-DP-003")
}
