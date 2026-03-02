package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/test/e2e/harness"
)

// This represents a dummy external operator's main.go
// trying to integrate kube-slint for SLI tracking.
func main() {
	fmt.Println("Starting external operator minimal onboarding validation...")

	// 1. Consumer defines their specs using the API
	mySpecs := []spec.SLISpec{
		{
			ID: "operator_churn",
			Inputs: []spec.MetricRef{
				spec.UnsafePromKey("workqueue_adds_total{name=\"my-operator\"}"),
			},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge: &spec.JudgeSpec{
				Rules: []spec.Rule{
					{Metric: "value", Op: spec.OpLT, Target: 100.0, Level: spec.LevelFail},
				},
			},
		},
	}

	// 2. Setup standard Session Config
	// friction: it requires configuring endpoints manually if not using default
	cfg := harness.SessionConfig{
		Namespace:          "my-operator-system",
		MetricsServiceName: "my-operator-metrics-service",
		Specs:              mySpecs,
	}

	// 3. Initialize Session
	session := harness.NewSession(cfg)

	ctx := context.Background()

	// Simulate starting the control loop
	fmt.Println("kube-slint Session Start()...")
	session.Start()

	// Simulate some operator work time
	time.Sleep(1 * time.Second)

	// Finish and generate report
	fmt.Println("kube-slint Session End()...")
	summary, err := session.End(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning during session end: %v\n", err)
	}

	// Prove API is accessible
	if summary != nil {
		fmt.Printf("Generated Reliability Status: %s\n", summary.Reliability.EvaluationStatus)
	}

	fmt.Println("External operator finished successfully.")
}
