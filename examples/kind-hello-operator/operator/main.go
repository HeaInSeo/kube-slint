// hello-operator: minimal metrics-emitting service for kube-slint integration testing.
//
// Uses only Go stdlib — no external dependencies required.
// Exposes Prometheus text-format /metrics on HTTP port 8080.
//
// Build:
//
//	docker build -t hello-operator:dev examples/kind-hello-operator/operator/
//	kind load docker-image hello-operator:dev --name slint-demo
//
// In your kube-slint SessionConfig set:
//
//	ServiceURLFormat: slint.ServiceURLHTTP  // "http://%s.%s.svc:8080/metrics"
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	reconcileTotal  atomic.Int64
	reconcileErrors atomic.Int64
	workqueueDepth  atomic.Int64
	workqueueAdds   atomic.Int64
)

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP hello_reconcile_total Total reconcile loops executed.\n")
	fmt.Fprintf(w, "# TYPE hello_reconcile_total counter\n")
	fmt.Fprintf(w, "hello_reconcile_total %d\n", reconcileTotal.Load())
	fmt.Fprintf(w, "# HELP hello_reconcile_errors_total Total reconcile errors.\n")
	fmt.Fprintf(w, "# TYPE hello_reconcile_errors_total counter\n")
	fmt.Fprintf(w, "hello_reconcile_errors_total %d\n", reconcileErrors.Load())
	fmt.Fprintf(w, "# HELP hello_workqueue_depth Current workqueue depth.\n")
	fmt.Fprintf(w, "# TYPE hello_workqueue_depth gauge\n")
	fmt.Fprintf(w, "hello_workqueue_depth %d\n", workqueueDepth.Load())
	fmt.Fprintf(w, "# HELP hello_workqueue_adds_total Total items added to workqueue.\n")
	fmt.Fprintf(w, "# TYPE hello_workqueue_adds_total counter\n")
	fmt.Fprintf(w, "hello_workqueue_adds_total %d\n", workqueueAdds.Load())
}

func main() {
	addr := ":8080"
	if v := os.Getenv("METRICS_ADDR"); v != "" {
		addr = v
	}

	// Simulate reconcile loops every 500ms.
	go func() {
		for {
			workqueueAdds.Add(1)
			workqueueDepth.Add(1)
			time.Sleep(50 * time.Millisecond)
			reconcileTotal.Add(1)
			workqueueDepth.Add(-1)
			if rand.Float64() < 0.05 {
				reconcileErrors.Add(1)
			}
			time.Sleep(450 * time.Millisecond)
		}
	}()

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	fmt.Printf("hello-operator: listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "hello-operator: %v\n", err)
		os.Exit(1)
	}
}
