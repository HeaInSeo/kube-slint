package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
)

func TestProbeWiringCheck_BothPresent_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name:           "c",
			ReadinessProbe: &dataplane.Probe{},
			LivenessProbe:  &dataplane.Probe{},
		}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-003", b))
}

func TestProbeWiringCheck_OnlyLivenessMissing_OneFinding(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c", ReadinessProbe: &dataplane.Probe{}}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-003", b), 1)
}

func TestProbeWiringCheck_BothMissing_TwoFindings(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c"}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-003", b), 2)
}

func TestProbeWiringCheck_ZeroContainers_NoPanic(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workload()}}
	assert.Empty(t, runCheck(t, "KSL-DP-003", b))
}
