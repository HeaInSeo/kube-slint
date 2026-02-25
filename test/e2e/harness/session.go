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
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/curlpod"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promtext"
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
	ServiceURLFormat string
	// Step 6 후보. 구성 확장 여지 있음
	CurlImage string
	// ServiceURLFormat 에서 결정됨. 일단 주석으로 남겨둠, 혹시 필요하면 살림.
	//MetricsPort      int
	// 추후
	// type SessionConfig struct {
	//     MetricsScheme string // "https"
	//     MetricsPort   int    // 8443
	//     MetricsPath   string // "/metrics"
	// } 이런식으로 필요할지 생각해보자. 일단 주석으로 남겨둠.

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

type Session struct {
	impl *sessionImpl
}

// NewSession builds a session with defaults applied.
// NewSession은 기본값이 적용된 세션을 생성함.
func NewSession(cfg SessionConfig) *Session {
	// Defaults
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	// InsideSnapshot 기본 사용. 오류 처리 로직 추가 검토 (Step 6 후보)
	if cfg.Method == "" {
		cfg.Method = engine.InsideSnapshot
	}

	runID := strings.TrimSpace(cfg.RunID)
	if runID == "" {
		runID = fmt.Sprintf("local-%d", cfg.Now().Unix())
	}

	// 1. Discover Configuration (Bridge Sprint)
	// 1. 설정 자동 탐색 (브리지 스프린트)
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

	// 2. Apply Discovered Configuration onto SessionConfig (Thin Wrapper)
	// 2. 탐색된 설정을 SessionConfig에 반영 (얇은 래퍼)
	if discoveredCfg != nil {
		if discoveredCfg.Write.ArtifactsDir != "" && cfg.ArtifactsDir == "" {
			cfg.ArtifactsDir = discoveredCfg.Write.ArtifactsDir
		}
		if discoveredCfg.Strictness.Mode != "" {
			cfg.StrictnessMode = discoveredCfg.Strictness.Mode
		}
		if discoveredCfg.Strictness.Thresholds.MaxStartSkewMs > 0 {
			cfg.MaxStartSkewMs = discoveredCfg.Strictness.Thresholds.MaxStartSkewMs
		}
		if discoveredCfg.Strictness.Thresholds.MaxEndSkewMs > 0 {
			cfg.MaxEndSkewMs = discoveredCfg.Strictness.Thresholds.MaxEndSkewMs
		}
		if discoveredCfg.Strictness.Thresholds.MaxScrapeLatencyMs > 0 {
			cfg.MaxScrapeLatencyMs = discoveredCfg.Strictness.Thresholds.MaxScrapeLatencyMs
		}

		if discoveredCfg.Gating.GateOnLevel != "" {
			cfg.GateOnLevel = discoveredCfg.Gating.GateOnLevel
		}
		// If explicitly set
		cfg.CleanupEnabled = discoveredCfg.Cleanup.Enabled
		if discoveredCfg.Cleanup.Mode != "" {
			cfg.CleanupMode = discoveredCfg.Cleanup.Mode
		}
	}

	autoTags := tags.AutoTags(tags.AutoTagsInput{
		Suite:     cfg.Suite,
		TestCase:  cfg.TestCase,
		Namespace: cfg.Namespace,
		RunID:     runID,
	})

	mergedTags := tags.MergeTags(cfg.Tags, autoTags)

	// Specs 기본값 확인 및 개선 (Step 6 후보)
	resolvedSpecs := defaultSpecs(cfg.Specs)

	// Writer default
	w := cfg.Writer
	if w == nil {
		w = summary.NewJSONFileWriter()
	}

	impl := &sessionImpl{
		Config: cfg,

		//MetricsPort:      8443,
		ServiceURLFormat: "https://%s.%s.svc:8443/metrics",
		CurlImage:        "curlimages/curl:latest",

		ScrapeTimeout:      2 * time.Minute,
		WaitPodDoneTimeout: 5 * time.Minute,
		LogsTimeout:        2 * time.Minute,

		RunID: runID,
		Tags:  mergedTags,

		specs:   resolvedSpecs,
		fetcher: cfg.Fetcher,
		writer:  w,
	}

	return &Session{impl: impl}
}

// reset replaces internal runtime state WITHOUT copying the whole Session struct.
// reset은 전체 Session 구조체를 복사하지 않고 내부 런타임 상태를 교체함.
// NOTE: This is NOT a deep copy. from must not be used after calling reset.
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

// ShouldWriteArtifacts reports whether session should write summary output.
// ShouldWriteArtifacts는 세션이 요약 출력을 기록해야 하는지 여부를 보고함.
func (s *Session) ShouldWriteArtifacts() bool {
	// 여기서, s != nil 체크를 하는 이유는 Ginkgo 훅 + placeholder 패턴 때문에, 사용될 수 있기때문에 panic 방지용으로 넣음.
	return s != nil && s.impl != nil && s.impl.Config.ArtifactsDir != ""
}

// NextSummaryPath returns a unique summary path by appending -<n> on collisions.
// NextSummaryPath는 충돌 시 -<n>을 추가하여 고유한 요약 경로를 반환함.
func (s *Session) NextSummaryPath(filename string) (string, error) {
	// s == nil 체크는 일부러 하는 것이다. 일반적으로는 필요없지만 지금 케이스에서는 필요하다.
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

// AddWarning records a warning message for BestEffort mode.
// AddWarning은 BestEffort 모드에 대한 경고 메시지를 기록함.
func (s *Session) AddWarning(message string) {
	if s == nil || s.impl == nil || strings.TrimSpace(message) == "" {
		return
	}
	s.impl.Warnings = append(s.impl.Warnings, message)
}

// MarkFailed explicitly flags this session as failed, affecting CleanupMode evaluations.
// It is monotonic (once failed, cannot be reverted) and idempotent.
// MarkFailed는 세션을 명시적으로 실패 상태로 플래그하여 CleanupMode 평가에 영향을 줍니다.
// 단조적(한 번 실패로 표시되면 되돌릴 수 없음)이며 멱등성을 가집니다.
func (s *Session) MarkFailed() {
	if s == nil || s.impl == nil {
		return
	}
	s.impl.hasFailed = true
}

// OrphanSweepOptions configures the orphan sweep behavior.
// OrphanSweepOptions는 고아(orphan) 리소스 정리 동작을 설정함.
type OrphanSweepOptions struct {
	Enabled bool
	Mode    string // "report-only" (기본값) | "delete"
}

// SweepOrphans detects and optionally deletes resources from previous kube-slint run-ids.
// It explicitly excludes the current session's run-id and limits scope to the current namespace.
// SweepOrphans는 이전 kube-slint run-id의 리소스를 탐지하고 선택적으로 삭제함.
// 현재 세션의 run-id는 명시적으로 제외되며, 현재 namespace로 범위가 제한됨.
func (s *Session) SweepOrphans(ctx context.Context, opts OrphanSweepOptions) error {
	if s == nil || s.impl == nil {
		return nil
	}
	if !opts.Enabled {
		return nil
	}

	mode := opts.Mode
	if mode != "delete" {
		mode = "report-only" // Default to safety
	}

	ns := s.impl.Config.Namespace
	runID := s.impl.RunID

	if ns == "" || runID == "" {
		fmt.Printf("kube-slint [orphan-sweep]: skip - missing namespace or run-id for safety guard\n")
		return nil
	}

	// Find orphaned pods managed by kube-slint, excluding current run-id
	labelSelector := fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id!=%s", runID)

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", ns, "-l", labelSelector, "-o", "jsonpath={.items[*].metadata.name}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list orphans: %v (output: %s)", err, string(out))
	}

	podsStr := strings.TrimSpace(string(out))
	if podsStr == "" {
		fmt.Printf("kube-slint [orphan-sweep]: no orphan resources detected\n")
		return nil
	}

	orphans := strings.Split(podsStr, " ")
	fmt.Printf("kube-slint [orphan-sweep]: detected %d orphan(s): %v\n", len(orphans), orphans)

	if mode == "delete" {
		fmt.Printf("kube-slint [orphan-sweep]: proceeding with deletion for %d orphan(s)...\n", len(orphans))
		delCmd := exec.CommandContext(ctx, "kubectl", "delete", "pods", "-n", ns, "-l", labelSelector, "--ignore-not-found=true")
		delOut, delErr := delCmd.CombinedOutput()
		if delErr != nil {
			return fmt.Errorf("failed to delete orphans: %v (output: %s)", delErr, string(delOut))
		}
		fmt.Printf("kube-slint [orphan-sweep]: deletion complete\n")
	} else {
		fmt.Printf("kube-slint [orphan-sweep]: report-only mode, skipped deletion. To delete, set option mode='delete'.\n")
	}

	return nil
}

// Cleanup removes temporary resources created by the session.
// It uses run-id and namespace as safety guards to prevent broad deletion.
// Cleanup은 세션에서 생성된 임시 리소스를 제거합니다.
// 광범위한 삭제를 방지하기 위해 run-id와 namespace를 안전 장치로 사용합니다.
func (s *Session) Cleanup(ctx context.Context) {
	if s == nil || s.impl == nil {
		return
	}

	mode := s.impl.Config.CleanupMode
	if mode == "manual" {
		return
	}

	// Resolve implicitly enabled state if not explicitly disabled or set via manual mode
	if mode == "" {
		if s.impl.Config.CleanupEnabled {
			mode = "always"
		} else {
			// fallback default when nothing is specified is manual/none for safety
			return
		}
	}

	if mode == "on-success" && s.impl.hasFailed {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - mode is %s and test failed\n", mode)
		return
	}
	if mode == "on-failure" && !s.impl.hasFailed {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - mode is %s and test succeeded\n", mode)
		return
	}

	ns := s.impl.Config.Namespace
	runID := s.impl.RunID

	if ns == "" || runID == "" {
		fmt.Printf("kube-slint [cleanup]: skip cleanup - missing namespace or run-id (ns=%q, runID=%q)\n", ns, runID)
		return
	}

	// Always restrict by namespace and run-id
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
	// Defensive: handle nil/invalid Now() injection.
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
// TODO(harness-thin):
//  - harness는 "연결자(glue)"로 얇아야 한다. 현재는 curlpod/promtext 구현까지 포함되어 두텁다.
//  - 목표: Method/FetchStrategy(또는 Source) 선택은 e2e_test.go(provider)가 결정하고,
//    harness는 검증/시간윈도우(StartedAt/FinishedAt)/Engine 실행 + artifact 경로 계산만 담당한다.
//  - 조치:
//    1) newCurlPodFetcher/curlPodFetcher/parsePrometheusText를 harness 밖(예: harness/source/curlpod 또는 test-side adapter)으로 이동.
//    2) End()에서 "fetcher nil이면 curlpod" 같은 암묵 기본값 제거(또는 NewSession 단계에서만 기본값 처리).
//    3) Method는 Config.Method를 그대로 사용(하드코딩 금지)하고, 장기적으로는 Method도 필수 입력으로 강제.
//    4) (선택) Source.Build(ctx) -> fetcher 생성 패턴 도입하여, K8s 의존 구현은 adapter에 격리.

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

	// Important: time synchronization and injection time should be discussed separately and documented.
	now := s.impl.Config.Now
	if now == nil {
		now = time.Now
	}
	finished := now()

	// TODO(harness-thin):
	//  - harness는 "연결자(glue)"로 얇아야 한다. 현재는 curlpod/promtext 구현까지 포함되어 두텁다.
	//  - 목표: Method/FetchStrategy(또는 Source) 선택은 e2e_test.go(provider)가 결정하고,
	//    harness는 검증/시간윈도우(StartedAt/FinishedAt)/Engine 실행 + artifact 경로 계산만 담당한다.
	//  - 조치:
	//    1) newCurlPodFetcher/curlPodFetcher/parsePrometheusText를 harness 밖으로 이동.
	//    2) End()의 fetcher fallback 정책 정리(장기적으로는 명시 주입).
	//    3) Method 하드코딩 금지, 장기적으로 Method 필수화.
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

// ---- (TEMP) Default Fetcher: curlpod + promtext ----
// Step 6 후보. 아래 구현은 pkg/slo/fetch/* 또는 test-side adapter로 이동 예정.

type curlPodFetcher struct {
	impl *sessionImpl
	pod  *curlpod.CurlPod
}

func newCurlPodFetcher(impl *sessionImpl) fetch.MetricsFetcher {
	client := curlpod.New(nil, nil)
	// Add required safety labels
	client.LabelSelector = fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id=%s", impl.RunID)

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

func defaultSpecs(specs []spec.SLISpec) []spec.SLISpec {
	if specs != nil {
		return specs
	}
	// Step 6 후보. DefaultSpecs 개선 또는 파일 로드 방식으로 대체 예정.
	return DefaultV3Specs()
}
