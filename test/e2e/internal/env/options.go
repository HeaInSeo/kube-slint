package env

import (
	"path/filepath"
	"time"
)

// Options는 e2e 전용 설정임.
// pkg/slo와 독립적으로 유지되어야 함 (v1 레거시).
type Options struct {
	Enabled      bool
	ArtifactsDir string
	RunID        string

	SkipCleanup            bool
	SkipCertManagerInstall bool

	TokenRequestTimeout time.Duration
}

// Validate는 설정을 확인하고 기본값을 적용함.
func (o Options) Validate() Options {
	out := o
	if out.ArtifactsDir == "" {
		out.ArtifactsDir = "/tmp"
	}
	if out.TokenRequestTimeout == 0 {
		out.TokenRequestTimeout = 2 * time.Minute
	}
	return out
}

// SummaryPath는 요약 파일의 경로를 반환함.
func (o Options) SummaryPath(filename string) string {
	v := o.Validate()
	if filename == "" {
		filename = "sli-summary.json"
	}
	return filepath.Join(v.ArtifactsDir, filename)
}
