package report_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdownTable_NoFindings(t *testing.T) {
	r := report.NewReport("manifests/", "v-test")
	r.Finalize()
	out := report.RenderMarkdownTable(r)
	assert.Contains(t, out, "No findings.")
}

func TestRenderMarkdownTable_WithFindings(t *testing.T) {
	r := report.NewReport("manifests/", "v-test")
	r.Add(report.Finding{
		RuleID: "KSL-DP-003", Severity: report.SeverityError, Message: "no livenessProbe | pipe\nnewline",
		Location: report.Location{File: "deployment.yaml", Kind: "Deployment", Namespace: "ns", Name: "app", Container: "app"},
	})
	r.Finalize()

	out := report.RenderMarkdownTable(r)
	assert.Contains(t, out, "KSL-DP-003")
	assert.Contains(t, out, "error")
	assert.Contains(t, out, "Deployment/ns/app")
	assert.Contains(t, out, "no livenessProbe \\| pipe newline") // escaped pipe, collapsed newline
	assert.NotContains(t, out, "\nnewline\n|")
}

func TestWriteGitHubStepSummary_NoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	r := report.NewReport("manifests/", "v-test")
	r.Finalize()
	assert.NoError(t, report.WriteGitHubStepSummary(r))
}

func TestWriteGitHubStepSummary_WritesAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.md")
	require.NoError(t, os.WriteFile(path, []byte("# existing\n"), 0o644))
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	r := report.NewReport("manifests/", "v-test")
	r.Add(report.Finding{RuleID: "KSL-DP-001", Severity: report.SeverityError, Message: "no metrics port"})
	r.Finalize()

	require.NoError(t, report.WriteGitHubStepSummary(r))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# existing")
	assert.Contains(t, string(data), "kube-slint dataplane-service Report")
	assert.Contains(t, string(data), "KSL-DP-001")
}
