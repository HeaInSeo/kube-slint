package harness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/curlpod"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promtext"
)

// ---- Default Fetcher: curlpod + promtext ----

type curlPodFetcher struct {
	impl *sessionImpl
	pod  *curlpod.CurlPod
}

func newCurlPodFetcher(impl *sessionImpl) fetch.MetricsFetcher {
	client := curlpod.New(nil, nil)
	// 필요한 안전 레이블을 추가함
	client.LabelSelector = fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id=%s", impl.RunID)
	// Apply TLS integration knob
	client.TLSInsecureSkipVerify = impl.TLSInsecureSkipVerify

	return &curlPodFetcher{
		impl: impl,
		pod: &curlpod.CurlPod{
			Client:             client,
			Namespace:          impl.Config.Namespace,
			MetricsServiceName: impl.Config.MetricsServiceName,
			ServiceAccountName: impl.Config.ServiceAccountName,
			Token:              impl.Config.Token,
			Image:              impl.CurlImage,
			ServiceURLFormat:   impl.ServiceURLFormat,
		},
	}
}

// Fetch retrieves a metric sample.
// Fetch는 메트릭 샘플을 조회함.
func (f *curlPodFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	podCtx, cancel := context.WithTimeout(ctx, f.impl.ScrapeTimeout)
	defer cancel()

	raw, err := f.pod.Run(podCtx, f.impl.WaitPodDoneTimeout, f.impl.LogsTimeout)
	if err != nil {
		return fetch.Sample{}, err
	}

	values, err := parsePrometheusText(raw)
	if err != nil {
		return fetch.Sample{}, err
	}

	return fetch.Sample{
		At:     at,
		Values: values,
	}, nil
}

func parsePrometheusText(raw string) (map[string]float64, error) {
	base, err := promtext.ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		return nil, err
	}

	out := map[string]float64{}
	for key, val := range base {
		out[key] = val
		if idx := strings.Index(key, "{"); idx > 0 {
			name := key[:idx]
			out[name] = out[name] + val
		}
	}
	return out, nil
}
