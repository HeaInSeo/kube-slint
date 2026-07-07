# kube-slint — 한국 공개SW 개발자대회 제출 문서

---

## 1. 프로젝트 개요

kube-slint는 쿠버네티스 오퍼레이터 E2E 테스트 세션에 내장하는 **shift-left 운영 SLI 가드레일 라이브러리**입니다. 오퍼레이터를 수정하지 않고 외부 curl pod 또는 port-forward로 `/metrics` 엔드포인트를 스크랩하여 reconcile rate, workqueue depth, REST client error 등의 SLI를 측정하고, 선언적 `policy.yaml`로 CI 게이트 판정을 내립니다. 기능 정확성을 검증하는 E2E 테스트와 독립적으로 동작하며, 계측 실패가 테스트를 중단시키지 않는 safety-first 원칙을 준수합니다.

**최신 릴리즈**: v1.5.2 (v1.5.0의 SLI Gate Onboarding UX 로드맵 Sprint 1-6, dataplane-service 정적 분석기, 커스텀 Semgrep 가드레일 / v1.5.1의 GitHub Action 아티팩트 보존 수정, `init` overwrite 방지, 공개 API 표면 주석/진단 메시지 영어 통일에 더해, ownerRef 메트릭 한계 문서화 및 이미지 pinning 정책 결정 포함)

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
| 오퍼레이터 코드 무수정 | 외부 curl pod 또는 port-forward로 `/metrics`를 스크랩하므로 오퍼레이터에 계측 코드를 삽입하지 않음 |
| 선언적 SLI 스펙 | `SLISpec` 구조체로 측정 대상, 계산 방식(delta / end-snapshot), 판정 규칙을 코드로 선언 |
| 기본 제공 스펙 세트 | reconcile total/success/error delta, workqueue depth/adds/retries, REST client requests/429/5xx — kubebuilder 오퍼레이터에 즉시 적용 가능 |
| 임계치 기반 게이팅 | `policy.yaml`의 threshold 규칙으로 절대값 기준 PASS/FAIL 판정 |
| 회귀 감지 | 이전 실행의 baseline JSON과 비교하여 허용 오차(`tolerance_percent`) 초과 시 FAIL |
| 신뢰도 점수 | 스크랩 지연, skew, 누락 입력을 반영한 0.0–1.0 `confidenceScore` 자동 산출 |
| 구조화된 아티팩트 | `sli-summary.json` (측정 결과), `slint-gate-summary.json` (게이트 판정) — JSON 스키마 고정 |
| GitHub Actions 통합 | Composite Action 한 줄 추가로 CI 게이팅; `$GITHUB_STEP_SUMMARY` 마크다운 렌더링 지원 |
| non-fatal 원칙 | 계측 실패는 테스트를 중단시키지 않고 `Reliability` 필드에 기록 |
| dual-write 전략 | 감사 추적용 unique 파일 + slint-gate 기본 입력용 static alias 동시 작성 |
| `SLINT_ENABLED=0` 비활성화 | 환경 변수 하나로 전체 계측 비활성화 |
| schemaVersion 계약 검증 | slint-gate가 입력 summary의 `schemaVersion`을 평가 전에 검증. 불일치 시 `MeasurementStatus=unsupported_schema`, `GateResult=NO_GRADE` |
| SLIResult.Status 반영 | 엔진이 계산한 SLI별 상태(fail/block→FAIL, warn→WARN, skip→NO_GRADE)가 gate 결과에 직접 반영됨 |
| CounterResetPolicy | `ComputeSpec.OnCounterReset` 필드로 delta<0 처리 정책을 SLI별로 설정 (`warn` / `no_grade` / `fail` / `skip`) |
| summary 공개 계약 API | `summary.LoadFile()` / `WriteFile()` / `Validate()` — 외부 도구가 별도 struct 없이 summary 계약 사용 가능 |
| K8s 오브젝트 churn 측정 | `K8sObjectFetcher` — kubectl list 기반 Pod/Job 수 측정. orphan / ownerRef missing / stuck terminating 게이지 자동 계산. ExcludeSelector로 kube-slint 관리 리소스 제외 |
| evidence redaction | `evidence.RedactString()` / `RedactMap()` — Bearer 토큰, token=/password=/secret= 값 마스킹 |
| curlpod 식별 레이블 | 모든 curlpod에 `app.kubernetes.io/managed-by=kube-slint`, `slint-run-id=<RunID>` 레이블 자동 부착. 삭제 실패 시 경고 로그 출력 |
| PortForward fetcher | `kubectl port-forward` 기반 fetcher — curl pod 생성 없이 로컬 개발 환경에서 사용 가능 |
| 온보딩 CLI 루프 | `slint-gate init → inspect → recommend-policy → baseline approve → ci github-actions`: 정책 스키마를 몰라도 측정 결과를 기반으로 정책 초안을 생성하고 CI에 연결. `slint-gate quickstart`가 현재 온보딩 단계와 다음 실행 명령을 알려줌 |
| 커스텀 Semgrep 보안 가드레일 | `.semgrep/rules/`의 6개 규칙(토큰 노출, insecure TLS, ClusterRoleBinding, TOCTOU, 안전하지 않은 cleanup)이 CI에서 blocking으로 실행되어 프로젝트 고유 보안 불변조건의 회귀를 코드 리뷰 없이 자동 차단 |

---

## 4. 기술 스택

| 항목 | 선택 | 이유 |
|---|---|---|
| 언어 | Go 1.25 | 쿠버네티스 생태계 표준 언어, 정적 바이너리 배포 용이 |
| controller-runtime 의존성 | 없음 | go.mod에 `sigs.k8s.io/controller-runtime` 미포함. 어떤 오퍼레이터와도 버전 충돌 없음 |
| 메트릭 수집 방식 | kubectl curl pod / port-forward | 클러스터 외부 테스트 프로세스에서 내부 네트워크에 직접 접근하지 않아도 됨 |
| 파싱 | Prometheus text-format | 별도 클라이언트 라이브러리 없이 `/metrics` 텍스트를 직접 파싱 |
| 정책 파일 형식 | YAML (gopkg.in/yaml.v3) | 사람이 읽고 수정하기 쉬운 선언적 설정 |
| 출력 형식 | JSON | CI 파이프라인 툴체인(jq, GitHub Actions)과 직접 연동 |
| CI 통합 | GitHub Composite Action | 별도 서버 없이 워크플로우에 한 줄 추가로 동작 |
| 이미지 | distroless/static:nonroot | 최소 공격 표면의 프로덕션 컨테이너 이미지 |
| 테스트 | Go 표준 testing + Ginkgo v2 | 단위 테스트와 E2E 테스트 프레임워크 분리 |

---

## 5. 아키텍처

전체 아키텍처는 [docs/architecture.md](architecture.md)에 Mermaid 다이어그램과 함께 상세히 기술되어 있습니다. 여기서는 핵심 흐름을 요약합니다.

```
E2E 테스트 세션
    |
    |-- sess.Start()   → Fetcher.PreFetch() → start 스냅샷 캐시
    |                    (curlpod / port-forward / K8sObjectFetcher)
    |
    |  (E2E 시나리오 실행)
    |
    |-- sess.End(ctx)  → Fetcher.Fetch() → end 스냅샷 수집
                         engine.Execute() → SLI 델타/스냅샷/judge 계산
                         sli-summary.json 작성 (unique + alias 2개)
                              |
                              v
                     slint-gate CLI
                     ① schemaVersion 검증
                     ② SLIResult.Status 반영 (result_status checks)
                     ③ policy.yaml threshold / 회귀 / 신뢰도 평가
                     slint-gate-summary.json 작성
                              |
                              v
                     GitHub Action
                     gate_result 기반 CI pass/fail
```

주요 패키지 구조:

| 패키지 | 역할 |
|---|---|
| `pkg/slint` | 공개 API 진입점 및 Session 구현체 (Session, SessionConfig, DefaultSpecs, ReadServiceAccountToken, curlpod-backed fetcher bridge, 설정 자동 탐색) |
| `test/e2e/harness` | 과거 test/e2e import 경로 호환용 wrapper |
| `pkg/slo/spec` | SLISpec 선언 타입 (측정 대상, ComputeSpec, CounterResetPolicy, JudgeSpec) |
| `pkg/slo/engine` | SLI 계산 코어 (delta/end-snapshot, CounterResetPolicy 적용, judge 규칙 평가) |
| `pkg/slo/fetch` | MetricsFetcher / SnapshotFetcher 인터페이스; WindowFetcher 설계 초안 |
| `pkg/slo/fetch/curlpod` | kubectl curl pod 기반 메트릭 스크랩; run-id 레이블 자동 부착 |
| `pkg/slo/fetch/portforward` | kubectl port-forward 기반 MetricsFetcher + SnapshotFetcher |
| `pkg/slo/fetch/k8sobject` | K8sObjectFetcher — kubectl list 기반 오브젝트 수 캡처; ExcludeSelector 지원 |
| `pkg/slo/summary` | 출력 스키마 타입 + SchemaVersion 상수 + LoadFile/WriteFile/Validate 공개 API |
| `pkg/slo/evidence` | RedactString / RedactMap — 토큰·패스워드 마스킹 유틸리티 |
| `pkg/gate` | policy 평가 (schemaVersion 검증, result_status, threshold, regression, reliability) |
| `cmd/slint-gate` | CLI 진입점: 게이트 평가(`--exit-on`, `--github-step-summary`) + 온보딩 서브커맨드(`init`, `inspect`, `recommend-policy`, `baseline approve/diff/merge`, `ci github-actions`, `quickstart`, `analyze-dataplane`) |
| `.github/actions/slint-gate` | GitHub Composite Action; 4단계 exit-on 지원 |
| `pkg/kubeutil` | 클러스터 유틸리티 (토큰, RBAC, WaitForReady, PollUntil) |
| `.semgrep/rules` | 프로젝트 전용 Semgrep 보안/안정성 가드레일 6종 (positive/negative fixture 포함, CI blocking) |

---

## 6. 차별점 및 독창성

### 기존 도구와의 비교

| 비교 항목 | Prometheus + Grafana | kube-slint |
|---|---|---|
| 동작 시점 | 프로덕션 런타임 (사후 관측) | CI E2E 테스트 (사전 가드레일) |
| 오퍼레이터 코드 수정 | 메트릭 등록 코드 필요 | 불필요 (외부 스크랩) |
| 인프라 요구사항 | Prometheus 서버, Grafana 서버 | kubectl 하나로 충분 |
| CI 통합 | 별도 스크립트 작성 필요 | GitHub Action 한 줄 |
| 회귀 감지 | 수동 대시보드 비교 | baseline JSON 자동 비교 |
| 정책 선언 | PromQL 알림 규칙 | policy.yaml (사람이 읽기 쉬운 YAML) |
| 측정 실패 처리 | 알림 누락 | non-fatal, Reliability 필드에 기록 |
| schema drift 방지 | 없음 | schemaVersion 계약 검증, 위반 시 NO_GRADE |

### 독창성

**shift-left 위치**: 쿠버네티스 생태계에서 운영 SLI를 프로덕션 모니터링이 아닌 CI 게이트로 가져온 라이브러리는 현재 공개 소스에서 찾기 어렵습니다. kube-slint는 "코드 리뷰 단계에서 운영 건전성을 확인한다"는 개념을 구체적인 구현물로 제시합니다.

**오퍼레이터 무수정 원칙**: curl pod 방식은 단순히 편의 기능이 아니라 설계 원칙입니다. 오퍼레이터 코드에 어떤 의존성도 주입하지 않으므로, 라이브러리 버전 업그레이드가 오퍼레이터의 빌드에 영향을 줄 수 없습니다. controller-runtime을 go.mod에 포함하지 않는 것도 같은 이유입니다.

**SnapshotFetcher 패턴**: curl pod은 항상 현재 상태의 `/metrics`를 반환합니다. `Start()`와 `End()` 사이의 시간 간격을 올바르게 측정하려면 `Start()` 시점의 스냅샷을 캐시해야 합니다. `SnapshotFetcher.PreFetch()` 인터페이스는 이 타이밍 문제를 fetcher 구현 내부로 캡슐화하여, engine이나 session 상위 코드에는 영향을 주지 않습니다. `K8sObjectFetcher`도 동일한 패턴으로 엔진 변경 없이 오브젝트 churn 측정을 추가합니다.

**신뢰도 점수**: 측정 결과의 수치만 보고하는 것이 아니라, 그 수치가 얼마나 신뢰할 수 있는지를 0.0–1.0 `confidenceScore`로 함께 보고합니다. 스크랩 지연, start/end skew, 누락 입력, 생략된 SLI 항목이 있을 때 점수가 자동으로 감점됩니다.

**schemaVersion 계약 강제**: 외부 도구가 다른 버전의 summary를 넣어도 조용히 통과하지 않습니다. `slint-gate`가 평가 전에 `schemaVersion`을 검증하여 schema drift를 즉시 탐지합니다. `summary.LoadFile()` / `Validate()`를 통해 외부 도구도 동일한 계약을 검증할 수 있습니다.

**CounterResetPolicy**: 프로세스 재시작으로 인한 카운터 리셋(delta < 0)을 SLI별로 다르게 처리할 수 있습니다. promotion gate처럼 측정 신뢰성이 중요한 경우 `no_grade`를 지정하면 counter reset 발생 시 PASS가 아닌 NO_GRADE로 처리되어 잘못된 승인을 방지합니다.

**측정 → 설명 → 추천 → 승인 → CI 온보딩 루프**: 처음 도입하는 사용자가 policy 스키마 전체를 배우기 전에 `init → inspect → recommend-policy → baseline approve → ci github-actions` 순서로 실제 CI 게이트까지 도달할 수 있습니다. `recommend-policy`는 실제 측정된 SLI만 active threshold로 만들고, profile이 기대하지만 측정되지 않은 SLI는 주석으로만 남기며, 원칙적인 pass/fail 기준이 없는 raw activity counter(예: 총 reconcile 횟수)는 어떤 strictness 설정에서도 threshold를 지어내지 않습니다 — "측정하지 못한 것"과 "측정했지만 게이트로 쓸 근거가 없는 것"을 구분해서 보여주는 설계입니다.

**프로젝트 전용 정적 분석 가드레일**: 범용 OWASP 룰셋 대신, 이 프로젝트가 이미 문서화한 보안 불변조건(ServiceURLFormat 검증 우회, 토큰의 커맨드 인자 노출, insecure TLS, ClusterRoleBinding 기본 생성, TOCTOU, label 없는 cleanup)을 코드 레벨에서 강제하는 6개의 커스텀 Semgrep 규칙을 작성하고, 실제 코드베이스를 전수 스캔하여 컴플라이언스를 검증한 뒤 CI에서 blocking으로 연결했습니다. 코드 리뷰 없이도 회귀를 자동으로 막습니다.

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

### K8s 오브젝트 churn 측정

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/k8sobject"

fetcher := k8sobject.New(k8sobject.Config{
    Namespace:                 "jumi-system",
    Resource:                  "pods",
    Selector:                  "app=jumi-worker",
    // kube-slint 관리 리소스(curlpod 등) 제외
    ExcludeSelector:           "app.kubernetes.io/managed-by=kube-slint",
    MetricPrefix:              "k8s_pods",
    StuckTerminatingThreshold: 5 * time.Minute,
})
// 생성되는 메트릭: k8s_pods_count, k8s_pods_orphan_end,
//   k8s_pods_ownerref_missing_end, k8s_pods_stuck_terminating_end
```

### summary 파일 로드 (외부 도구)

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/summary"

// schemaVersion 검증 포함
s, err := summary.LoadFile("artifacts/sli-summary.json")

// 종합 검증
err = summary.Validate(s)
```

### 정책 파일 (`.slint/policy.yaml`)

```yaml
schema_version: "slint.policy.v1"
thresholds:
  - name: "reconcile-activity"
    metric: "reconcile_total_delta"
    operator: ">="
    value: 1
  - name: "workqueue-not-backed-up"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 5
regression:
  enabled: true
  tolerance_percent: 10
promote_to_fail:
  - threshold_miss
  - regression_detected
```

(구 필드명 `fail_on`도 deprecated alias로 계속 동작 — 둘 다 union으로 반영되며 `fail_on` 사용 시 `policy_warnings`에 마이그레이션 안내가 남음.)

### CI 게이팅 (GitHub Actions)

```yaml
- name: slint-gate
  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@main
  with:
    measurement-summary: artifacts/sli-summary.json
    policy:              .slint/policy.yaml
    exit-on:             FAIL_OR_NOGRADE
```

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
  --measurement-summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --baseline docs/baselines/current.json \
  --exit-on FAIL_OR_NOGRADE \
  --github-step-summary

# 지금 어느 단계인지 + 다음에 뭘 실행해야 하는지 확인
go run ./cmd/slint-gate quickstart
```

---

## 8. 활용 방안

### 대상 팀 및 프로젝트

**kubebuilder / controller-gen 기반 오퍼레이터 팀**: controller-runtime이 기본으로 노출하는 `controller_runtime_reconcile_total`, `workqueue_depth`, `rest_client_requests_total` 등의 메트릭이 kube-slint의 기본 스펙 세트(`DefaultSpecs()`)와 정확히 일치합니다. 설정 없이 즉시 사용할 수 있습니다.

**Operator SDK 기반 프로젝트**: Operator SDK도 controller-runtime 위에서 동작하므로 동일하게 적용됩니다.

**복잡한 Job/Pod 라이프사이클을 가진 오퍼레이터**: `K8sObjectFetcher`로 reconcile 중 생성되는 오브젝트의 수, orphan 상태, ownerRef 누락, stuck terminating 여부를 측정하여 리소스 누수를 CI 단계에서 조기 탐지할 수 있습니다.

**사내 오퍼레이터 플랫폼 팀**: 여러 오퍼레이터에 공통 SLI 정책을 적용할 때, 각 오퍼레이터 레포지토리에 policy.yaml 파일과 GitHub Action만 추가하면 됩니다. 중앙 모니터링 서버 없이 CI 레벨에서 운영 기준선을 강제할 수 있습니다.

**CNCF 프로젝트 기여자**: 오픈소스 오퍼레이터에 PR을 제출할 때, 기여자 자신이 CI에서 운영 SLI 영향을 확인하고 리뷰어에게 근거를 제시할 수 있습니다.

**클라우드 네이티브 교육 과정**: 오퍼레이터 개발 교육에서 "운영 관점의 테스트"를 실습할 때, 실제 클러스터 없이 kind 클러스터와 hello-operator 예제(`examples/kind-hello-operator`)로 전체 흐름을 체험할 수 있습니다. `make demo` 하나로 클러스터 생성부터 게이트 판정까지 전 과정이 실행됩니다.

---

## 9. 향후 계획

| 항목 | 설명 |
|---|---|
| K8sObjectFetcher E2E 연결 | `pkg/slint` session 경로에 `K8sObjectFetcher` 연결 코드 추가 — bori 통합 시나리오 확정 후 진행 |
| PromQL range query 지원 | Tier 2 소스 모델(WindowFetcher) 구현 — 엔진 확장 설계 후 진행 (`docs/verification-sources.md` 참조) |
| Helm chart | Kustomize 없이 Helm으로 RBAC 및 설정 패키지 배포 |
| 다중 클러스터 지원 | 여러 kubeconfig 컨텍스트에 대한 병렬 측정 |
| 웹 대시보드 | sli-summary.json 히스토리를 시각화하는 경량 로컬 대시보드 |
