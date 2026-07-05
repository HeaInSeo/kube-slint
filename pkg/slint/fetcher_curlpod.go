package slint

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

// curlPodFetcher 는 curlpod를 실행하여 /metrics를 수집하는 fetcher 구현체임.
// fetch.SnapshotFetcher 를 구현하므로 Session.Start() 시점에 시작 스냅샷을 미리 캡처한다.
// 이를 통해 Gap G(engine.Execute()의 두 번의 Fetch() 모두 post-workload 상태를 반환하는 문제)를 해소함.
type curlPodFetcher struct {
	impl       *sessionImpl
	pod        *curlpod.CurlPod
	startCache *fetch.Sample // PreFetch() 성공 시 설정, 첫 번째 Fetch()에서 반환
	startErr   error         // PreFetch() 실패 시 첫 번째 Fetch()에서 신뢰 불가 상태로 전파
	fetchCount int           // Fetch() 호출 횟수 추적
}

func newCurlPodFetcher(impl *sessionImpl) fetch.MetricsFetcher {
	client := curlpod.New(nil, nil)
	// 필요한 안전 레이블을 추가함
	client.LabelSelector = fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id=%s", SanitizeKubernetesLabelValue(impl.RunID))
	// Apply TLS integration knob and dangerous security-boundary opt-ins.
	//nolint:staticcheck // SA1019: intentional bridge for the deprecated legacy field, kept for backward compatibility.
	client.TLSInsecureSkipVerify = impl.TLSInsecureSkipVerify
	client.DangerouslySkipTLSVerify = impl.DangerouslySkipTLSVerify
	client.DangerouslyAllowExternalMetricsURL = impl.DangerouslyAllowExternalMetricsURL
	client.DangerouslyAllowKubeSystemNamespace = impl.DangerouslyAllowKubeSystemNamespace

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

// PreFetch 는 측정 창 시작 시점의 스냅샷을 미리 캡처함.
// Session.Start()에서 호출되며, fetch.SnapshotFetcher 인터페이스를 구현함.
// 실패 시 startCache를 설정하지 않고, 첫 번째 Fetch()에서 실패를 전파한다.
func (f *curlPodFetcher) PreFetch(ctx context.Context) error {
	raw, err := f.pod.Run(ctx, f.impl.WaitPodDoneTimeout, f.impl.LogsTimeout)
	if err != nil {
		f.startErr = err
		return err
	}
	values, err := parsePrometheusText(raw)
	if err != nil {
		f.startErr = err
		return err
	}
	s := fetch.Sample{At: time.Now(), Values: values}
	f.startCache = &s
	f.startErr = nil
	return nil
}

// Fetch retrieves a metric sample.
// Fetch는 메트릭 샘플을 조회함.
// 첫 번째 호출 시 PreFetch()로 캐시된 시작 스냅샷이 있으면 그것을 반환하고, 이후 호출은 curlpod를 실행함.
func (f *curlPodFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	f.fetchCount++
	if f.fetchCount == 1 && f.startErr != nil {
		return fetch.Sample{}, fmt.Errorf("prefetch start snapshot failed: %w", f.startErr)
	}
	// 첫 번째 Fetch() 호출이고 startCache가 있으면 캐시된 시작 스냅샷을 반환함.
	// engine.Execute()가 첫 번째로 Fetch(startedAt)을 호출할 때 pre-workload 상태를 반환하게 됨.
	if f.fetchCount == 1 && f.startCache != nil {
		cached := *f.startCache
		cached.At = at
		return cached, nil
	}

	podCtx, cancel := context.WithTimeout(ctx, f.impl.podRunTimeout())
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
	return promtext.ParseTextToMapWithAggregates(strings.NewReader(raw))
}
