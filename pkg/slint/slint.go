// Package slint is the public API entry point for kube-slint.
//
// Consumers should import this package rather than test/e2e/harness:
//
//	import "github.com/HeaInSeo/kube-slint/pkg/slint"
//
//	sess := slint.NewSession(slint.SessionConfig{
//	    Namespace:          "my-operator-system",
//	    MetricsServiceName: "my-operator-metrics-service",
//	    ArtifactsDir:       "artifacts",
//	    Specs:              slint.DefaultSpecs(),
//	})
//	sess.Start()
//	// ... run your E2E scenario ...
//	sess.End(ctx)
package slint

import "github.com/HeaInSeo/kube-slint/test/e2e/harness"

// Session manages a single SLI measurement window.
// Start() begins observation; End() collects metrics and writes artifacts.
type Session = harness.Session

// SessionConfig holds all inputs for a measurement session.
type SessionConfig = harness.SessionConfig

// NewSession creates a new Session with defaults applied.
var NewSession = harness.NewSession

// DefaultSpecs returns the standard controller-runtime SLI spec set:
// reconcile totals, workqueue depth/adds/retries, REST client requests.
var DefaultSpecs = harness.DefaultV3Specs

// BaselineSpecs is an alias for DefaultSpecs.
var BaselineSpecs = harness.BaselineV3Specs
