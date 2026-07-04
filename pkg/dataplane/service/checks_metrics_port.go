package service

import (
	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

func metricsPortCheck() CheckDef {
	return CheckDef{
		ID:          "KSL-DP-001",
		Title:       "Metrics port exposed",
		Description: "The workload's pod template must expose a container port named \"metrics\".",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				if _, ok := findMetricsPort(w); ok {
					continue
				}
				out = append(out, report.Finding{
					RuleID:      "KSL-DP-001",
					Severity:    report.SeverityError,
					Message:     "no container port named \"metrics\" found in this workload's pod template",
					Remediation: "add a containerPort with name: metrics to the container that serves /metrics",
					Location:    workloadLocation(w),
				})
			}
			return out
		},
	}
}
