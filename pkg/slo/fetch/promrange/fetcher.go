// Package promrange provides a WindowFetcher for Prometheus query_range API.
package promrange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/common/promkey"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
)

const (
	defaultHTTPTimeout = 30 * time.Second
	maxErrorBodyBytes  = 512
)

// Fetcher calls Prometheus /api/v1/query_range and converts matrix values into
// window samples. Each series becomes a key derived from its metric labels.
type Fetcher struct {
	BaseURL string
	Query   string
	Step    time.Duration
	Headers map[string]string
	Client  *http.Client
}

// New returns a query_range fetcher for a Prometheus base URL and query.
func New(baseURL, query string, step time.Duration) *Fetcher {
	return &Fetcher{BaseURL: baseURL, Query: query, Step: step}
}

// FetchRange implements fetch.WindowFetcher.
func (f *Fetcher) FetchRange(ctx context.Context, start, end time.Time) ([]fetch.Sample, error) {
	u, err := f.queryURL(start, end)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("promrange: build request: %w", err)
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
		return nil, fmt.Errorf("promrange: GET %s: %w", u, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("promrange: GET %s: status %d: %s", u, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseResponse(resp.Body, f.Query)
}

func (f *Fetcher) queryURL(start, end time.Time) (string, error) {
	if strings.TrimSpace(f.BaseURL) == "" {
		return "", fmt.Errorf("promrange: BaseURL is required")
	}
	if strings.TrimSpace(f.Query) == "" {
		return "", fmt.Errorf("promrange: Query is required")
	}
	step := f.Step
	if step <= 0 {
		return "", fmt.Errorf("promrange: Step must be positive")
	}
	base, err := url.Parse(strings.TrimRight(f.BaseURL, "/") + "/api/v1/query_range")
	if err != nil {
		return "", fmt.Errorf("promrange: parse BaseURL: %w", err)
	}
	q := base.Query()
	q.Set("query", f.Query)
	q.Set("start", formatUnixSeconds(start))
	q.Set("end", formatUnixSeconds(end))
	q.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))
	base.RawQuery = q.Encode()
	return base.String(), nil
}

func formatUnixSeconds(t time.Time) string {
	return strconv.FormatFloat(float64(t.UnixNano())/1e9, 'f', -1, 64)
}

type apiResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		ResultType string      `json:"resultType"`
		Result     []apiSeries `json:"result"`
	} `json:"data"`
}

type apiSeries struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

func parseResponse(r io.Reader, fallbackKey string) ([]fetch.Sample, error) {
	var resp apiResponse
	dec := json.NewDecoder(r)
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Status != "success" {
		if resp.Error == "" {
			resp.Error = "unknown error"
		}
		return nil, fmt.Errorf("prometheus status %q: %s", resp.Status, resp.Error)
	}
	if resp.Data.ResultType != "" && resp.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("prometheus resultType %q is not matrix", resp.Data.ResultType)
	}

	byTime := map[int64]map[string]float64{}
	for _, series := range resp.Data.Result {
		key := seriesKey(series.Metric, fallbackKey)
		for _, pair := range series.Values {
			if len(pair) != 2 {
				continue
			}
			ts, ok := numberToFloat(pair[0])
			if !ok {
				continue
			}
			val, ok := valueToFloat(pair[1])
			if !ok {
				continue
			}
			ns := int64(ts * 1e9)
			if byTime[ns] == nil {
				byTime[ns] = map[string]float64{}
			}
			byTime[ns][key] = val
		}
	}

	timestamps := make([]int64, 0, len(byTime))
	for ts := range byTime {
		timestamps = append(timestamps, ts)
	}
	sortInt64s(timestamps)

	samples := make([]fetch.Sample, 0, len(timestamps))
	for _, ts := range timestamps {
		samples = append(samples, fetch.Sample{At: time.Unix(0, ts), Values: byTime[ts]})
	}
	return samples, nil
}

func seriesKey(metric map[string]string, fallback string) string {
	name := metric["__name__"]
	labels := map[string]string{}
	for k, v := range metric {
		if k == "__name__" {
			continue
		}
		labels[k] = v
	}
	if name == "" {
		name = fallback
	}
	if len(labels) == 0 {
		return name
	}
	return promkey.Format(name, labels)
}

func numberToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case float64:
		return x, true
	default:
		return 0, false
	}
}

func valueToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case float64:
		return x, true
	default:
		return 0, false
	}
}

func sortInt64s(v []int64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
