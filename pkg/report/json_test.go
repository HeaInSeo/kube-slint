package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	r := report.NewReport("manifests/", "v-test")
	r.Rules = []report.Rule{{ID: "KSL-DP-001", Title: "t"}}
	r.Add(report.Finding{RuleID: "KSL-DP-001", Severity: report.SeverityError, Message: "m"})
	r.Finalize()

	require.NoError(t, report.WriteJSON(path, r))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got report.Report
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, r.SchemaVersion, got.SchemaVersion)
	assert.Len(t, got.Findings, 1)
	assert.Equal(t, "KSL-DP-001", got.Findings[0].RuleID)
}

func TestWriteJSON_NoHTMLEscape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	r := report.NewReport("manifests/", "v-test")
	r.Add(report.Finding{RuleID: "KSL-DP-001", Message: "a < b && c > d"})
	r.Finalize()

	require.NoError(t, report.WriteJSON(path, r))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "a < b && c > d")
}

func TestWriteJSON_InvalidPath(t *testing.T) {
	err := report.WriteJSON(filepath.Join(t.TempDir(), "nonexistent-dir", "report.json"), report.NewReport("x", "v"))
	assert.Error(t, err)
}
