package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/engine"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/HeaInSeo/kube-slint/pkg/slo/tags"
)

// SessionConfig contains session inputs and defaults.
// SessionConfig는 세션 입력값과 기본값을 포함함.
type SessionConfig struct {
	Namespace          string
	MetricsServiceName string
	TestCase           string
	Suite              string

	// Optional inputs
	RunID string
	Tags  map[string]string

	ServiceAccountName string
	Token              string
	ArtifactsDir       string

	Method engine.MeasurementMethod
	Now    func() time.Time

	// Optional overrides
	Specs   []spec.SLISpec
	Fetcher fetch.MetricsFetcher
	Writer  summary.Writer

	// Real-cluster Integration Knobs
	CurlImage             string // overrides default curl image
	TLSInsecureSkipVerify bool   // true to skip self-signed cert verification (pass -k to curl)

	// Internal metadata
	ConfigSourceType string
	ConfigSourcePath string

	StrictnessMode     string // BestEffort | StrictCollection | StrictEvaluation
	MaxStartSkewMs     int64
	MaxEndSkewMs       int64
	MaxScrapeLatencyMs int64

	GateOnLevel    string // none | warn | fail
	CleanupEnabled bool
	CleanupMode    string // always | on-success | on-failure | manual
}

type sessionImpl struct {
	Config SessionConfig

	// Tunables (defaults are set in NewSession)
	ServiceURLFormat      string
	CurlImage             string
	TLSInsecureSkipVerify bool

	ScrapeTimeout      time.Duration
	WaitPodDoneTimeout time.Duration
	LogsTimeout        time.Duration

	// Normalized (derived) runtime values
	RunID string
	Tags  map[string]string

	Warnings []string

	specs   []spec.SLISpec
	fetcher fetch.MetricsFetcher
	writer  summary.Writer

	started   time.Time
	hasFailed bool
}

// Session 은 측정 세션을 관리함.
type Session struct {
	impl *sessionImpl
}

// NewSession builds a session with defaults applied.
// NewSession은 기본값이 적용된 세션을 생성함.
func NewSession(cfg SessionConfig) *Session {
	cfg = applySessionBaseDefaults(cfg)
	cfg = discoverAndApplyConfig(cfg)

	autoTags := tags.AutoTags(tags.AutoTagsInput{
		Suite:     cfg.Suite,
		TestCase:  cfg.TestCase,
		Namespace: cfg.Namespace,
		RunID:     cfg.RunID,
	})

	mergedTags := tags.MergeTags(cfg.Tags, autoTags)
	resolvedSpecs := defaultSpecs(cfg.Specs)

	w := cfg.Writer
	if w == nil {
		w = summary.NewJSONFileWriter()
	}

	impl := &sessionImpl{
		Config: cfg,

		ServiceURLFormat: "https://%s.%s.svc:8443/metrics",
		CurlImage:        "curlimages/curl:latest",

		ScrapeTimeout:      2 * time.Minute,
		WaitPodDoneTimeout: 5 * time.Minute,
		LogsTimeout:        2 * time.Minute,

		RunID: cfg.RunID,
		Tags:  mergedTags,

		specs:   resolvedSpecs,
		fetcher: cfg.Fetcher,
		writer:  w,
	}

	if cfg.CurlImage != "" {
		impl.CurlImage = cfg.CurlImage
	}
	impl.TLSInsecureSkipVerify = cfg.TLSInsecureSkipVerify

	return &Session{impl: impl}
}

func applySessionBaseDefaults(cfg SessionConfig) SessionConfig {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Method == "" {
		cfg.Method = engine.InsideSnapshot
	}

	runID := strings.TrimSpace(cfg.RunID)
	if runID == "" {
		runID = fmt.Sprintf("local-%d", cfg.Now().Unix())
	}
	cfg.RunID = runID
	return cfg
}

func discoverAndApplyConfig(cfg SessionConfig) SessionConfig {
	discoveredCfg, source, err := DiscoverConfig("")
	if err != nil {
		fmt.Printf("kube-slint [discovery]: warning - %v\n", err)
	}

	cfg.ConfigSourceType = source.Type
	cfg.ConfigSourcePath = source.Path

	if source.Disabled {
		fmt.Println("kube-slint [discovery]: discovery disabled via SLINT_DISABLE_DISCOVERY")
	} else {
		if source.Path != "" {
			fmt.Printf("kube-slint [discovery]: resolved config source type=%s path=%s\n", source.Type, source.Path)
		} else {
			fmt.Printf("kube-slint [discovery]: resolved config source type=%s\n", source.Type)
		}
	}

	if discoveredCfg != nil {
		cfg = mergeDiscoveredConfig(cfg, discoveredCfg)
	}
	return cfg
}

func mergeDiscoveredConfig(cfg SessionConfig, d *DiscoveredConfig) SessionConfig {
	if d.Write.ArtifactsDir != "" && cfg.ArtifactsDir == "" {
		cfg.ArtifactsDir = d.Write.ArtifactsDir
	}
	if d.Strictness.Mode != "" {
		cfg.StrictnessMode = d.Strictness.Mode
	}
	if d.Strictness.Thresholds.MaxStartSkewMs > 0 {
		cfg.MaxStartSkewMs = d.Strictness.Thresholds.MaxStartSkewMs
	}
	if d.Strictness.Thresholds.MaxEndSkewMs > 0 {
		cfg.MaxEndSkewMs = d.Strictness.Thresholds.MaxEndSkewMs
	}
	if d.Strictness.Thresholds.MaxScrapeLatencyMs > 0 {
		cfg.MaxScrapeLatencyMs = d.Strictness.Thresholds.MaxScrapeLatencyMs
	}
	if d.Gating.GateOnLevel != "" {
		cfg.GateOnLevel = d.Gating.GateOnLevel
	}
	cfg.CleanupEnabled = d.Cleanup.Enabled
	if d.Cleanup.Mode != "" {
		cfg.CleanupMode = d.Cleanup.Mode
	}
	return cfg
}

// reset은 전체 Session 구조체를 복사하지 않고 내부 런타임 상태를 교체함.
// 참고: 깊은 복사(deep copy)가 아니므로 reset 호출 후 from을 사용해서는 안 됨.
func (s *Session) reset(from *Session) {
	if s == nil {
		return
	}

	if from == nil {
		s.impl = nil
		return
	}

	s.impl = from.impl
}

// ShouldWriteArtifacts 는 세션이 요약 출력을 기록할지 여부를 보고함.
func (s *Session) ShouldWriteArtifacts() bool {
	// Ginkgo 훅과 placeholder 패턴 때문에 s != nil 체크를 추가하여 패닉을 방지함.
	return s != nil && s.impl != nil && s.impl.Config.ArtifactsDir != ""
}

// NextSummaryPath 는 충돌 시 -<n>을 추가하여 고유한 요약 경로를 반환함.
func (s *Session) NextSummaryPath(filename string) (string, error) {
	// s == nil 체크는 현재 구조상 예외적으로 필요하므로 포함됨.
	if s == nil || s.impl == nil {
		return "", fmt.Errorf("harness: session not initialized")
	}

	if strings.TrimSpace(s.impl.Config.ArtifactsDir) == "" {
		return "", nil
	}

	base := filepath.Join(s.impl.Config.ArtifactsDir, filename)
	path := base
	for i := 1; ; i++ {
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return path, nil
			}
			return "", err
		}
		path = fmt.Sprintf("%s-%d", base, i)
	}
}

// AddWarning 은 BestEffort 모드에 대한 경고 메시지를 기록함.
func (s *Session) AddWarning(message string) {
	if s == nil || s.impl == nil || strings.TrimSpace(message) == "" {
		return
	}
	s.impl.Warnings = append(s.impl.Warnings, message)
}

// MarkFailed 는 세션을 명시적으로 실패 상태로 플래그하여 CleanupMode 평가에 영향을 줌.
// 단조적(한 번 실패로 표시되면 되돌릴 수 없음)이며 멱등성을 가짐.
func (s *Session) MarkFailed() {
	if s == nil || s.impl == nil {
		return
	}
	s.impl.hasFailed = true
}

// Cleanup 은 세션에서 생성된 임시 리소스를 제거함.
// 광범위한 삭제를 방지하기 위해 run-id와 namespace를 안전 장치로 사용함.
func (s *Session) Cleanup(ctx context.Context) {
	if s == nil || s.impl == nil {
		return
	}

	shouldRun := shouldRunCleanup(s.impl.Config.CleanupMode, s.impl.Config.CleanupEnabled, s.impl.hasFailed)
	if !shouldRun {
		return
	}

	ns := s.impl.Config.Namespace
	runID := s.impl.RunID

	if ns == "" || runID == "" {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - missing namespace or run-id (ns=%q, runID=%q)\n", ns, runID)
		return
	}

	runCleanupActions(ctx, ns, runID)
}

func shouldRunCleanup(mode string, enabled, hasFailed bool) bool {
	if mode == "manual" {
		return false
	}

	if mode == "" {
		if enabled {
			mode = "always"
		} else {
			return false
		}
	}

	if mode == "on-success" && hasFailed {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - mode is %s and test failed\n", mode)
		return false
	}
	if mode == "on-failure" && !hasFailed {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - mode is %s and test succeeded\n", mode)
		return false
	}

	return true
}

func runCleanupActions(ctx context.Context, ns, runID string) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id=%s", runID)

	cmd := exec.CommandContext(ctx, "kubectl", "delete", "pod", "-n", ns, "-l", labelSelector, "--ignore-not-found=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("kube-slint [cleanup]: failed for run-id %s: %v (output: %s)\n", runID, err, string(out))
	} else {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			fmt.Printf("kube-slint [cleanup]: success for run-id %s: %s\n", runID, outStr)
		}
	}
}

// Start begins measurement.
// Start는 측정을 시작함.
func (s *Session) Start() {
	if s == nil || s.impl == nil {
		return
	}
	// 방어 코드: nil 또는 유효하지 않은 Now() 주입 처리
	now := s.impl.Config.Now
	if now == nil {
		now = time.Now
	}

	t := now()
	if t.IsZero() {
		t = time.Now()
	}
	s.impl.started = t
}

// End는 측정을 완료함.

// End concludes the measurement session.
func (s *Session) End(ctx context.Context) (*summary.Summary, error) {
	if s == nil || s.impl == nil {
		return nil, fmt.Errorf("harness: session not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	started := s.impl.started
	if started.IsZero() {
		return nil, fmt.Errorf("harness: Start() was not called (RunID=%q TestCase=%q)", s.impl.RunID, s.impl.Config.TestCase)

	}

	// 중요: 시간 동기화 및 주입 시간은 별도로 논의되고 문서화되어야 함
	now := s.impl.Config.Now
	if now == nil {
		now = time.Now
	}
	finished := now()

	m := s.impl.Config.Method
	if m == "" {
		m = engine.InsideSnapshot
	}

	fetcher := s.impl.fetcher
	if fetcher == nil {
		fetcher = newCurlPodFetcher(s.impl)
	}

	eng := engine.New(fetcher, s.impl.writer, nil)

	outPath := ""
	if s.ShouldWriteArtifacts() {
		filename := fmt.Sprintf(
			"sli-summary.%s.%s.json",
			SanitizeFilename(s.impl.RunID),
			SanitizeFilename(s.impl.Config.TestCase),
		)
		path, err := s.NextSummaryPath(filename)
		if err != nil {
			return nil, err
		}
		outPath = path
	}

	rel := &summary.Reliability{
		ConfigSourceType: s.impl.Config.ConfigSourceType,
		ConfigSourcePath: s.impl.Config.ConfigSourcePath,
	}

	sum, err := engine.ExecuteStandard(ctx, eng, engine.ExecuteRequestStandard{
		Method: m,
		Config: engine.RunConfig{
			RunID:      s.impl.RunID,
			StartedAt:  started,
			FinishedAt: finished,
			Format:     "v4.4",
			Tags:       s.impl.Tags,
		},
		Specs:       s.impl.specs,
		OutPath:     outPath,
		Reliability: rel,
	})

	if err != nil {
		s.impl.hasFailed = true
		return sum, err
	}

	// 1. Strictness Check (파이프라인 신뢰도 검증)
	if strictErr := CheckStrictness(s.impl.Config, sum); strictErr != nil {
		s.impl.hasFailed = true
		return sum, strictErr
	}

	// 2. Gating Check (정상 결과 승격 검증)
	if gatingErr := CheckGating(s.impl.Config, sum); gatingErr != nil {
		s.impl.hasFailed = true
		return sum, gatingErr
	}

	return sum, nil
}

func defaultSpecs(specs []spec.SLISpec) []spec.SLISpec {
	if specs != nil {
		return specs
	}

	return DefaultV3Specs()
}
