package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/test/e2e/harness"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	cfg       *rest.Config
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	slintSess *harness.Session
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Consumer Onboarding Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false, // In real world it would be True, but here we don't have CRDs for minimal test
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Start manager (the "consumer operator")
	go func() {
		defer GinkgoRecover()
		// Re-use main() logic essentially
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Metrics: metricsserver.Options{BindAddress: ":8080"},
			HealthProbeBindAddress: "0",
			LeaderElection:         false,
		})
		Expect(err).NotTo(HaveOccurred())
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// Wait for metrics server
	time.Sleep(3 * time.Second)

	By("initializing kube-slint harness session")
	slintSess = harness.NewSession(
		harness.SessionConfig{
			Namespace:          "default",
			MetricsServiceName: "test-operator",
			TestCase:           "integration-test",
			Suite:              "consumer-onboarding",
			Specs: []spec.SLISpec{
				// Just a basic mock spec to see if engine runs
				{
					ID:    "harness.boot.test",
					Title: "Harness Boot Test",
					Inputs: []spec.MetricRef{
						spec.UnsafePromKey(`up`),
					},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
				},
			},
		},
	)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Minimal Default SLI Pathway", func() {
	It("should attach to the mock operator metrics endpoint and generate summary", func() {
		// Evaluate the SLIs using Start and End
		slintSess.Start()
		time.Sleep(1 * time.Second) // wait a little for metrics to be scrappable
		_, err := slintSess.End(ctx)
		// We expect no panic. If it fails to scrape it logs. We want to see if the interface works.
		Expect(err).NotTo(HaveOccurred())
	})
})
