package service

import (
	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

// Analyze loads Kubernetes manifests from dir and runs all default
// dataplane-service checks against them, returning a finalized Report and
// any non-fatal load warnings encountered along the way.
func Analyze(dir, toolVersion string) (*report.Report, []dataplane.LoadWarning, error) {
	bundle, warnings, err := dataplane.LoadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	rep := report.NewReport(dir, toolVersion)
	reg := DefaultRegistry()
	for _, c := range reg.List() {
		rep.Rules = append(rep.Rules, c.Rule())
		for _, f := range c.Run(bundle) {
			rep.Add(f)
		}
	}
	rep.Finalize()

	return rep, warnings, nil
}
