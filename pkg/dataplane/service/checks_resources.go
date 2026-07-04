package service

import (
	"fmt"
	"sort"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

var requiredResourceKeys = []string{"cpu", "memory"}

func resourcesCheck() CheckDef {
	return CheckDef{
		ID:          "KSL-DP-005",
		Title:       "Resource requests and limits are set",
		Description: "Every container must set cpu and memory under both resources.requests and resources.limits.",
		Run: func(b *dataplane.Bundle) []report.Finding {
			var out []report.Finding
			for _, w := range b.Workloads {
				for _, c := range w.Spec.Template.Spec.Containers {
					if missing := missingKeys(c.Resources.Requests); len(missing) > 0 {
						out = append(out, resourceFinding(w, c, "requests", missing))
					}
					if missing := missingKeys(c.Resources.Limits); len(missing) > 0 {
						out = append(out, resourceFinding(w, c, "limits", missing))
					}
				}
			}
			return out
		},
	}
}

func missingKeys(list dataplane.ResourceList) []string {
	var missing []string
	for _, k := range requiredResourceKeys {
		if _, ok := list[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

func resourceFinding(w dataplane.Workload, c dataplane.Container, field string, missing []string) report.Finding {
	return report.Finding{
		RuleID:      "KSL-DP-005",
		Severity:    report.SeverityError,
		Message:     fmt.Sprintf("container %q: resources.%s missing keys: %v", c.Name, field, missing),
		Remediation: fmt.Sprintf("set resources.%s.{%v} on this container", field, missing),
		Location:    containerLocation(w, c.Name),
	}
}
