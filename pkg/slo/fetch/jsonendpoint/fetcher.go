// Package jsonendpoint provides a source-neutral HTTP JSON MetricsFetcher.
//
// It is intended for expvar-style endpoints and small custom status endpoints
// that expose numeric values as JSON. Numeric leaves are flattened into
// dot-separated keys, e.g. {"memstats":{"Alloc":10}} becomes
// "memstats.Alloc" = 10.
package jsonendpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
)

const (
	defaultHTTPTimeout = 30 * time.Second
	maxErrorBodyBytes  = 512
)

// Fetcher retrieves a JSON document over HTTP and converts numeric leaves into
// a fetch.Sample. It implements fetch.SnapshotFetcher so Session.Start() can
// cache the start-of-window sample for delta computation.
//
// Fetcher is not safe for concurrent use; kube-slint sessions call Fetch
// sequentially.
type Fetcher struct {
	URL     string
	Headers map[string]string
	Client  *http.Client

	startCache *fetch.Sample
	startErr   error
	fetchCount int
}

// New returns a Fetcher for url with safe defaults.
func New(url string) *Fetcher {
	return &Fetcher{URL: url}
}

// PreFetch captures a start-of-window JSON snapshot.
func (f *Fetcher) PreFetch(ctx context.Context) error {
	s, err := f.scrape(ctx)
	if err != nil {
		f.startErr = err
		return err
	}
	f.startCache = &s
	f.startErr = nil
	return nil
}

// Fetch returns the cached PreFetch snapshot on the first call, then live
// snapshots afterwards.
func (f *Fetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	if f.fetchCount == 0 && f.startCache != nil {
		f.fetchCount++
		return *f.startCache, nil
	}
	if f.fetchCount == 0 && f.startErr != nil {
		f.fetchCount++
		return fetch.Sample{}, fmt.Errorf("prefetch start snapshot failed: %w", f.startErr)
	}
	f.fetchCount++
	return f.scrape(ctx)
}

func (f *Fetcher) scrape(ctx context.Context) (fetch.Sample, error) {
	if strings.TrimSpace(f.URL) == "" {
		return fetch.Sample{}, fmt.Errorf("jsonendpoint: URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("jsonendpoint: build request: %w", err)
	}
	for k, v := range f.Headers {
		req.Header.Set(k, v)
	}

	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("jsonendpoint: GET %s: %w", f.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return fetch.Sample{}, fmt.Errorf(
			"jsonendpoint: GET %s: status %d: %s",
			f.URL, resp.StatusCode, strings.TrimSpace(string(body)),
		)
	}

	values, err := Parse(resp.Body)
	if err != nil {
		return fetch.Sample{}, fmt.Errorf("jsonendpoint: parse JSON: %w", err)
	}
	return fetch.Sample{At: time.Now(), Values: values}, nil
}

// Parse flattens numeric JSON leaves into dot-separated metric keys. Objects
// contribute field names, arrays contribute numeric indexes, and non-numeric
// leaves are ignored.
func Parse(r io.Reader) (map[string]float64, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}

	values := map[string]float64{}
	flatten(values, "", root)
	return values, nil
}

func flatten(out map[string]float64, prefix string, v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flatten(out, key, child)
		}
	case []any:
		for i, child := range x {
			key := strconv.Itoa(i)
			if prefix != "" {
				key = prefix + "." + key
			}
			flatten(out, key, child)
		}
	case json.Number:
		if prefix == "" {
			return
		}
		if f, err := x.Float64(); err == nil {
			out[prefix] = f
		}
	}
}
