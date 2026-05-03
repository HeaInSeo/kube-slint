// Package portforward provides a MetricsFetcher that scrapes /metrics via
// kubectl port-forward, eliminating the need to create a curl pod.
package portforward

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promtext"
)

const (
	defaultRemotePort = 8080
	defaultPath       = "/metrics"
	readyTimeout      = 4 * time.Second // total deadline for port-forward to become reachable
	readyInterval     = 200 * time.Millisecond
	scrapeHTTPTimeout = 30 * time.Second // per-request timeout; prevents hang if port-forward dies
)

// httpClient is a shared client with a bounded timeout.
// http.DefaultClient has no timeout and can block indefinitely.
var httpClient = &http.Client{Timeout: scrapeHTTPTimeout}

// Fetcher scrapes a Kubernetes Service's /metrics endpoint via kubectl port-forward.
// It implements both fetch.MetricsFetcher and fetch.SnapshotFetcher.
//
// Usage:
//
//	f := &portforward.Fetcher{Namespace: "my-ns", ServiceName: "my-svc"}
//	sess := harness.NewSession(harness.SessionConfig{..., Fetcher: f})
//
// Note: Fetcher is not safe for concurrent use; the harness calls Fetch sequentially.
type Fetcher struct {
	Namespace   string
	ServiceName string
	RemotePort  int    // defaults to 8080 if zero
	Path        string // defaults to /metrics if empty

	cmd        *exec.Cmd
	localPort  int
	cancel     context.CancelFunc
	startCache *fetch.Sample
	fetchCount int
}

// Start launches kubectl port-forward and waits until the local port is reachable.
// The caller must call Stop() when done.
func (f *Fetcher) Start(ctx context.Context) error {
	port, err := freePort()
	if err != nil {
		return fmt.Errorf("portforward: find free port: %w", err)
	}
	f.localPort = port

	remote := f.RemotePort
	if remote == 0 {
		remote = defaultRemotePort
	}

	pfCtx, cancel := context.WithCancel(ctx)
	f.cancel = cancel

	target := fmt.Sprintf("svc/%s", f.ServiceName)
	portMap := fmt.Sprintf("%d:%d", f.localPort, remote)
	f.cmd = exec.CommandContext(pfCtx, "kubectl", "port-forward",
		"-n", f.Namespace, target, portMap)

	if err := f.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("portforward: start kubectl: %w", err)
	}

	if err := f.waitReady(ctx); err != nil {
		f.Stop()
		return err
	}
	return nil
}

// Stop terminates the kubectl port-forward process.
func (f *Fetcher) Stop() {
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	if f.cmd != nil && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
		_ = f.cmd.Wait()
		f.cmd = nil
	}
}

// PreFetch captures a start-of-window snapshot (implements fetch.SnapshotFetcher).
// Called by harness.Session.Start().
func (f *Fetcher) PreFetch(ctx context.Context) error {
	if f.localPort == 0 {
		if err := f.Start(ctx); err != nil {
			return err
		}
	}
	s, err := f.scrape(ctx)
	if err != nil {
		return err
	}
	f.startCache = &s
	return nil
}

// Fetch returns a metrics snapshot. The first call returns the cached start
// snapshot (captured by PreFetch); subsequent calls make a live HTTP request.
func (f *Fetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	if f.fetchCount == 0 && f.startCache != nil {
		f.fetchCount++
		return *f.startCache, nil
	}
	f.fetchCount++
	return f.scrape(ctx)
}

func (f *Fetcher) scrape(ctx context.Context) (fetch.Sample, error) {
	path := f.Path
	if path == "" {
		path = defaultPath
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", f.localPort, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("portforward: build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("portforward: GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fetch.Sample{}, fmt.Errorf(
			"portforward: GET %s: status %d: %s",
			url, resp.StatusCode, strings.TrimSpace(string(body)),
		)
	}

	values, err := promtext.ParseTextToMap(resp.Body)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("portforward: parse metrics: %w", err)
	}
	return fetch.Sample{At: time.Now(), Values: values}, nil
}

// waitReady polls the /metrics endpoint until it responds or readyTimeout elapses.
// The ticker-based loop is ctx-aware: no time.Sleep blocking.
func (f *Fetcher) waitReady(ctx context.Context) error {
	path := f.Path
	if path == "" {
		path = defaultPath
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", f.localPort, path)

	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	ticker := time.NewTicker(readyInterval)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(readyCtx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("portforward: wait-ready build request: %w", err)
		}
		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("portforward: service %s/%s not ready after %v: %w",
				f.Namespace, f.ServiceName, readyTimeout, readyCtx.Err())
		case <-ticker.C:
		}
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}
