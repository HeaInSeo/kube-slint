package promrange

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

func TestFetcher_FetchRange(t *testing.T) {
	var gotQuery, gotStep string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		gotStep = r.URL.Query().Get("step")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"__name__":"http_requests_total","code":"500"},
					"values":[[1000,"1"],[1010,"3"]]
				}]
			}
		}`))
	}))
	defer ts.Close()

	f := New(ts.URL, `http_requests_total{code="500"}`, 10*time.Second)
	samples, err := f.FetchRange(context.Background(), time.Unix(1000, 0), time.Unix(1010, 0))
	require.NoError(t, err)
	assert.Equal(t, `http_requests_total{code="500"}`, gotQuery)
	assert.Equal(t, "10", gotStep)
	require.Len(t, samples, 2)
	assert.Equal(t, 1.0, samples[0].Values[`http_requests_total{code="500"}`])
	assert.Equal(t, 3.0, samples[1].Values[`http_requests_total{code="500"}`])
}

func TestParseResponse_FallbackKeyForUnnamedExpression(t *testing.T) {
	samples, err := parseResponse(strings.NewReader(`{
		"status":"success",
		"data":{
			"resultType":"matrix",
			"result":[{
				"metric":{"job":"api"},
				"values":[[1000,"0.5"]]
			}]
		}
	}`), "rate(errors[5m])")
	require.NoError(t, err)
	require.Len(t, samples, 1)
	assert.Equal(t, 0.5, samples[0].Values[`rate(errors[5m]){job="api"}`])
}

func TestFetcher_ImplementsWindowFetcher(t *testing.T) {
	var _ fetch.WindowFetcher = New("http://127.0.0.1:9090", "up", time.Second)
}
