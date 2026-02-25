package fetch

import (
	"context"
	"time"
)

// Sample은 특정 시점의 스냅샷 하나임.
type Sample struct {
	At     time.Time
	Values map[string]float64 // metricKey -> value
}

// MetricsFetcher는 하나의 스냅샷을 가져오며, 구체적인 방법은 구현체에서 결정함.
// - outside: curl /metrics (Pod, port-forward, HTTP를 통해)
// - inside: localhost로 직접 HTTP 요청
// - trigger: /metrics 또는 status/log 파생 메트릭에서 가져옴
type MetricsFetcher interface {
	Fetch(ctx context.Context, at time.Time) (Sample, error)
}
