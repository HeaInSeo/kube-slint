package jsonendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_FlattensNumericLeaves(t *testing.T) {
	values, err := Parse(strings.NewReader(`{
		"requests": 3,
		"memstats": {"Alloc": 10, "Name": "ignored"},
		"buckets": [1, 2],
		"enabled": true
	}`))
	require.NoError(t, err)

	assert.Equal(t, 3.0, values["requests"])
	assert.Equal(t, 10.0, values["memstats.Alloc"])
	assert.Equal(t, 1.0, values["buckets.0"])
	assert.Equal(t, 2.0, values["buckets.1"])
	assert.NotContains(t, values, "memstats.Name")
	assert.NotContains(t, values, "enabled")
}

func TestFetcher_FetchesHTTPJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token", r.Header.Get("X-Test"))
		_, _ = w.Write([]byte(`{"counter": 7}`))
	}))
	defer ts.Close()

	f := &Fetcher{URL: ts.URL, Headers: map[string]string{"X-Test": "token"}}
	s, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 7.0, s.Values["counter"])
}

func TestFetcher_PreFetchCachesFirstSample(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"counter": ` + string(rune('0'+calls)) + `}`))
	}))
	defer ts.Close()

	f := New(ts.URL)
	require.NoError(t, f.PreFetch(context.Background()))

	first, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)
	second, err := f.Fetch(context.Background(), time.Now())
	require.NoError(t, err)

	assert.Equal(t, 1.0, first.Values["counter"])
	assert.Equal(t, 2.0, second.Values["counter"])
	assert.Equal(t, 2, calls)
}

func TestFetcher_NonOKStatusReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	_, err := New(ts.URL).Fetch(context.Background(), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 503")
}

func TestFetcher_ImplementsSnapshotFetcher(t *testing.T) {
	var _ fetch.SnapshotFetcher = New("http://127.0.0.1/debug/vars")
}
