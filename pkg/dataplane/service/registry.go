// Package service implements the "dataplane-service" static-analysis
// profile: checks over a pkg/dataplane.Bundle covering observability-contract
// concerns for Deployment/StatefulSet/DaemonSet-shaped workloads that are
// NOT already covered by established Kubernetes manifest linters (kube-linter,
// polaris, kubeaudit) — metrics port naming, probe HTTP path convention,
// metrics Service/ServiceMonitor wiring, and terminationGracePeriodSeconds.
//
// Deliberately NOT included: generic "probe missing" and "resource
// requests/limits unset" checks — kube-linter's no-liveness-probe,
// no-readiness-probe, unset-cpu-requirements, and unset-memory-requirements
// already cover that ground well; duplicating them here would just be
// reinventing an existing, actively-maintained tool. Pair kube-slint with
// kube-linter (or similar) for that class of check.
package service

import (
	"fmt"
	"sort"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

// CheckFunc inspects a Bundle and returns the Findings it produces.
type CheckFunc func(b *dataplane.Bundle) []report.Finding

// CheckDef pairs a check's static rule metadata with its behavior.
type CheckDef struct {
	ID          string
	Title       string
	Description string
	HelpURI     string
	Run         CheckFunc
}

// Rule converts a CheckDef to its report.Rule metadata form.
func (c CheckDef) Rule() report.Rule {
	return report.Rule{ID: c.ID, Title: c.Title, Description: c.Description, HelpURI: c.HelpURI}
}

// Registry holds a set of registered checks, keyed by ID.
type Registry struct {
	items map[string]CheckDef
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{items: map[string]CheckDef{}}
}

// Register adds a check to the registry, returning an error if the ID is
// empty or already registered.
func (r *Registry) Register(c CheckDef) error {
	if c.ID == "" {
		return fmt.Errorf("check id is required")
	}
	if _, exists := r.items[c.ID]; exists {
		return fmt.Errorf("check already registered: %s", c.ID)
	}
	r.items[c.ID] = c
	return nil
}

// MustRegister adds a check to the registry, panicking on error.
func (r *Registry) MustRegister(c CheckDef) {
	if err := r.Register(c); err != nil {
		panic(err)
	}
}

// Get looks up a check by ID.
func (r *Registry) Get(id string) (CheckDef, bool) {
	c, ok := r.items[id]
	return c, ok
}

// List returns all registered checks, sorted by ID.
func (r *Registry) List() []CheckDef {
	out := make([]CheckDef, 0, len(r.items))
	for _, c := range r.items {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DefaultRegistry builds a fresh Registry with all dataplane-service checks
// registered. Callers own the returned instance — there is no package-level
// global registry.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.MustRegister(metricsPortCheck())
	r.MustRegister(probePathCheck())
	r.MustRegister(serviceWiringCheck())
	r.MustRegister(gracePeriodCheck())
	return r
}
