//go:build ignore

// hello-operator: minimal Prometheus-instrumented service for kube-slint integration testing.
//
// Build and deploy:
//
//	docker build -f operator/Dockerfile -t hello-operator:dev .
//	kind load docker-image hello-operator:dev --name slint-demo
//
// Exposes /metrics on HTTP port 8080 — no TLS required.
// In your kube-slint SessionConfig set:
//
//	ServiceURLFormat: slint.ServiceURLHTTP  // "http://%s.%s.svc:8080/metrics"
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reconcileTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hello_reconcile_total",
		Help: "Total number of reconcile loops executed.",
	})
	reconcileErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hello_reconcile_errors_total",
		Help: "Total number of reconcile errors.",
	})
	workqueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "hello_workqueue_depth",
		Help: "Current depth of the hello-operator workqueue.",
	})
	workqueueAdds = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hello_workqueue_adds_total",
		Help: "Total items added to the workqueue.",
	})
)

func main() {
	addr := ":8080"
	if v := os.Getenv("METRICS_ADDR"); v != "" {
		addr = v
	}

	// Background goroutine: simulate reconcile loops every 500ms.
	go func() {
		for {
			workqueueAdds.Inc()
			workqueueDepth.Inc()
			time.Sleep(50 * time.Millisecond)
			reconcileTotal.Inc()
			workqueueDepth.Dec()
			if rand.Float64() < 0.05 { // 5% error rate
				reconcileErrors.Inc()
			}
			time.Sleep(450 * time.Millisecond)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	fmt.Printf("hello-operator: listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "hello-operator: %v\n", err)
		os.Exit(1)
	}
}
