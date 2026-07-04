package service

import (
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

// findMetricsPort returns the first container port named "metrics"
// (case-insensitive) anywhere in the workload's pod template, if any.
func findMetricsPort(w dataplane.Workload) (dataplane.ContainerPort, bool) {
	for _, c := range w.Spec.Template.Spec.Containers {
		for _, p := range c.Ports {
			if strings.EqualFold(p.Name, "metrics") {
				return p, true
			}
		}
	}
	return dataplane.ContainerPort{}, false
}

// workloadLocation builds a report.Location for a workload-scoped finding.
func workloadLocation(w dataplane.Workload) report.Location {
	return report.Location{
		File:      w.SourceFile,
		Kind:      w.Kind,
		Namespace: w.Metadata.Namespace,
		Name:      w.Metadata.Name,
	}
}

// containerLocation builds a report.Location for a container-scoped finding.
func containerLocation(w dataplane.Workload, containerName string) report.Location {
	loc := workloadLocation(w)
	loc.Container = containerName
	return loc
}

// isSubset reports whether every key/value pair in sub is present in super.
// An empty/nil sub is NOT considered a subset here (used for Service
// selectors, where an empty selector matches nothing — see
// checks_service_wiring.go for the opposite convention used for
// ServiceMonitor label selectors).
func isSubset(sub, super map[string]string) bool {
	if len(sub) == 0 {
		return false
	}
	for k, v := range sub {
		if super[k] != v {
			return false
		}
	}
	return true
}
