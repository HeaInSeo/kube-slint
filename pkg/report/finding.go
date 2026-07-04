// Package report provides a generic, Kubernetes-agnostic finding/report model
// (rule ID, severity, message, location) and output writers (JSON, SARIF,
// GitHub Actions step summary). It has no knowledge of any specific profile
// (dataplane-service, dataplane-job, ...) — those live in their own packages
// and produce []Finding for this package to render.
package report

import "strings"

// Severity is the level of a Finding.
type Severity string

const (
	// SeverityError indicates a hard contract violation.
	SeverityError Severity = "error"
	// SeverityWarning indicates a non-blocking recommendation.
	SeverityWarning Severity = "warning"
)

// Location identifies where a Finding applies: the source file plus the
// Kubernetes object (and, optionally, container) it was found in.
//
// Note: Location deliberately does not carry a YAML line number. Precise
// per-field line tracking would require plumbing yaml.Node position data
// through every check; File+Kind+Namespace+Name(+Container) is enough to
// locate the object by hand. Revisit if line-level annotations are requested.
type Location struct {
	File      string `json:"file"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Container string `json:"container,omitempty"`
}

// Finding is one rule violation/warning produced by a static check.
type Finding struct {
	RuleID      string   `json:"rule_id"`
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation,omitempty"`
	Location    Location `json:"location"`
}

// FullyQualifiedName renders the Location as a single identity string, e.g.
// "Deployment/hello-system/hello-operator" or, when Container is set,
// "Deployment/hello-system/hello-operator/container/hello-operator". Used as
// SARIF's logicalLocations.fullyQualifiedName, since SARIF's physicalLocation
// has no notion of Kubernetes object identity and Location carries no line
// number to anchor a precise region.
func (l Location) FullyQualifiedName() string {
	parts := []string{l.Kind, l.Namespace, l.Name}
	if l.Container != "" {
		parts = append(parts, "container", l.Container)
	}
	return strings.Join(parts, "/")
}

// Rule is static metadata about a check, independent of whether it produced
// a Finding in a given run. SARIF requires the full rule catalog in
// driver.rules[], not just the rules that fired.
type Rule struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	HelpURI     string `json:"help_uri,omitempty"`
}
