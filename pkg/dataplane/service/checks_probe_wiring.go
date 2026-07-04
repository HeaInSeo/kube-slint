package service

import (
	"fmt"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

func probeWiringCheck() CheckDef {
	return CheckDef{
		ID:          "KSL-DP-003",
		Title:       "readinessProbe and livenessProbe are configured",
		Description: "Every container must have both a readinessProbe and a livenessProbe configured.",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				for _, c := range w.Spec.Template.Spec.Containers {
					if c.ReadinessProbe == nil {
						out = append(out, missingProbeFinding(w, c, "readinessProbe"))
					}
					if c.LivenessProbe == nil {
						out = append(out, missingProbeFinding(w, c, "livenessProbe"))
					}
				}
			}
			return out
		},
	}
}

func missingProbeFinding(w dataplane.Workload, c dataplane.Container, probeName string) report.Finding {
	return report.Finding{
		RuleID:      "KSL-DP-003",
		Severity:    report.SeverityError,
		Message:     fmt.Sprintf("container %q: no %s configured", c.Name, probeName),
		Remediation: fmt.Sprintf("add a %s to this container", probeName),
		Location:    containerLocation(w, c.Name),
	}
}
