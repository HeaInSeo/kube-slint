package service

import (
	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

func gracePeriodCheck() CheckDef {
	return CheckDef{
		ID:    "KSL-DP-006",
		Title: "terminationGracePeriodSeconds is explicit",
		Description: "The pod template should explicitly set terminationGracePeriodSeconds rather than " +
			"relying on the implicit Kubernetes default (30s).",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				if w.Spec.Template.Spec.TerminationGracePeriodSeconds != nil {
					continue
				}
				out = append(out, report.Finding{
					RuleID:      "KSL-DP-006",
					Severity:    report.SeverityWarning,
					Message:     "terminationGracePeriodSeconds is not set; relying on the implicit default",
					Remediation: "set spec.template.spec.terminationGracePeriodSeconds explicitly",
					Location:    workloadLocation(w),
				})
			}
			return out
		},
	}
}
