# kube-slint — 한국 공개SW 개발자대회 제출 문서

---

## 1. 프로젝트 개요

kube-slint는 쿠버네티스 오퍼레이터 E2E 테스트 세션에 내장하는 **shift-left 운영 SLI 가드레일 라이브러리**입니다. 오퍼레이터를 수정하지 않고 외부 curl pod으로 `/metrics` 엔드포인트를 스크랩하여 reconcile rate, workqueue depth, REST client error 등의 SLI를 측정하고, 선언적 `policy.yaml`로 CI 게이트 판정을 내립니다. 기능 정확성을 검증하는 E2E 테스트와 독립적으로 동작하며, 계측 실패가 테스트를 중단시키지 않는 safety-first 원칙을 준수합니다.

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
| 오퍼레이터 코드 무수정 | 외부 curl pod으로 `/metrics`를 스크랩하므로 오퍼레이터에 계측 코드를 삽입하지 않음 |
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
| `slint-gate init` | `.slint/policy.yaml` 스캐폴딩 서브커맨드 |
| `slint-gate diagnose` | 정책 파일 unknown field 경고 출력 |

---

## 4. 기술 스택

| 항목 | 선택 | 이유 |
|---|---|---|
| 언어 | Go 1.25 | 쿠버네티스 생태계 표준 언어, 정적 바이너리 배포 용이 |
| controller-runtime 의존성 | 없음 | go.mod에 `sigs.k8s.io/controller-runtime` 미포함. 어떤 오퍼레이터와도 버전 충돌 없음 |
| 메트릭 수집 방식 | kubectl curl pod | 클러스터 외부에서 테스트 프로세스가 클러스터 내부 네트워크에 직접 접근하지 않아도 됨 |
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
    |-- sess.Start()   → curl pod 실행 → pre-workload /metrics 스냅샷 캐시
    |
    |  (E2E 시나리오 실행)
    |
    |-- sess.End(ctx)  → curl pod 실행 → post-workload 스냅샷 수집
                         engine.Execute() → SLI 델타/스냅샷 계산
                         sli-summary.json 작성 (unique + alias 2개)
                              |
                              v
                     slint-gate CLI
                     policy.yaml + baseline(선택) 로드
                     임계치 / 회귀 / 신뢰도 평가
                     slint-gate-summary.json 작성
                              |
                              v
                     GitHub Action
                     gate_result 기반 CI pass/fail
```

주요 패키지 구조:

| 패키지 | 역할 |
|---|---|
| `pkg/slint` | 공개 API 진입점 (Session, SessionConfig, DefaultSpecs) |
| `test/e2e/harness` | Session 구현체, curlPodFetcher, 설정 자동 탐색 |
| `pkg/slo/spec` | SLISpec 선언 타입 (측정 대상, 계산 방식, 판정 규칙) |
| `pkg/slo/engine` | SLI 계산 코어 (delta/end-snapshot, judge 규칙 평가) |
| `pkg/slo/fetch` | MetricsFetcher 인터페이스, SnapshotFetcher 확장 |
| `pkg/slo/summary` | 출력 스키마 타입 및 Writer 인터페이스 |
| `internal/gate` | policy 평가 (threshold/regression/reliability) |
| `cmd/slint-gate` | CLI 진입점 |
| `.github/actions/slint-gate` | GitHub Composite Action |

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

### 독창성

**shift-left 위치**: 쿠버네티스 생태계에서 운영 SLI를 프로덕션 모니터링이 아닌 CI 게이트로 가져온 라이브러리는 현재 공개 소스에서 찾기 어렵습니다. kube-slint는 "코드 리뷰 단계에서 운영 건전성을 확인한다"는 개념을 구체적인 구현물로 제시합니다.

**오퍼레이터 무수정 원칙**: curl pod 방식은 단순히 편의 기능이 아니라 설계 원칙입니다. 오퍼레이터 코드에 어떤 의존성도 주입하지 않으므로, 라이브러리 버전 업그레이드가 오퍼레이터의 빌드에 영향을 줄 수 없습니다. controller-runtime을 go.mod에 포함하지 않는 것도 같은 이유입니다.

**SnapshotFetcher 패턴**: curl pod은 항상 현재 상태의 `/metrics`를 반환합니다. `Start()`와 `End()` 사이의 시간 간격을 올바르게 측정하려면 `Start()` 시점의 스냅샷을 캐시해야 합니다. `SnapshotFetcher.PreFetch()` 인터페이스는 이 타이밍 문제를 fetcher 구현 내부로 캡슐화하여, engine이나 harness의 상위 코드에는 영향을 주지 않습니다.

**신뢰도 점수**: 측정 결과의 수치만 보고하는 것이 아니라, 그 수치가 얼마나 신뢰할 수 있는지를 0.0–1.0 `confidenceScore`로 함께 보고합니다. 스크랩 지연, start/end skew, 누락 입력, 생략된 SLI 항목이 있을 때 점수가 자동으로 감점됩니다.

---

## 7. 사용 방법

### 라이브러리 임베드 (E2E 테스트)

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    Namespace:          "my-operator-system",
    MetricsServiceName: "my-operator-controller-manager-metrics-service",
    Token:              token, // ServiceAccount 토큰
    ArtifactsDir:       "artifacts",
    Specs:              slint.DefaultSpecs(),
})

sess.Start()
// ... E2E 시나리오 실행 ...
sum, err := sess.End(ctx)
```

### 커스텀 SLI 스펙 정의

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
        Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
        Judge: &spec.JudgeSpec{Rules: []spec.Rule{
            {Op: spec.OpGT, Target: 0, Level: spec.LevelFail},
        }},
    },
}
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
fail_on:
  - threshold_miss
  - regression_detected
```

### CI 게이팅 (GitHub Actions)

```yaml
- name: slint-gate
  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@main
  with:
    measurement-summary: artifacts/sli-summary.json
    policy:              .slint/policy.yaml
    fail-on:             FAIL
```

### CLI 직접 실행

```sh
# 정책 파일 스캐폴딩
go run ./cmd/slint-gate init

# 게이트 평가
go run ./cmd/slint-gate \
  --measurement-summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --baseline docs/baselines/current.json \
  --fail-on FAIL \
  --github-step-summary
```

---

## 8. 활용 방안

### 대상 팀 및 프로젝트

**kubebuilder / controller-gen 기반 오퍼레이터 팀**: controller-runtime이 기본으로 노출하는 `controller_runtime_reconcile_total`, `workqueue_depth`, `rest_client_requests_total` 등의 메트릭이 kube-slint의 기본 스펙 세트(`DefaultSpecs()`)와 정확히 일치합니다. 설정 없이 즉시 사용할 수 있습니다.

**Operator SDK 기반 프로젝트**: Operator SDK도 controller-runtime 위에서 동작하므로 동일하게 적용됩니다.

**사내 오퍼레이터 플랫폼 팀**: 여러 오퍼레이터에 공통 SLI 정책을 적용할 때, 각 오퍼레이터 레포지토리에 policy.yaml 파일과 GitHub Action만 추가하면 됩니다. 중앙 모니터링 서버 없이 CI 레벨에서 운영 기준선을 강제할 수 있습니다.

**CNCF 프로젝트 기여자**: 오픈소스 오퍼레이터에 PR을 제출할 때, 기여자 자신이 CI에서 운영 SLI 영향을 확인하고 리뷰어에게 근거를 제시할 수 있습니다.

**클라우드 네이티브 교육 과정**: 오퍼레이터 개발 교육에서 "운영 관점의 테스트"를 실습할 때, 실제 클러스터 없이 kind 클러스터와 hello-operator 예제(`examples/kind-hello-operator`)로 전체 흐름을 체험할 수 있습니다.

---

## 9. 향후 계획

| 항목 | 설명 |
|---|---|
| kind + hello-operator 예제 완성 | kind 클러스터 기반 end-to-end 데모 — 클론 후 `./setup.sh` 하나로 전체 흐름 실행 |
| `make coverage` 커버리지 리포트 | CI에서 커버리지 임계치 자동 검증 |
| `v0.1.0` 릴리스 태그 | CHANGELOG, GitHub Release, 공모전 제출 패키지 |
| ServiceURLFormat SessionConfig 노출 | 현재 하드코딩된 `https://%s.%s.svc:8443/metrics` 패턴을 SessionConfig에서 완전히 제어 |
| PortForward fetcher | curl pod 대신 `kubectl port-forward` 기반 fetcher 옵션 (로컬 개발 환경용) |
| Helm chart | 복잡한 Kustomize 오버레이 없이 Helm으로 RBAC 및 설정 패키지 배포 |
| 다중 클러스터 지원 | 여러 kubeconfig 컨텍스트에 대한 병렬 측정 |
| 웹 대시보드 | sli-summary.json 히스토리를 시각화하는 경량 로컬 대시보드 |
