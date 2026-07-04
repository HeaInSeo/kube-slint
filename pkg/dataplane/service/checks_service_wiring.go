package service

import (
	"fmt"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

func serviceWiringCheck() CheckDef {
	return CheckDef{
		ID:    "KSL-DP-004",
		Title: "Metrics Service and ServiceMonitor wiring",
		Description: "A Service must select the workload's metrics port. A Prometheus Operator " +
			"ServiceMonitor matching that Service is recommended but only produces a warning if absent.",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				port, ok := findMetricsPort(w)
				if !ok {
					continue // KSL-DP-001 already reports the missing metrics port
				}

				svc, svcPort, ok := findMatchingService(b.Services, w, port)
				if !ok {
					out = append(out, report.Finding{
						RuleID:      "KSL-DP-004",
						Severity:    report.SeverityError,
						Message:     "no Service selects this workload's metrics port",
						Remediation: "add a Service in the same namespace whose selector matches this pod template's labels and whose port targets the metrics containerPort",
						Location:    workloadLocation(w),
					})
					continue
				}

				if !hasMatchingServiceMonitor(b.ServiceMonitors, svc, svcPort) {
					out = append(out, report.Finding{
						RuleID:   "KSL-DP-004",
						Severity: report.SeverityWarning,
						Message: fmt.Sprintf("no ServiceMonitor matches metrics Service %s/%s",
							svc.Metadata.Namespace, svc.Metadata.Name),
						Remediation: "add a ServiceMonitor selecting this Service's labels with an endpoint for its metrics port",
						Location:    workloadLocation(w),
					})
				}
			}
			return out
		},
	}
}

// findMatchingService finds a same-namespace Service whose selector matches
// the workload's pod template labels and whose port targets the workload's
// metrics containerPort. An empty/nil Service selector never matches (real
// Kubernetes semantics: an empty selector selects nothing automatically).
func findMatchingService(services []dataplane.Service, w dataplane.Workload, port dataplane.ContainerPort) (dataplane.Service, dataplane.ServicePort, bool) {
	podLabels := w.Spec.Template.Metadata.Labels
	for _, svc := range services {
		if svc.Metadata.Namespace != w.Metadata.Namespace {
			continue
		}
		if !isSubset(svc.Spec.Selector, podLabels) {
			continue
		}
		for _, sp := range svc.Spec.Ports {
			if servicePortMatchesContainerPort(sp, port) {
				return svc, sp, true
			}
		}
	}
	return dataplane.Service{}, dataplane.ServicePort{}, false
}

func servicePortMatchesContainerPort(sp dataplane.ServicePort, cp dataplane.ContainerPort) bool {
	if sp.TargetPort != nil {
		switch tp := sp.TargetPort.(type) {
		case string:
			return tp == cp.Name
		case int:
			return tp == cp.ContainerPort
		default:
			// yaml.v3 may decode a bare integer scalar into `any` as int,
			// int64, or uint64 depending on value range; fall back to a
			// string comparison against the numeric containerPort.
			return fmt.Sprint(tp) == fmt.Sprint(cp.ContainerPort)
		}
	}
	return sp.Port == cp.ContainerPort
}

// hasMatchingServiceMonitor reports whether any same-namespace ServiceMonitor
// selects svc's labels and has an endpoint naming the metrics ServicePort
// that findMatchingService actually matched.
//
// Unlike findMatchingService's Service selector (where an empty selector
// matches nothing), an empty ServiceMonitor MatchLabels is treated as
// matching all Services — this mirrors real Kubernetes label-selector
// semantics (an empty selector selects everything) and is intentionally the
// opposite convention from the Service->pod match above.
func hasMatchingServiceMonitor(monitors []dataplane.ServiceMonitor, svc dataplane.Service, metricsPort dataplane.ServicePort) bool {
	for _, sm := range monitors {
		if sm.Metadata.Namespace != svc.Metadata.Namespace {
			continue
		}
		if len(sm.Spec.Selector.MatchLabels) > 0 && !isSubset(sm.Spec.Selector.MatchLabels, svc.Metadata.Labels) {
			continue
		}
		for _, ep := range sm.Spec.Endpoints {
			if ep.Port == metricsPort.Name {
				return true
			}
		}
	}
	return false
}
