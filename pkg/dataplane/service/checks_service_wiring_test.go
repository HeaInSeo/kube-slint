package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
)

func workloadWithMetricsPort() dataplane.Workload {
	return workload(dataplane.Container{Name: "c", Ports: []dataplane.ContainerPort{{Name: "metrics", ContainerPort: 8080}}})
}

func metricsService() dataplane.Service {
	return dataplane.Service{
		Kind:       "Service",
		Metadata:   dataplane.ObjectMeta{Name: "app-metrics", Namespace: "ns", Labels: map[string]string{"app": "app"}},
		SourceFile: "app.yaml",
		Spec: dataplane.ServiceSpec{
			Selector: map[string]string{"app": "app"},
			Ports:    []dataplane.ServicePort{{Name: "metrics", Port: 8080, TargetPort: 8080}},
		},
	}
}

func TestServiceWiringCheck_NoMetricsPort_DeferredToKSL001(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workload(dataplane.Container{Name: "c"})}}
	assert.Empty(t, runCheck(t, "KSL-DP-004", b))
}

func TestServiceWiringCheck_NoMatchingService_Error(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workloadWithMetricsPort()}}
	msgs := runCheck(t, "KSL-DP-004", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "no Service selects")
}

func TestServiceWiringCheck_ServiceMatches_NoServiceMonitor_Warn(t *testing.T) {
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{metricsService()},
	}
	msgs := runCheck(t, "KSL-DP-004", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "no ServiceMonitor matches")
}

func TestServiceWiringCheck_ServiceMonitorMatches_Pass(t *testing.T) {
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{metricsService()},
		ServiceMonitors: []dataplane.ServiceMonitor{{
			Kind:       "ServiceMonitor",
			Metadata:   dataplane.ObjectMeta{Name: "app-sm", Namespace: "ns"},
			SourceFile: "app.yaml",
			Spec: dataplane.ServiceMonitorSpec{
				Selector:  dataplane.LabelSelector{MatchLabels: map[string]string{"app": "app"}},
				Endpoints: []dataplane.ServiceMonitorEndpoint{{Port: "metrics"}},
			},
		}},
	}
	assert.Empty(t, runCheck(t, "KSL-DP-004", b))
}

func TestServiceWiringCheck_ServiceMonitorEmptySelector_MatchesAll_Pass(t *testing.T) {
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{metricsService()},
		ServiceMonitors: []dataplane.ServiceMonitor{{
			Kind:       "ServiceMonitor",
			Metadata:   dataplane.ObjectMeta{Name: "catch-all-sm", Namespace: "ns"},
			SourceFile: "app.yaml",
			Spec: dataplane.ServiceMonitorSpec{
				Endpoints: []dataplane.ServiceMonitorEndpoint{{Port: "metrics"}},
			},
		}},
	}
	assert.Empty(t, runCheck(t, "KSL-DP-004", b))
}

func TestServiceWiringCheck_ServiceSelectorNotSubsetOfPodLabels_NoMatch(t *testing.T) {
	svc := metricsService()
	svc.Spec.Selector = map[string]string{"app": "different"}
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{svc},
	}
	msgs := runCheck(t, "KSL-DP-004", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "no Service selects")
}

func TestServiceWiringCheck_EmptyServiceSelector_NoMatch(t *testing.T) {
	svc := metricsService()
	svc.Spec.Selector = nil
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{svc},
	}
	msgs := runCheck(t, "KSL-DP-004", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "no Service selects")
}

func TestServiceWiringCheck_DifferentNamespace_NoMatch(t *testing.T) {
	svc := metricsService()
	svc.Metadata.Namespace = "other-ns"
	b := &dataplane.Bundle{
		Workloads: []dataplane.Workload{workloadWithMetricsPort()},
		Services:  []dataplane.Service{svc},
	}
	msgs := runCheck(t, "KSL-DP-004", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "no Service selects")
}
