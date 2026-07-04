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

// sarifShape mirrors only the SARIF 2.1.0 fields kube-slint populates. This
// keeps SARIF-shape validation dependency-free (no external schema validator).
type sarifShape struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name  string `json:"name"`
				Rules []struct {
					ID string `json:"id"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []struct {
			RuleID    string `json:"ruleId"`
			Level     string `json:"level"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI string `json:"uri"`
					} `json:"artifactLocation"`
				} `json:"physicalLocation"`
				LogicalLocations []struct {
					FullyQualifiedName string `json:"fullyQualifiedName"`
				} `json:"logicalLocations"`
			} `json:"locations"`
		} `json:"results"`
	} `json:"runs"`
}

func TestWriteSARIF_Shape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.sarif")

	r := report.NewReport("manifests/", "v-test")
	ruleIDs := []string{"KSL-DP-001", "KSL-DP-002", "KSL-DP-003", "KSL-DP-004", "KSL-DP-005", "KSL-DP-006"}
	for _, id := range ruleIDs {
		r.Rules = append(r.Rules, report.Rule{ID: id, Title: id + " title", Description: id + " description"})
	}
	r.Add(report.Finding{
		RuleID:   "KSL-DP-003",
		Severity: report.SeverityError,
		Message:  "no livenessProbe configured",
		Location: report.Location{File: "deployment.yaml", Kind: "Deployment", Namespace: "ns", Name: "app", Container: "app"},
	})
	r.Add(report.Finding{
		RuleID:   "KSL-DP-006",
		Severity: report.SeverityWarning,
		Message:  "terminationGracePeriodSeconds not set",
		Location: report.Location{File: "deployment.yaml", Kind: "Deployment", Namespace: "ns", Name: "app"},
	})
	r.Finalize()

	require.NoError(t, report.WriteSARIF(path, r))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got sarifShape
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "2.1.0", got.Version)
	assert.Contains(t, got.Schema, "sarif-schema-2.1.0")
	require.Len(t, got.Runs, 1)
	assert.Equal(t, "kube-slint-dataplane", got.Runs[0].Tool.Driver.Name)

	gotRuleIDs := make([]string, 0, len(got.Runs[0].Tool.Driver.Rules))
	for _, rule := range got.Runs[0].Tool.Driver.Rules {
		gotRuleIDs = append(gotRuleIDs, rule.ID)
	}
	assert.ElementsMatch(t, ruleIDs, gotRuleIDs)

	require.Len(t, got.Runs[0].Results, 2)
	for _, res := range got.Runs[0].Results {
		assert.Contains(t, []string{"error", "warning"}, res.Level)
		require.Len(t, res.Locations, 1)
		assert.Equal(t, "deployment.yaml", res.Locations[0].PhysicalLocation.ArtifactLocation.URI)
		require.Len(t, res.Locations[0].LogicalLocations, 1)
	}
	assert.Equal(t, "Deployment/ns/app/container/app", got.Runs[0].Results[0].Locations[0].LogicalLocations[0].FullyQualifiedName)
}
