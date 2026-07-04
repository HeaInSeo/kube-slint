package service

import (
	"fmt"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

func probePathCheck() CheckDef {
	return CheckDef{
		ID:    "KSL-DP-002",
		Title: "Probe paths follow /livez, /readyz convention",
		Description: "An HTTP readinessProbe should target /readyz and an HTTP livenessProbe should " +
			"target /livez. Non-HTTP probes (tcpSocket/exec) and missing probes are not judged here " +
			"(see KSL-DP-003 for missing probes).",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				for _, c := range w.Spec.Template.Spec.Containers {
					out = append(out, checkProbePath(w, c, "readinessProbe", c.ReadinessProbe, "/readyz")...)
					out = append(out, checkProbePath(w, c, "livenessProbe", c.LivenessProbe, "/livez")...)
				}
			}
			return out
		},
	}
}

func checkProbePath(w dataplane.Workload, c dataplane.Container, probeName string, probe *dataplane.Probe, wantPath string) []report.Finding {
	if probe == nil || probe.HTTPGet == nil || probe.HTTPGet.Path == wantPath {
		return nil
	}
	return []report.Finding{{
		RuleID:   "KSL-DP-002",
		Severity: report.SeverityWarning,
		Message: fmt.Sprintf("container %q: %s path %q does not follow the %q convention",
			c.Name, probeName, probe.HTTPGet.Path, wantPath),
		Remediation: fmt.Sprintf("point %s.httpGet.path at %s, or document why this container uses a different convention", probeName, wantPath),
		Location:    containerLocation(w, c.Name),
	}}
}
