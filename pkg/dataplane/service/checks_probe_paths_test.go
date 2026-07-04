package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
)

func TestProbePathCheck_CorrectPaths_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name:           "c",
			ReadinessProbe: &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/readyz"}},
			LivenessProbe:  &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/livez"}},
		}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-002", b))
}

func TestProbePathCheck_WrongReadinessPath_Warn(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name:           "c",
			ReadinessProbe: &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/healthz"}},
			LivenessProbe:  &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/livez"}},
		}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-002", b), 1)
}

func TestProbePathCheck_BothWrong_TwoWarnings(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name:           "c",
			ReadinessProbe: &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/healthz"}},
			LivenessProbe:  &dataplane.Probe{HTTPGet: &dataplane.HTTPGetAction{Path: "/healthz"}},
		}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-002", b), 2)
}

func TestProbePathCheck_TCPSocketProbe_Skipped(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name:           "c",
			ReadinessProbe: &dataplane.Probe{}, // HTTPGet nil == tcpSocket/exec probe
		}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-002", b))
}

func TestProbePathCheck_MissingProbe_Skipped(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c"}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-002", b))
}
