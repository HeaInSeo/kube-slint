// Package k8sobject provides a fetch.SnapshotFetcher that captures Kubernetes
// object counts (Pods, Jobs) via kubectl list calls.
//
// It fits the existing 2-point engine model:
//   - PreFetch() captures object state at session start.
//   - First Fetch() returns the cached start snapshot.
//   - Second Fetch() queries end state and computes derived metrics.
package k8sobject

import (
	"context"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/kubeutil"
	"github.com/HeaInSeo/kube-slint/pkg/slo"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
)

// Config holds parameters for K8sObjectFetcher.
type Config struct {
	Namespace string
	Resource  string // "pods" | "jobs" — default "pods"
	Selector  string // include label selector
	// ExcludeSelector filters out objects that must not be counted
	// (e.g. "app.kubernetes.io/managed-by=kube-slint" to skip curlpods).
	ExcludeSelector string
	// MetricPrefix is prepended to all metric keys.
	// Default: "k8s_" + Resource  (e.g. "k8s_pods").
	MetricPrefix string
	// StuckTerminatingThreshold: minimum time in Terminating before an object
	// is counted as stuck. 0 means any Terminating object is counted.
	StuckTerminatingThreshold time.Duration

	Runner kubeutil.CmdRunner
	Logger slo.Logger
}

// K8sObjectFetcher implements fetch.SnapshotFetcher.
type K8sObjectFetcher struct {
	cfg        Config
	startCache *fetch.Sample
	fetchCount int
}

// New returns a K8sObjectFetcher with safe defaults applied.
func New(cfg Config) *K8sObjectFetcher {
	if cfg.Resource == "" {
		cfg.Resource = "pods"
	}
	if cfg.MetricPrefix == "" {
		cfg.MetricPrefix = "k8s_" + cfg.Resource
	}
	if cfg.Runner == nil {
		cfg.Runner = kubeutil.DefaultRunner{}
	}
	cfg.Logger = slo.NewLogger(cfg.Logger)
	return &K8sObjectFetcher{cfg: cfg}
}

// PreFetch captures the start-of-window object state (implements fetch.SnapshotFetcher).
// Called by harness.Session.Start(). Failure is non-fatal per kube-slint safety-first policy.
func (f *K8sObjectFetcher) PreFetch(ctx context.Context) error {
	objs, err := listObjects(ctx, f.cfg)
	if err != nil {
		return err
	}
	values := toStartMetrics(objs, f.cfg.MetricPrefix)
	s := fetch.Sample{At: time.Now(), Values: values}
	f.startCache = &s
	return nil
}

// Fetch returns a metrics snapshot.
// First call: returns the cached start snapshot from PreFetch.
// Second call: queries end-of-window object state and computes derived metrics.
func (f *K8sObjectFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	f.fetchCount++
	if f.fetchCount == 1 && f.startCache != nil {
		cached := *f.startCache
		cached.At = at
		return cached, nil
	}

	objs, err := listObjects(ctx, f.cfg)
	if err != nil {
		return fetch.Sample{}, err
	}
	values := toEndMetrics(objs, f.cfg.MetricPrefix, f.cfg.StuckTerminatingThreshold, time.Now())
	return fetch.Sample{At: at, Values: values}, nil
}
