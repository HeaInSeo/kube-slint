package report_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
)

func TestReport_Finalize_CountsAndSortsFindings(t *testing.T) {
	r := report.NewReport("manifests/", "v-test")
	r.Rules = []report.Rule{{ID: "KSL-DP-001"}, {ID: "KSL-DP-002"}, {ID: "KSL-DP-003"}}

	r.Add(report.Finding{RuleID: "KSL-DP-002", Severity: report.SeverityWarning, Location: report.Location{Name: "b"}})
	r.Add(report.Finding{RuleID: "KSL-DP-001", Severity: report.SeverityError, Location: report.Location{Name: "a"}})
	r.Add(report.Finding{RuleID: "KSL-DP-001", Severity: report.SeverityError, Location: report.Location{Name: "b"}})

	r.Finalize()

	assert.Equal(t, 2, r.Summary.ErrorCount)
	assert.Equal(t, 1, r.Summary.WarningCount)
	assert.Equal(t, 3, r.Summary.RulesRun)

	require := []string{"KSL-DP-001", "KSL-DP-001", "KSL-DP-002"}
	for i, want := range require {
		assert.Equal(t, want, r.Findings[i].RuleID)
	}
	// within the same RuleID, sorted by Location.Name
	assert.Equal(t, "a", r.Findings[0].Location.Name)
	assert.Equal(t, "b", r.Findings[1].Location.Name)
}

func TestReport_Finalize_NoFindings(t *testing.T) {
	r := report.NewReport("manifests/", "v-test")
	r.Rules = []report.Rule{{ID: "KSL-DP-001"}}
	r.Finalize()

	assert.Equal(t, 0, r.Summary.ErrorCount)
	assert.Equal(t, 0, r.Summary.WarningCount)
	assert.Equal(t, 1, r.Summary.RulesRun)
	assert.Empty(t, r.Findings)
}

func TestNewReport_Defaults(t *testing.T) {
	r := report.NewReport("manifests/", "v1.5.0")
	assert.Equal(t, report.SchemaVersion, r.SchemaVersion)
	assert.Equal(t, "kube-slint-dataplane", r.Tool)
	assert.Equal(t, "v1.5.0", r.ToolVersion)
	assert.Equal(t, "manifests/", r.SourceDir)
	assert.NotEmpty(t, r.GeneratedAt)
	assert.Empty(t, r.Findings)
}
