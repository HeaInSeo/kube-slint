# kube-slint — 한국 공개SW 개발자대회 제출 문서

---

## 1. 프로젝트 개요

kube-slint는 쿠버네티스 오퍼레이터 E2E 테스트 세션에 내장하는 **shift-left 운영 SLI 가드레일 라이브러리**입니다. 기본 경로는 클러스터 내부의 임시 curl pod로 `/metrics`를 스크랩하지만, 계측 경계는 source-neutral하게 설계되어 port-forward, HTTP JSON/expvar, Prometheus range query 같은 다른 소스도 동일한 SLI 계산과 정책 게이트에 연결할 수 있습니다. reconcile rate, workqueue depth, REST client error 등의 운영 지표를 코드 리뷰 단계에서 측정하고 선언적 `policy.yaml`로 CI 게이트 판정을 내리며, 기능 정확성을 검증하는 E2E 테스트와는 독립적으로 동작합니다.

**현재 구현 기준**: v1.7 계열 기능 세트에는 JSON/expvar `SnapshotFetcher`, `WindowFetcher`/`promrange`, `window_min`/`window_max`/`window_avg`/`window_p95`/`window_p99`/`window_ratio`, strict coverage governance(`coverage.required`, `coverage.informational`, `coverage_gap`)가 포함되어 있습니다. 현재 `main`은 이 기능 세트를 기반으로 문서·CI·실사용 hardening을 이어가는 상태입니다.

---

## 2. 개발 동기

쿠버네티스 오퍼레이터 개발 주기에는 구조적인 관측 공백이 존재합니다.

기존 E2E 테스트는 "Custom Resource가 Ready 상태가 됐는가?"라는 정확성(correctness)만 검증합니다. 반면 "reconcile 루프가 몇 번 실행됐는가?", "workqueue에 항목이 쌓이고 있는가?", "API 서버 에러가 급증했는가?" 같은 운영 건전성 지표는 통상적으로 프로덕션 Prometheus/Grafana 스택에서만 확인됩니다.

이 구조는 두 가지 문제를 낳습니다.

첫째, 운영 SLI 회귀가 프로덕션 배포 이후에야 발견됩니다. PR 리뷰 단계에서 reconcile 오류율이 두 배로 늘어나는 변경이 통과되더라도, 모니터링 대시보드에서 경보가 울리기 전까지 아무도 알아채지 못합니다.

둘째, 오퍼레이터 코드에 계측 로직을 삽입하면 관심사 분리가 깨집니다. 컨트롤러의 Reconcile 함수에 SLI 측정 코드가 섞이면 유지보수성이 저하되고, 계측 버그가 조정(reconciliation) 동작에 영향을 줄 위험이 생깁니다.

kube-slint는 두 문제를 동시에 해결합니다. 오퍼레이터 코드를 전혀 수정하지 않으면서, CI 파이프라인에서 실행 중인 테스트 세션에 SLI 측정을 붙여 운영 지표를 코드 리뷰 단계(shift-left)로 끌어올립니다.

---

## 3. 주요 기능

| 기능 | 설명 |
|---|---|
| 오퍼레이터 코드 무수정 | 기본 curl pod 또는 port-forward 등 외부 계측 경로를 사용하므로 오퍼레이터에 kube-slint 계측 코드를 삽입하지 않음 |
| source-neutral 계측 경계 | `MetricsFetcher` / `SnapshotFetcher` / `WindowFetcher` 인터페이스로 point, snapshot, range/window 소스를 동일한 엔진에 연결 |
| 선언적 SLI 스펙 | `SLISpec` 구조체로 측정 대상, 계산 방식(delta / start / end / window aggregation), 판정 규칙을 선언 |
| 기본 제공 스펙 세트 | reconcile total/success/error delta, workqueue depth/adds/retries, REST client requests/429/5xx — kubebuilder 오퍼레이터에 즉시 적용 가능 |
| JSON/expvar 소스 | `pkg/slo/fetch/jsonendpoint`가 HTTP JSON의 숫자 leaf를 dot-separated key로 평탄화해 snapshot SLI 입력으로 사용 |
| Prometheus range/window 소스 | `pkg/slo/fetch/promrange`와 `WindowFetcher`로 query_range 결과를 `window_min/max/avg/p95/p99/ratio` 계산에 사용 |
| 임계치 기반 게이팅 | `policy.yaml`의 threshold 규칙으로 절대값 기준 PASS/WARN/FAIL 판정 |
| 회귀 감지 | 이전 실행의 baseline JSON과 비교하여 허용 오차(`tolerance_percent`) 초과 시 회귀 감지 |
| coverage governance | 측정된 scalar SLI가 threshold 또는 `coverage.informational`로 분류되지 않으면 `coverage_gap`으로 판정 가능 |
| 신뢰도 점수 | 스크랩 지연, skew, 누락 입력을 반영한 0.0–1.0 `confidenceScore` 자동 산출 |
| 구조화된 아티팩트 | `sli-summary.json` (측정 결과), `slint-gate-summary.json` (게이트 판정) — JSON 스키마 고정 |
| GitHub Actions 통합 | Composite Action으로 CI 게이팅; `$GITHUB_STEP_SUMMARY` 마크다운 렌더링 지원 |
| non-fatal 원칙 | 계측 실패 자체는 correctness 테스트를 중단시키지 않고 Reliability/NO_GRADE 경로로 드러냄 |
| dual-write 전략 | 감사 추적용 unique 파일 + slint-gate 기본 입력용 static alias 동시 작성 |
| `SLINT_ENABLED=0` 비활성화 | 환경 변수 하나로 전체 계측 비활성화 |
| schemaVersion 계약 검증 | slint-gate가 입력 summary의 `schemaVersion`을 평가 전에 검증. 불일치 시 신뢰 가능한 PASS를 만들지 않음 |
| SLIResult.Status 반영 | 엔진이 계산한 SLI별 상태(fail/block→FAIL, warn→WARN, skip→NO_GRADE)가 gate 결과에 반영됨 |
| CounterResetPolicy | `ComputeSpec.OnCounterReset`으로 delta<0 처리 정책을 SLI별로 설정 (`warn` / `no_grade` / `fail` / `skip`) |
| summary 공개 계약 API | `summary.LoadFile()` / `WriteFile()` / `Validate()` — 외부 도구가 별도 struct 없이 summary 계약 사용 가능 |
| K8s 오브젝트 churn 측정 | `K8sObjectFetcher` — kubectl list 기반 오브젝트 수, orphan, ownerRef missing, stuck terminating 게이지 계산. 단, ownerRef missing은 현재 same-kind-only 제한이 있음 |
| evidence redaction | `evidence.RedactString()` / `RedactMap()` — Bearer 토큰, token=/password=/secret= 값 마스킹 |
| curlpod 식별 레이블 | 모든 curlpod에 `app.kubernetes.io/managed-by=kube-slint`, `slint-run-id=<RunID>` 레이블 부착; cleanup 범위를 소유 리소스로 제한 |
| 온보딩 CLI 루프 | `slint-gate init → inspect → recommend-policy → baseline approve → ci github-actions`; `quickstart`는 비대화형 상태 안내, `wizard`는 실제 TTY에서만 interactive하게 동작 |
| 커스텀 Semgrep 보안 가드레일 | `.semgrep/rules/`의 **7개** 프로젝트 전용 규칙이 토큰 노출, insecure TLS, ClusterRoleBinding, TOCTOU, unsafe cleanup, PodSpec/JSON injection 등의 회귀를 blocking CI로 차단 |

---

## 4. 기술 스택

| 항목 | 선택 | 이유 |
|---|---|---|
| 언어 | Go (`go.mod` 최소 1.22) | 쿠버네티스 생태계 표준 언어, 정적 바이너리 배포 용이 |
| controller-runtime 의존성 | 없음 | `go.mod`에 `sigs.k8s.io/controller-runtime` 미포함. 어떤 오퍼레이터와도 버전 충돌 최소화 |
| 계측 소스 모델 | point / snapshot / range-window | 데이터 모양에 따라 소스를 교체하면서 동일한 SLI 엔진과 gate 재사용 |
| 기본 클러스터 수집 | kubectl curl pod | 테스트 프로세스에서 클러스터 내부 네트워크 직접 접근 없이 `/metrics` 스크랩 가능 |
| 대체 수집 | port-forward / HTTP JSON·expvar / Prometheus query_range | 로컬 개발, non-Prometheus endpoint, window SLI 등 실사용 환경 확장 |
| 파싱 | Prometheus text-format + JSON | 기본 controller-runtime metrics와 source-neutral JSON 입력 모두 지원 |
| 정책 파일 형식 | YAML (`gopkg.in/yaml.v3`) | 사람이 읽고 수정하기 쉬운 선언적 설정 |
| 출력 형식 | JSON | CI 파이프라인 툴체인(jq, GitHub Actions)과 직접 연동 |
| CI 통합 | GitHub Composite Action | 별도 서버 없이 워크플로우에 추가 가능 |
| 이미지 | `distroless/static:nonroot` | 최소 공격 표면의 프로덕션 컨테이너 이미지 |
| 테스트 | Go 표준 testing + Ginkgo v2 | 단위 테스트와 E2E 테스트 프레임워크 분리 |

---

## 5. 아키텍처

전체 아키텍처는 [docs/architecture.md](architecture.md)에 Mermaid 다이어그램과 함께 상세히 기술되어 있습니다. 여기서는 핵심 흐름을 요약합니다.

```
E2E 테스트 세션
    |
    |-- sess.Start()
    |      └─ SnapshotFetcher.PreFetch()가 있으면 start 스냅샷 캐시
    |         (curlpod / port-forward / jsonendpoint 등)
    |
    |  (E2E 시나리오 실행)
    |
    |-- sess.End(ctx)
           ├─ point/snapshot source → end sample 수집
           ├─ WindowFetcher가 있으면 test window 범위 sample 수집
           ├─ engine → delta/start/end/window SLI 계산 + judge
           └─ sli-summary.json 작성 (unique + alias 2개)
                              |
                              v
                     slint-gate CLI
                     ① schemaVersion/입력 계약 검증
                     ② SLIResult.Status 반영
                     ③ policy threshold / regression / reliability / coverage 평가
                     slint-gate-summary.json 작성
                              |
                              v
                     GitHub Action
                     gate_result + exit-on 기준으로 CI pass/fail
```

주요 패키지 구조:

| 패키지 | 역할 |
|---|---|
| `pkg/slint` | 공개 API 진입점 및 Session 구현체 (Session, SessionConfig, DefaultSpecs, curlpod-backed 기본 fetcher bridge, 설정 자동 탐색) |
| `test/e2e/harness` | 과거 test/e2e import 경로 호환용 wrapper |
| `pkg/slo/spec` | SLISpec 선언 타입 (입력, ComputeSpec, CounterResetPolicy, JudgeSpec) |
| `pkg/slo/engine` | SLI 계산 코어 (delta/start/end + scalar window aggregation, judge, reliability 보조 계산) |
| `pkg/slo/fetch` | source-neutral `MetricsFetcher` / `SnapshotFetcher` / `WindowFetcher` 인터페이스 |
| `pkg/slo/fetch/curlpod` | kubectl curl pod 기반 in-cluster `/metrics` 스크랩; run-id 레이블 자동 부착 |
| `pkg/slo/fetch/portforward` | kubectl port-forward 기반 point/snapshot source |
| `pkg/slo/fetch/jsonendpoint` | HTTP JSON/expvar 숫자 leaf를 key/value sample로 변환하는 SnapshotFetcher |
| `pkg/slo/fetch/promrange` | Prometheus `query_range` 결과를 window sample로 변환하는 WindowFetcher |
| `pkg/slo/fetch/k8sobject` | kubectl list 기반 오브젝트 수 캡처; ExcludeSelector 지원. ownerRef missing은 same-kind-only 제한 |
| `pkg/slo/summary` | 출력 스키마 타입 + SchemaVersion 상수 + LoadFile/WriteFile/Validate 공개 API |
| `pkg/slo/evidence` | RedactString / RedactMap — 토큰·패스워드 마스킹 유틸리티 |
| `pkg/gate` | policy 평가 (schemaVersion 검증, result_status, threshold, regression, reliability, coverage) |
| `cmd/slint-gate` | CLI 진입점: gate 평가 + `init`, `inspect`, `recommend-policy`, `baseline approve/diff/merge`, `ci github-actions`, `quickstart`, `wizard`, `analyze-dataplane` |
| `.github/actions/slint-gate` | GitHub Composite Action; gate artifact/output 보존 후 `exit-on` 기준으로 최종 CI 상태 강제 |
| `pkg/kubeutil` | 클러스터 유틸리티 (토큰, namespace-scoped RBAC, WaitForReady, PollUntil 등) |
| `.semgrep/rules` | 프로젝트 전용 Semgrep 보안/안정성 가드레일 **7종** (positive/negative fixture 포함, CI blocking) |

---

## 6. 차별점 및 독창성

### 기존 도구와의 비교

| 비교 항목 | Prometheus + Grafana | kube-slint |
|---|---|---|
| 동작 시점 | 프로덕션 런타임 (사후 관측) | CI E2E 테스트 (사전 가드레일) |
| 오퍼레이터 코드 수정 | 메트릭 등록/노출 설정 필요 | kube-slint 자체 계측 코드는 오퍼레이터에 삽입하지 않음 |
| 인프라 요구사항 | 상시 Prometheus/Grafana 운영 | 기본 curlpod 경로는 별도 모니터링 서버 없이 동작 |
| CI 통합 | 별도 스크립트/규칙 구성 필요 | Composite Action + policy.yaml로 직접 연결 |
| 회귀 감지 | 수동 대시보드 비교 또는 별도 규칙 | baseline JSON 자동 비교 |
| 정책 선언 | PromQL alerting rule | policy.yaml (threshold/regression/reliability/coverage) |
| 측정 실패 처리 | 수집 누락을 별도 운영적으로 해석 | measurement failure와 correctness test failure를 분리하고 NO_GRADE로 표현 가능 |
| schema drift 방지 | 도구마다 별도 계약 필요 | schemaVersion 검증, invalid input이 신뢰 가능한 PASS를 만들지 않도록 차단 |

### 독창성

**shift-left 위치**: kube-slint의 핵심은 운영 SLI를 프로덕션 모니터링에서만 보는 것이 아니라 E2E 테스트 실행 구간의 증거로 만들어 코드 리뷰·CI 단계에서 판정하는 것입니다. correctness 테스트와 운영 품질 gate를 합치지 않고 별도 신호로 유지하는 것이 제품 정체성입니다.

**오퍼레이터 무수정 원칙**: 기본 curl pod 방식은 단순한 편의 기능이 아니라 책임 경계입니다. kube-slint는 operator binary 내부에 자신을 주입하지 않고, 테스트 환경에서 외부 관측자로 동작합니다. `controller-runtime`을 직접 의존하지 않기 때문에 대상 오퍼레이터의 dependency version과 분리됩니다.

**SnapshotFetcher 패턴**: curl pod 같은 source는 항상 현재 상태만 읽기 때문에 `Start()` 시점의 pre-workload sample을 캐시하지 않으면 `End()`에서 올바른 delta를 계산할 수 없습니다. `SnapshotFetcher.PreFetch()`는 이 타이밍 요구를 fetcher 내부 계약으로 캡슐화합니다.

**source-neutral + window 확장**: 엔진 입력을 Prometheus 구현에 고정하지 않고 keyed numeric sample 인터페이스로 분리했습니다. JSON/expvar endpoint도 같은 SLI 계산에 넣을 수 있고, N개의 시간 샘플이 필요한 경우 `WindowFetcher`를 통해 p95/p99/ratio 같은 window SLI까지 확장됩니다.

**신뢰도 점수**: 측정 결과의 수치만 보고하는 것이 아니라, 스크랩 지연, start/end skew, 누락 입력 등 측정 품질을 `Reliability`와 `confidenceScore`로 함께 기록합니다. 즉 "값"과 "그 값을 얼마나 믿을 수 있는가"를 분리합니다.

**schemaVersion 계약 강제**: 외부 도구가 다른 버전의 summary를 넣어도 조용히 통과하지 않습니다. `slint-gate`가 평가 전에 `schemaVersion`과 입력 계약을 검증하며, `summary.LoadFile()` / `Validate()`를 통해 외부 도구도 같은 계약을 재사용할 수 있습니다.

**CounterResetPolicy**: 프로세스 재시작으로 인한 카운터 리셋(delta < 0)을 SLI별로 다르게 처리할 수 있습니다. promotion gate처럼 측정 신뢰성이 중요한 경우 `no_grade`를 지정해 잘못된 PASS를 방지할 수 있습니다.

**측정 → 설명 → 추천 → 승인 → CI 온보딩 루프**: 처음 도입하는 사용자가 policy 스키마 전체를 먼저 학습하지 않아도 `init → inspect → recommend-policy → baseline approve → ci github-actions` 순서로 실제 CI gate까지 도달할 수 있습니다. `recommend-policy`는 측정되지 않은 SLI나 원칙적인 기준을 만들 수 없는 raw activity counter에 임의 threshold를 지어내지 않습니다.

**프로젝트 전용 정적 분석 가드레일**: 범용 룰셋에만 의존하지 않고, kube-slint가 보장해야 하는 보안 불변조건을 7개의 custom Semgrep rule로 코드화했습니다. 외부 metrics URL 우회, bearer token 노출, insecure TLS, ClusterRoleBinding, TOCTOU, unsafe cleanup, PodSpec raw JSON injection 같은 프로젝트 고유 회귀를 CI에서 blocking으로 차단합니다.

---

## 7. 사용 방법

### 라이브러리 임베드 (E2E 테스트)

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    Namespace:          "my-operator-system",
    MetricsServiceName: "my-operator-controller-manager-metrics-service",
    ServiceAccountName: "kube-slint-scraper", // curl pod가 자신의 마운트된 토큰을 직접 읽음
    ArtifactsDir:       "artifacts",
    Specs:              slint.DefaultSpecs(),
})

sess.Start()
// ... E2E 시나리오 실행 ...
sum, err := sess.End(ctx)
```

### 커스텀 SLI 스펙 정의 (CounterResetPolicy 포함)

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/spec"

mySpecs := []spec.SLISpec{
    {
        ID:    "reconcile_error_delta",
        Unit:  "count",
        Kind:  "delta_counter",
        Inputs: []spec.MetricRef{
            spec.PromMetric("controller_runtime_reconcile_total",
                spec.Labels{"result": "error"}),
        },
        // counter reset 시 NO_GRADE — 잘못된 PASS 방지
        Compute: spec.ComputeSpec{
            Mode:           spec.ComputeDelta,
            OnCounterReset: spec.CounterResetNoGrade,
        },
        Judge: &spec.JudgeSpec{Rules: []spec.Rule{
            {Op: spec.OpGT, Target: 0, Level: spec.LevelFail},
        }},
    },
}
```

### JSON/expvar source 사용

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/jsonendpoint"

fetcher := jsonendpoint.New("http://127.0.0.1:8080/debug/vars")
sess := slint.NewSession(slint.SessionConfig{
    Specs:   myJSONSpecs,
    Fetcher: fetcher,
})
```

### Prometheus range/window source 사용

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promrange"

windowFetcher := promrange.New(
    "http://prometheus:9090",
    `rate(http_requests_total[5m])`,
    30*time.Second,
)

sess := slint.NewSession(slint.SessionConfig{
    Specs:         windowSpecs, // 예: window_p95, window_ratio
    WindowFetcher: windowFetcher,
})
```

### K8s 오브젝트 churn 측정

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/k8sobject"

fetcher := k8sobject.New(k8sobject.Config{
    Namespace:                 "my-operator-system",
    Resource:                  "pods",
    Selector:                  "app=my-worker",
    ExcludeSelector:           "app.kubernetes.io/managed-by=kube-slint",
    MetricPrefix:              "k8s_pods",
    StuckTerminatingThreshold: 5 * time.Minute,
})
```

> `*_ownerref_missing_end`는 현재 같은 Resource kind 안에서만 owner를 확인합니다. Pod의 ReplicaSet/Job 같은 cross-kind owner를 해석하지 않으므로 이 값 하나만으로 hard gate를 구성하면 안 됩니다.

### summary 파일 로드 (외부 도구)

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/summary"

s, err := summary.LoadFile("artifacts/sli-summary.json")
err = summary.Validate(s)
```

### 정책 파일 (`.slint/policy.yaml`)

```yaml
schema_version: "slint.policy.v1"
thresholds:
  - name: "workqueue-not-backed-up"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 5
regression:
  enabled: true
  tolerance_percent: 10
coverage:
  required: true
  informational:
    - reconcile_success_delta
promote_to_fail:
  - threshold_miss
  - regression_detected
  - coverage_gap
```

(구 필드명 `fail_on`도 deprecated alias로 계속 동작하며 `promote_to_fail`과 union으로 반영됩니다.)

### CI 게이팅 (GitHub Actions)

```yaml
- name: slint-gate
  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@main
  with:
    summary: artifacts/sli-summary.json
    policy: .slint/policy.yaml
    exit-on: FAIL_OR_NOGRADE
```

장기 운용 CI에서는 `main` 대신 tag 또는 commit SHA pinning을 권장합니다.

### CLI 직접 실행

```sh
# 측정 결과 설명 (읽기 전용)
go run ./cmd/slint-gate inspect --summary artifacts/sli-summary.json

# 측정 기반 정책 초안 생성
go run ./cmd/slint-gate recommend-policy \
  --summary artifacts/sli-summary.json \
  --output .slint/policy.yaml

# 게이트 평가
go run ./cmd/slint-gate \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --baseline docs/baselines/current.json \
  --exit-on FAIL_OR_NOGRADE \
  --github-step-summary

# 현재 온보딩 상태와 다음 단계 확인
go run ./cmd/slint-gate quickstart
```

---

## 8. 활용 방안

### 대상 팀 및 프로젝트

**kubebuilder / controller-gen 기반 오퍼레이터 팀**: controller-runtime이 기본으로 노출하는 `controller_runtime_reconcile_total`, `workqueue_depth`, `rest_client_requests_total` 등의 메트릭이 kube-slint 기본 스펙과 잘 맞아 빠르게 시작할 수 있습니다.

**Operator SDK 기반 프로젝트**: Operator SDK도 controller-runtime 기반인 경우 같은 기본 SLI 경로를 적용할 수 있습니다.

**커스텀 runtime/서비스 상태를 함께 보고 싶은 팀**: HTTP JSON/expvar endpoint의 숫자 값도 keyed sample로 변환해 같은 SLI 엔진에 넣을 수 있으므로 Prometheus text-format만 강제하지 않습니다.

**window latency/ratio를 CI에서 보고 싶은 팀**: Prometheus `query_range`를 이용해 테스트 구간의 p95/p99 또는 ratio 형태의 scalar window SLI를 구성할 수 있습니다.

**복잡한 Job/Pod 라이프사이클을 가진 오퍼레이터**: `K8sObjectFetcher`로 오브젝트 수와 stuck terminating 같은 보조 신호를 수집할 수 있습니다. ownerRef missing은 same-kind-only 제한을 감안해 informational 또는 보조 신호로 사용하는 것이 안전합니다.

**사내 오퍼레이터 플랫폼 팀**: 여러 오퍼레이터에 공통 SLI 정책을 적용할 때, 각 repository에 policy와 CI gate를 넣어 공통 운영 기준선을 코드 리뷰 단계에서 강제할 수 있습니다.

**CNCF 프로젝트 기여자**: 오픈소스 오퍼레이터에 PR을 제출할 때, 기여자가 CI에서 운영 SLI 영향을 확인하고 리뷰어에게 구조화된 증거를 제시할 수 있습니다.

**클라우드 네이티브 교육 과정**: kind 클러스터와 `examples/kind-hello-operator` 예제를 이용해 `make demo` 한 번으로 계측 → summary → policy gate 흐름을 실습할 수 있습니다.

---

## 9. 향후 계획

| 항목 | 설명 |
|---|---|
| K8sObjectFetcher Session/E2E 연결 강화 | 현재 독립 fetcher의 실사용 연결 경로와 cross-kind owner semantics를 설계 후 확장 |
| histogram bucket quantile | Prometheus histogram bucket 기반 p95/p99 계산을 현재 scalar window 모델 위에 추가 |
| specialized burn-rate semantics | generic `window_ratio`보다 높은 수준의 error-budget burn-rate 계산과 정책 UX 설계 |
| Helm chart | Kustomize 없이 Helm으로 RBAC 및 설정 패키지 배포 |
| 다중 클러스터 지원 | 여러 kubeconfig 컨텍스트에 대한 병렬 측정 |
| 웹/로컬 히스토리 대시보드 | `sli-summary.json` 히스토리를 시각화하는 경량 도구 |
