package k8sobject

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner returns canned stdout for each Run call in order.
type mockRunner struct {
	responses []string
	idx       int
}

func (m *mockRunner) Run(_ context.Context, _ slo.Logger, _ *exec.Cmd) (string, error) {
	if m.idx >= len(m.responses) {
		return `{"items":[]}`, nil
	}
	out := m.responses[m.idx]
	m.idx++
	return out, nil
}

// podListJSON builds a minimal kubectl get pods -o json response.
func podListJSON(pods []podSpec) string {
	items := ""
	for i, p := range pods {
		if i > 0 {
			items += ","
		}
		ownerRefs := ""
		for j, ref := range p.ownerUIDs {
			if j > 0 {
				ownerRefs += ","
			}
			ownerRefs += `{"uid":"` + ref + `"}`
		}
		deletionTS := ""
		if !p.deletionTimestamp.IsZero() {
			deletionTS = `,"deletionTimestamp":"` + p.deletionTimestamp.Format(time.RFC3339) + `"`
		}
		items += `{
			"metadata":{"uid":"` + p.uid + `","ownerReferences":[` + ownerRefs + `]` + deletionTS + `},
			"status":{"phase":"` + p.phase + `"}
		}`
	}
	return `{"items":[` + items + `]}`
}

type podSpec struct {
	uid               string
	phase             string
	ownerUIDs         []string
	deletionTimestamp time.Time
}

// --- toStartMetrics ---

func TestToStartMetrics_ZeroesGauges(t *testing.T) {
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a"}},
		{Metadata: k8sObjectMeta{UID: "b"}},
	}
	m := toStartMetrics(objs, "k8s_pods")
	assert.Equal(t, 2.0, m["k8s_pods_count"])
	assert.Equal(t, 0.0, m["k8s_pods_orphan_end"])
	assert.Equal(t, 0.0, m["k8s_pods_ownerref_missing_end"])
	assert.Equal(t, 0.0, m["k8s_pods_stuck_terminating_end"])
}

// --- toEndMetrics ---

func TestToEndMetrics_Orphan(t *testing.T) {
	// object with no ownerRefs → orphan
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a", OwnerReferences: nil}},
		{Metadata: k8sObjectMeta{UID: "b", OwnerReferences: []k8sOwnerRef{{UID: "owner1"}}}},
	}
	m := toEndMetrics(objs, "k8s_pods", 0, time.Now())
	assert.Equal(t, 2.0, m["k8s_pods_count"])
	assert.Equal(t, 1.0, m["k8s_pods_orphan_end"])
}

func TestToEndMetrics_OwnerRefMissing(t *testing.T) {
	// "b" has ownerRef "owner1" which is NOT in the object set
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a"}},
		{Metadata: k8sObjectMeta{UID: "b", OwnerReferences: []k8sOwnerRef{{UID: "owner1"}}}},
	}
	m := toEndMetrics(objs, "k8s_pods", 0, time.Now())
	assert.Equal(t, 1.0, m["k8s_pods_ownerref_missing_end"])
}

func TestToEndMetrics_OwnerRefPresent(t *testing.T) {
	// "b" has ownerRef "a" which IS in the object set → not missing
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a"}},
		{Metadata: k8sObjectMeta{UID: "b", OwnerReferences: []k8sOwnerRef{{UID: "a"}}}},
	}
	m := toEndMetrics(objs, "k8s_pods", 0, time.Now())
	assert.Equal(t, 0.0, m["k8s_pods_ownerref_missing_end"])
}

func TestToEndMetrics_StuckTerminating_NoThreshold(t *testing.T) {
	// any pod with DeletionTimestamp is stuck when threshold=0
	ts := time.Now().Add(-10 * time.Second)
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a", DeletionTimestamp: &ts}},
		{Metadata: k8sObjectMeta{UID: "b"}},
	}
	m := toEndMetrics(objs, "k8s_pods", 0, time.Now())
	assert.Equal(t, 1.0, m["k8s_pods_stuck_terminating_end"])
}

func TestToEndMetrics_StuckTerminating_BelowThreshold(t *testing.T) {
	// pod has been terminating for 10s, threshold is 5m → not stuck
	ts := time.Now().Add(-10 * time.Second)
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a", DeletionTimestamp: &ts}},
	}
	m := toEndMetrics(objs, "k8s_pods", 5*time.Minute, time.Now())
	assert.Equal(t, 0.0, m["k8s_pods_stuck_terminating_end"])
}

func TestToEndMetrics_StuckTerminating_AboveThreshold(t *testing.T) {
	ts := time.Now().Add(-10 * time.Minute)
	objs := []k8sObject{
		{Metadata: k8sObjectMeta{UID: "a", DeletionTimestamp: &ts}},
	}
	m := toEndMetrics(objs, "k8s_pods", 5*time.Minute, time.Now())
	assert.Equal(t, 1.0, m["k8s_pods_stuck_terminating_end"])
}

// --- K8sObjectFetcher ---

func TestFetcher_PreFetch_And_Fetch(t *testing.T) {
	startJSON := podListJSON([]podSpec{
		{uid: "pod-a", phase: "Running"},
		{uid: "pod-b", phase: "Running"},
	})
	endJSON := podListJSON([]podSpec{
		{uid: "pod-a", phase: "Running"},
		{uid: "pod-b", phase: "Running"},
		{uid: "pod-c", phase: "Running"},
	})

	runner := &mockRunner{responses: []string{startJSON, endJSON}}
	cfg := Config{
		Namespace:    "test-ns",
		Resource:     "pods",
		MetricPrefix: "k8s_pods",
		Runner:       runner,
	}
	f := New(cfg)

	require.NoError(t, f.PreFetch(context.Background()))

	startSample, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 2.0, startSample.Values["k8s_pods_count"])

	endSample, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 3.0, endSample.Values["k8s_pods_count"])
}

func TestFetcher_Fetch_WithoutPreFetch_LiveQuery(t *testing.T) {
	// without PreFetch, first Fetch makes a live query
	liveJSON := podListJSON([]podSpec{{uid: "pod-x", phase: "Running"}})
	runner := &mockRunner{responses: []string{liveJSON}}
	f := New(Config{Namespace: "ns", Resource: "pods", MetricPrefix: "k8s_pods", Runner: runner})

	s, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1.0, s.Values["k8s_pods_count"])
}

func TestFetcher_ExcludeSelector_RemovesObjects(t *testing.T) {
	// main list has pod-a and pod-b; exclude list returns pod-b (kube-slint curlpod)
	mainJSON := podListJSON([]podSpec{
		{uid: "pod-a", phase: "Running"},
		{uid: "pod-b", phase: "Running"},
	})
	excludeJSON := podListJSON([]podSpec{
		{uid: "pod-b", phase: "Running"},
	})
	runner := &mockRunner{responses: []string{mainJSON, excludeJSON}}
	cfg := Config{
		Namespace:       "ns",
		Resource:        "pods",
		MetricPrefix:    "k8s_pods",
		ExcludeSelector: "app.kubernetes.io/managed-by=kube-slint",
		Runner:          runner,
	}
	f := New(cfg)
	require.NoError(t, f.PreFetch(context.Background()))

	s, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	// only pod-a remains after excluding pod-b
	assert.Equal(t, 1.0, s.Values["k8s_pods_count"])
}

func TestFetcher_StartMetrics_GaugesAreZero(t *testing.T) {
	// start snapshot must include gauge keys zeroed so ComputeEnd doesn't skip
	startJSON := podListJSON([]podSpec{{uid: "pod-a", phase: "Running"}})
	endJSON := podListJSON([]podSpec{{uid: "pod-a", phase: "Running"}})
	runner := &mockRunner{responses: []string{startJSON, endJSON}}
	f := New(Config{Namespace: "ns", Resource: "pods", MetricPrefix: "k8s_pods", Runner: runner})

	require.NoError(t, f.PreFetch(context.Background()))
	start, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)

	assert.Equal(t, 0.0, start.Values["k8s_pods_orphan_end"],
		"start gauge keys must be 0 so engine does not produce StatusSkip")
	assert.Equal(t, 0.0, start.Values["k8s_pods_ownerref_missing_end"])
	assert.Equal(t, 0.0, start.Values["k8s_pods_stuck_terminating_end"])
}

// --- fetch.SnapshotFetcher interface compliance ---

func TestFetcher_ImplementsSnapshotFetcher(t *testing.T) {
	var _ fetch.SnapshotFetcher = New(Config{})
}
