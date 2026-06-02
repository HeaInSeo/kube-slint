package k8sobject

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/kubeutil"
	"github.com/HeaInSeo/kube-slint/pkg/slo"
)

// k8sObjectList is the minimal representation of a kubectl get -o json response.
type k8sObjectList struct {
	Items []k8sObject `json:"items"`
}

type k8sObject struct {
	Metadata k8sObjectMeta   `json:"metadata"`
	Status   k8sObjectStatus `json:"status"`
}

type k8sObjectMeta struct {
	UID               string            `json:"uid"`
	DeletionTimestamp *time.Time        `json:"deletionTimestamp,omitempty"`
	OwnerReferences   []k8sOwnerRef     `json:"ownerReferences,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
}

type k8sObjectStatus struct {
	Phase string `json:"phase,omitempty"` // pods only
}

type k8sOwnerRef struct {
	UID string `json:"uid"`
}

// listObjects fetches objects matching cfg.Selector, then removes any that
// also match cfg.ExcludeSelector (e.g. kube-slint curlpods).
func listObjects(ctx context.Context, cfg Config) ([]k8sObject, error) {
	main, err := kubectlList(ctx, cfg.Runner, cfg.Logger, cfg.Namespace, cfg.Resource, cfg.Selector)
	if err != nil {
		return nil, fmt.Errorf("k8sobject: list %s: %w", cfg.Resource, err)
	}
	if cfg.ExcludeSelector == "" {
		return main, nil
	}
	excluded, err := kubectlList(ctx, cfg.Runner, cfg.Logger, cfg.Namespace, cfg.Resource, cfg.ExcludeSelector)
	if err != nil {
		// exclusion failure is non-fatal: log and continue with full list
		cfg.Logger.Logf("k8sobject: exclude list failed (using full list): %v", err)
		return main, nil
	}
	excludeUIDs := make(map[string]bool, len(excluded))
	for _, o := range excluded {
		excludeUIDs[o.Metadata.UID] = true
	}
	filtered := main[:0]
	for _, o := range main {
		if !excludeUIDs[o.Metadata.UID] {
			filtered = append(filtered, o)
		}
	}
	return filtered, nil
}

func kubectlList(ctx context.Context, runner kubeutil.CmdRunner, logger slo.Logger, ns, resource, selector string) ([]k8sObject, error) {
	args := []string{"get", resource, "-n", ns, "-o", "json"}
	if selector != "" {
		args = append(args, "-l", selector)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := runner.Run(ctx, logger, cmd)
	if err != nil {
		return nil, err
	}
	var list k8sObjectList
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("parse kubectl output: %w", err)
	}
	return list.Items, nil
}

// toStartMetrics converts a start-of-window object list to a metric map.
// Gauge metrics are zeroed at start so ComputeEnd specs don't get StatusSkip.
func toStartMetrics(objs []k8sObject, prefix string) map[string]float64 {
	return map[string]float64{
		prefix + "_count":                 float64(len(objs)),
		prefix + "_orphan_end":            0,
		prefix + "_ownerref_missing_end":  0,
		prefix + "_stuck_terminating_end": 0,
	}
}

// toEndMetrics converts an end-of-window object list to a metric map.
func toEndMetrics(objs []k8sObject, prefix string, stuckThreshold time.Duration, now time.Time) map[string]float64 {
	uidSet := make(map[string]bool, len(objs))
	for _, o := range objs {
		uidSet[o.Metadata.UID] = true
	}

	var orphan, ownerrefMissing, stuckTerminating int
	for _, o := range objs {
		// orphan: no ownerReferences
		if len(o.Metadata.OwnerReferences) == 0 {
			orphan++
		}
		// ownerref_missing: at least one ownerRef UID not in current object set
		for _, ref := range o.Metadata.OwnerReferences {
			if !uidSet[ref.UID] {
				ownerrefMissing++
				break
			}
		}
		// stuck_terminating: DeletionTimestamp set and overdue
		if o.Metadata.DeletionTimestamp != nil {
			age := now.Sub(*o.Metadata.DeletionTimestamp)
			if stuckThreshold == 0 || age > stuckThreshold {
				stuckTerminating++
			}
		}
	}

	return map[string]float64{
		prefix + "_count":                 float64(len(objs)),
		prefix + "_orphan_end":            float64(orphan),
		prefix + "_ownerref_missing_end":  float64(ownerrefMissing),
		prefix + "_stuck_terminating_end": float64(stuckTerminating),
	}
}
