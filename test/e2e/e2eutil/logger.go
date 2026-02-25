package e2eutil

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
)

// 사용 예시
// logger := slo.NewLogger(utils.GinkgoLog) // nil이면 noop
// logger.Logf("hello %s", "world")

// GinkgoLogger는 slo.Logger를 GinkgoWriter에 맞게 어댑터함.
type GinkgoLogger struct{}

// Logf는 포맷된 문자열을 로그로 남김.
func (GinkgoLogger) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, format+"\n", args...)
}

// 컴파일 타임 검사
var _ slo.Logger = (*GinkgoLogger)(nil)

// GinkgoLog는 테스트 파일 전용의 즉시 사용 가능한 인스턴스임.
var GinkgoLog slo.Logger = GinkgoLogger{}
