package report

import (
	"sort"
	"time"
)

// SchemaVersion is the schema version for Report JSON output.
const SchemaVersion = "slint.dataplane.v1"

// Summary holds aggregate counts over a Report's Findings.
type Summary struct {
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
	RulesRun     int `json:"rules_run"`
}

// Report is the top-level output of a static analysis run.
type Report struct {
	SchemaVersion string    `json:"schema_version"`
	Tool          string    `json:"tool"`
	ToolVersion   string    `json:"tool_version"`
	GeneratedAt   string    `json:"generated_at"`
	SourceDir     string    `json:"source_dir"`
	Rules         []Rule    `json:"rules"`
	Findings      []Finding `json:"findings"`
	Summary       Summary   `json:"summary"`
}

// NewReport builds an empty Report for the given source directory and tool version.
func NewReport(sourceDir, toolVersion string) *Report {
	return &Report{
		SchemaVersion: SchemaVersion,
		Tool:          "kube-slint-dataplane",
		ToolVersion:   toolVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		SourceDir:     sourceDir,
		Rules:         []Rule{},
		Findings:      []Finding{},
	}
}

// Add appends a Finding to the report.
func (r *Report) Add(f Finding) {
	r.Findings = append(r.Findings, f)
}

// Finalize sorts Findings deterministically (by RuleID, then Location) and
// recomputes Summary. Call once after all checks have run.
func (r *Report) Finalize() {
	sort.SliceStable(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		if a.Location.Name != b.Location.Name {
			return a.Location.Name < b.Location.Name
		}
		return a.Location.Container < b.Location.Container
	})

	s := Summary{RulesRun: len(r.Rules)}
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			s.ErrorCount++
		case SeverityWarning:
			s.WarningCount++
		}
	}
	r.Summary = s
}
