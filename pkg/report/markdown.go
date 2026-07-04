package report

import (
	"fmt"
	"os"
	"strings"
)

// mdCell is a small, private copy of cmd/slint-gate's markdown-table cell
// escaping helper. pkg/report cannot import cmd/slint-gate (that would
// invert the module's dependency direction), so this 4-line duplication is
// intentional rather than an oversight.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// RenderMarkdownTable renders r as a GitHub-flavored Markdown report.
func RenderMarkdownTable(r *Report) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# kube-slint dataplane-service Report\n\n")
	fmt.Fprintf(&sb, "- **Source:** %s\n", r.SourceDir)
	fmt.Fprintf(&sb, "- **Errors:** %d · **Warnings:** %d · **Rules run:** %d\n\n",
		r.Summary.ErrorCount, r.Summary.WarningCount, r.Summary.RulesRun)

	if len(r.Findings) == 0 {
		fmt.Fprintf(&sb, "No findings.\n")
		return sb.String()
	}

	fmt.Fprintf(&sb, "| Rule | Severity | Object | Container | File | Message |\n")
	fmt.Fprintf(&sb, "|---|---|---|---|---|---|\n")
	for _, f := range r.Findings {
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s |\n",
			mdCell(f.RuleID),
			mdCell(string(f.Severity)),
			mdCell(fmt.Sprintf("%s/%s/%s", f.Location.Kind, f.Location.Namespace, f.Location.Name)),
			mdCell(f.Location.Container),
			mdCell(f.Location.File),
			mdCell(f.Message),
		)
	}
	return sb.String()
}

// WriteGitHubStepSummary appends RenderMarkdownTable(r) to $GITHUB_STEP_SUMMARY.
// A no-op when that env var is unset.
func WriteGitHubStepSummary(r *Report) error {
	sumPath := os.Getenv("GITHUB_STEP_SUMMARY")
	if sumPath == "" {
		return nil
	}

	f, err := os.OpenFile(sumPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(RenderMarkdownTable(r))
	if err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
