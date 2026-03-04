package promtext

import (
	"strings"
	"testing"
)

// BenchmarkParseTextToMap 은 실제 오퍼레이터 /metrics 응답과 유사한 입력을 사용함.
// controller-runtime 스타일: 레이블 있는 카운터 + 레이블 없는 게이지 혼합.
func BenchmarkParseTextToMap(b *testing.B) {
	const input = `# HELP controller_runtime_reconcile_total Total number of reconciliations
# TYPE controller_runtime_reconcile_total counter
controller_runtime_reconcile_total{controller="hello",result="success"} 42
controller_runtime_reconcile_total{controller="hello",result="error"} 3
controller_runtime_reconcile_total{controller="hello",result="requeue"} 1
# HELP controller_runtime_reconcile_errors_total Total reconciliation errors
# TYPE controller_runtime_reconcile_errors_total counter
controller_runtime_reconcile_errors_total{controller="hello"} 3
# HELP workqueue_queue_duration_seconds How long items sit in the queue
# TYPE workqueue_queue_duration_seconds histogram
workqueue_queue_duration_seconds_bucket{name="hello",le="0.001"} 10
workqueue_queue_duration_seconds_bucket{name="hello",le="0.01"} 20
workqueue_queue_duration_seconds_bucket{name="hello",le="+Inf"} 25
workqueue_queue_duration_seconds_sum{name="hello"} 0.12
workqueue_queue_duration_seconds_count{name="hello"} 25
# HELP process_resident_memory_bytes RSS in bytes
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 1.234e+07
`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseTextToMap(strings.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
	}
}
