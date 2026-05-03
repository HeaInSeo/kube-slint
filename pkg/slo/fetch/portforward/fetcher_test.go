package portforward

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleMetrics = `# HELP controller_runtime_reconcile_total Total reconciliations
# TYPE controller_runtime_reconcile_total counter
controller_runtime_reconcile_total 10
workqueue_depth 0
rest_client_requests_total{code="200",method="GET"} 5
`

// portFrom extracts the integer port from an httptest.Server URL (http://127.0.0.1:<port>).
func portFrom(srv *httptest.Server) int {
	parts := strings.Split(srv.URL, ":")
	p, _ := strconv.Atoi(parts[len(parts)-1])
	return p
}

func TestScrape_ParsesMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, sampleMetrics)
	}))
	defer srv.Close()

	f := &Fetcher{localPort: portFrom(srv)}
	s, err := f.scrape(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10.0, s.Values["controller_runtime_reconcile_total"])
	assert.Equal(t, 0.0, s.Values["workqueue_depth"])
}

func TestScrape_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	f := &Fetcher{localPort: portFrom(srv)}
	_, err := f.scrape(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestFetch_ReturnsCacheOnFirstCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		_, _ = fmt.Fprint(w, sampleMetrics)
	}))
	defer srv.Close()

	f := &Fetcher{localPort: portFrom(srv)}

	// Simulate PreFetch: scrape once and cache
	s, err := f.scrape(context.Background())
	require.NoError(t, err)
	f.startCache = &s
	callsAfterPreFetch := callCount

	// First Fetch() must return the cache without an HTTP call
	_, err = f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, callsAfterPreFetch, callCount, "first Fetch must use cache")

	// Second Fetch() must make a live HTTP call
	_, err = f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, callsAfterPreFetch+1, callCount, "second Fetch must call HTTP")
}

func TestFreePort_ReturnsValidPort(t *testing.T) {
	port, err := freePort()
	require.NoError(t, err)
	assert.Greater(t, port, 1024)
	assert.Less(t, port, 65536)
}
