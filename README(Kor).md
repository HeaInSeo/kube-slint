# kube-slint

English documentation is available in [README.md](README.md).

**kube-slint는 여러분의 테스트를 대체하지 않습니다. 테스트 중에 일어나는 일을 측정합니다.**

기존 Kubernetes 오퍼레이터 E2E 세션에 kube-slint를 붙이세요. 워크로드 전후의 `/metrics`를 읽어 운영 SLI 델타(reconcile 비율, 워크큐 깊이, REST 오류)를 계산하고, 선언적 정책과 대조해 평가합니다 — 오퍼레이터 코드를 수정할 필요 없습니다.

**지금 바로 체험** (kind ≥ v0.22, Docker, Go 1.25+ 필요):

```bash
cd examples/kind-hello-operator
make demo
```

---

## 핵심 계약

1. **계측 실패는 테스트 실패가 아닙니다.** kube-slint가 메트릭을 스크랩할 수 없는 경우, 해당 결과는 미계측으로 기록됩니다. E2E 테스트는 계속 진행됩니다.
2. **정책 위반은 CI를 실패시킬 수 있습니다.** 임계치 미달 또는 기준선 대비 회귀가 감지되면 게이트 스텝이 비정상 종료 코드로 종료되어 CI 작업이 실패합니다.
3. **가드레일 평가와 correctness 테스트는 분리됩니다.** 기존 E2E 단언과 kube-slint 게이트 결과는 독립적인 신호입니다. 둘 다 독립적으로 실패할 수 있습니다.

---

## 동작 방식

```
E2E 테스트 프로세스
     |
     |--- sess.Start() --------> 메트릭 관측 시작
     |
     | (E2E 시나리오 실행)
     |
     |--- sess.End(ctx) -------> 메트릭 수집, SLI 스펙 평가
                                  artifacts/sli-summary.json 작성
                                         |
                                         v
                              slint-gate CLI
                         (cmd/slint-gate 바이너리)
                                  |
                         sli-summary.json 읽기
                         .slint/policy.yaml 읽기
                         baseline 읽기 (선택)
                                  |
                                  v
                       slint-gate-summary.json
                                  |
                         gate_result: PASS
                                     WARN
                                     FAIL      ---> CI 실패
                                     NO_GRADE
```

---

## 빠른 시작

**1단계: 의존성 추가**

```sh
go get github.com/HeaInSeo/kube-slint
```

**2단계: E2E 테스트에 하네스 임베드**

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    Namespace:             "my-operator-system",
    MetricsServiceName:    "my-operator-controller-manager-metrics-service",
    ServiceAccountName:    "kube-slint-scraper",
    ArtifactsDir:          "artifacts",
    Specs:                 slint.DefaultSpecs(),
    CurlImage:             "my-registry/curlimages/curl:8.11.0",
})
sess.Start()
// ... E2E 시나리오 실행 ...
sess.End(ctx)
```

**3단계: 결과 게이팅**

```sh
make slint-gate   # bin/slint-gate 빌드
./bin/slint-gate --measurement-summary artifacts/sli-summary.json \
                 --policy .slint/policy.yaml \
                 --output slint-gate-summary.json
```

게이트 결과 확인:

```sh
jq -r '.gate_result' slint-gate-summary.json
```

---

## 상세 사용법

### 1. SLI 스펙 정의

**프리셋 스펙 사용**

`slint.DefaultSpecs()`(구 이름: `DefaultV3Specs`, `BaselineV3Specs`)는 kubebuilder로 생성된 오퍼레이터를 위해 설계된 프리셋 스펙 세트를 반환합니다. 다음 항목을 포함합니다:

| ID | 설명 |
|---|---|
| `reconcile_total_delta` | 세션 중 총 reconcile 호출 횟수 |
| `reconcile_success_delta` | 성공한 reconcile 호출 횟수 |
| `reconcile_error_delta` | 실패한 reconcile 호출 횟수 |
| `workqueue_adds_total_delta` | 워크큐에 추가된 항목 수 |
| `workqueue_retries_total_delta` | 워크큐 재시도 횟수 |
| `workqueue_depth_end` | 세션 종료 시점의 워크큐 깊이 |
| `rest_client_requests_total_delta` | 총 REST 클라이언트 요청 수 |
| `rest_client_429_delta` | 수신된 속도 제한(429) 응답 수 |
| `rest_client_5xx_delta` | 수신된 서버 오류(5xx) 응답 수 |

```go
specs := slint.DefaultSpecs()
```

**커스텀 SLI 스펙 정의**

```go
import (
    "github.com/HeaInSeo/kube-slint/pkg/slo/spec"
)

mySpecs := []spec.SLISpec{
    {
        ID:    "reconcile_error_delta",
        Title: "Reconcile Error Delta",
        Unit:  "count",
        Kind:  "delta_counter",
        Inputs: []spec.MetricRef{
            spec.PromMetric("controller_runtime_reconcile_total", spec.Labels{"result": "error"}),
        },
        Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
        Judge: &spec.JudgeSpec{Rules: []spec.Rule{
            {Op: spec.OpGT, Target: 0, Level: spec.LevelFail},
        }},
    },
}
```

`mySpecs`를 `SessionConfig`의 `Specs` 필드에 전달합니다.

---

### 2. E2E 테스트에 하네스 임베드

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    // 오퍼레이터가 실행 중인 대상 네임스페이스
    Namespace: "my-operator-system",

    // /metrics를 노출하는 쿠버네티스 서비스 이름
    MetricsServiceName: "my-operator-controller-manager-metrics-service",

    // 임시 curl pod가 사용할 ServiceAccount
    // bearer token은 kubectl args에 넣지 않고 pod 내부 mounted token을 읽습니다.
    ServiceAccountName: "kube-slint-scraper",

    // sli-summary.json이 작성될 디렉토리
    ArtifactsDir: "artifacts",

    // SLI 스펙 세트 — 프리셋 또는 커스텀 사용
    Specs: slint.DefaultSpecs(),

    // 선택적 실클러스터 설정
    CurlImage: "my-registry/curlimages/curl:8.11.0",
})

sess.Start()
// ... 여기에 E2E 시나리오 실행 ...
sess.End(ctx)
```

**RBAC 주의사항:** 하네스는 curl 기반 페처를 사용하여 메트릭 엔드포인트를 스크랩하기 위한 임시 파드를 생성합니다. `slint-gate init --emit-rbac rbac.yaml`은 대상 네임스페이스 안에 필요한 ServiceAccount, Role, RoleBinding을 생성하는 매니페스트를 출력합니다.

**출력:** `sess.End(ctx)` 호출 시 두 파일이 작성됩니다:
- `artifacts/sli-summary.<runID>.<testcase>.json` — 감사 추적용 unique 파일
- `artifacts/sli-summary.json` — slint-gate 기본 입력 경로 (latest alias)

---

### 2a. Token 처리

기본 curl pod 경로는 pod 내부의 mounted ServiceAccount token을 직접 읽습니다. 따라서 bearer token이 `kubectl` 인자나 Pod command에 직접 들어가지 않습니다. `pkg/slint`의 token 읽기 헬퍼는 호환용 API로 유지됩니다:

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

// 기본 경로에서 토큰 읽기 (/var/run/secrets/kubernetes.io/serviceaccount/token)
token, err := slint.ReadServiceAccountToken(slint.DefaultTokenPath)

// 환경 변수 우선, 없으면 파일 폴백
token, err := slint.ReadServiceAccountTokenFromEnv("SLINT_TOKEN", slint.DefaultTokenPath)
```

새 코드에서는 이 값을 curl pod command에 삽입하지 않습니다.

**HTTP 엔드포인트 사용 시 (기본 포트 8443 → 8080 변경):**

```go
sess := slint.NewSession(slint.SessionConfig{
    // ...
    ServiceURLFormat: slint.ServiceURLHTTP, // "http://%s.%s.svc:8080/metrics"
})
```

기본값은 `slint.ServiceURLHTTPS` (`"https://%s.%s.svc:8443/metrics"`)입니다.

기본적으로 `ServiceURLFormat`은 클러스터 내부 Service 주소
(`<service>.<namespace>.svc` 또는 `.svc.cluster.local`, `http`/`https`만
허용)로만 해석되어야 합니다 — 그 외에는 curl pod 생성 전에 거부되므로
잘못 설정되거나 악의적인 외부 URL이 스크랩용 Authorization 토큰을 받을 수
없습니다. 이 기본값을 명시적으로 해제하려면 `DangerouslyAllowExternalMetricsURL: true`를 설정하세요.

`TLSInsecureSkipVerify`는 `DangerouslySkipTLSVerify`로 대체된 deprecated
필드입니다(효과는 동일하고 이름만 명확함) — 자체 서명 인증서를 쓰는 개발
클러스터와의 호환을 위해 남아 있지만 TLS 검증을 약화합니다. 공유 CI나
운영과 유사한 환경에서는 위험을 명시적으로 수용한 경우가 아니라면 켜지
마십시오. 기본값은 `false`입니다.

---

### 3. 결과 게이팅 (slint-gate CLI)

게이트 CLI는 `cmd/slint-gate`에 위치한 Go 바이너리입니다. 이전 Python prototype(`hack/slint_gate.py`)은 완전히 제거되었으며, 운영 경로는 Go 바이너리만 사용합니다.

**빌드**

```sh
make slint-gate
# bin/slint-gate 생성

# 또는 빌드 없이 직접 실행
go run ./cmd/slint-gate [flags]
```

**플래그**

| 플래그 | 기본값 | 설명 |
|---|---|---|
| `--measurement-summary` | `artifacts/sli-summary.json` | 하네스가 생성한 SLI 요약 파일 경로 |
| `--policy` | `.slint/policy.yaml` | 정책 파일 경로 |
| `--baseline` | `""` (비활성) | 회귀 비교용 기준선 요약 경로; 생략하면 건너뜀 |
| `--output` | `slint-gate-summary.json` | 게이트 결과 JSON 작성 경로 |
| `--fail-on` | `NEVER` | 종료 코드 1을 반환할 gate_result 조건 (아래 참조) |
| `--github-step-summary` | false | `$GITHUB_STEP_SUMMARY`에 마크다운 작성 (GitHub Actions용) |

**`--fail-on` 레벨**

| 값 | 설명 |
|---|---|
| `NEVER` | 항상 0으로 종료 (기본값) |
| `FAIL` | `gate_result=FAIL`일 때 exit 1 |
| `FAIL_OR_WARN` | `FAIL` 또는 `WARN`일 때 exit 1 |
| `FAIL_OR_NOGRADE` | `FAIL` 또는 `NO_GRADE`일 때 exit 1 |
| `FAIL_WARN_OR_NOGRADE` | `FAIL`, `WARN`, `NO_GRADE` 모두 exit 1 |

**종료 동작:** 기본값(`NEVER`)에서는 항상 0으로 종료합니다. `--fail-on FAIL` 이상을 지정하면 해당 조건에서 exit 1이 반환됩니다. 알 수 없는 `--fail-on` 값은 즉시 거부됩니다.

**정책 파일 (`.slint/policy.yaml`)**

```yaml
schema_version: "slint.policy.v1"
thresholds:
  - name: "reconcile_total_delta_min"
    metric: "reconcile_total_delta"   # sli-summary.json의 results[].id와 일치해야 함
    operator: ">="
    value: 1
  - name: "workqueue_depth_end_max"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 5
regression:
  enabled: true
  tolerance_percent: 5
reliability:
  required: false
  min_level: "partial"
fail_on:
  - "threshold_miss"
  - "regression_detected"
```

**게이트 결과값**

| 결과 | 의미 |
|---|---|
| `PASS` | 모든 임계치 및 회귀 검사 통과 |
| `WARN` | 검사가 실패했지만 해당 항목이 `fail_on`에 없거나, 비차단 조건(기준선 없는 첫 실행, 신뢰도 최솟값 미달)인 경우 |
| `FAIL` | `fail_on`에 포함된 정책 위반 — 임계치 미달 또는 회귀 감지 |
| `NO_GRADE` | 평가 불가 — 입력 파일 누락 또는 손상 |

**`fail_on` 두 레이어 구조**

`policy.fail_on`과 CLI `--fail-on`은 별개의 제어 레이어입니다.

| 레이어 | 역할 |
|---|---|
| `policy.fail_on` | 어떤 위반 항목을 `gate_result=FAIL`로 승격할지 결정. 목록에 없는 위반은 `WARN`이 됩니다. 실패한 검사가 `PASS`가 되는 일은 없습니다. |
| CLI `--fail-on` | 어떤 `gate_result`에서 프로세스를 exit 1로 종료할지 결정 |

`fail_on`을 생략하거나 빈 배열로 두면 기본 hard-fail 항목인 `threshold_miss`, `regression_detected`가 적용됩니다.

**`checks[].observed` 타입 주의사항**

`observed`는 일반적으로 숫자입니다. 다만 baseline=0이고 current가 0이 아닌 regression처럼 수치 비율로 표현하기 어려운 경우에는 `"baseline_zero_current_nonzero"` 같은 문자열 marker가 들어갈 수 있습니다. jq 스크립트나 대시보드는 `observed`가 항상 숫자라고 가정하면 안 됩니다.

---

### 4. 관측성 스택 배포 (Kustomize)

kube-slint는 클러스터에 메트릭 수집 인프라를 설치하는 Kustomize 베이스를 제공합니다.

**원격 베이스 참조**

```yaml
# your overlay/kustomization.yaml
resources:
  - github.com/HeaInSeo/kube-slint//config/default?ref=<tag-or-SHA>
```

규칙:
- 항상 태그 또는 SHA로 고정합니다. `?ref=main`은 절대 사용하지 않습니다.
- 오버레이에 반드시 `namespace:`를 선언합니다. 베이스는 네임스페이스를 가정하지 않습니다.

**ServiceMonitor 레이블**

베이스의 ServiceMonitor에는 레이블이 하드코딩되어 있습니다. Prometheus 오퍼레이터의 셀렉터에 맞게 전략적 머지 패치로 이를 재정의해야 합니다.

```yaml
# overlay/patch-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kube-slint-controller-manager-metrics-monitor
  namespace: <your-namespace>
spec:
  selector:
    matchLabels:
      # Prometheus 오퍼레이터가 선택하는 레이블로 재정의
      app: my-operator
```

전체 온보딩 튜토리얼은 `test/consumer-onboarding/kustomize-remote-consumer/`에서 확인할 수 있습니다.

---

## 계측 모드

kube-slint는 세션 또는 스펙별로 설정할 수 있는 세 가지 1급 계측 모드를 지원합니다:

| 모드 | 설명 |
|---|---|
| `InsideSnapshot` (기본) | 세션 시작과 종료 시 스냅샷 기반 수집; 두 값의 차이로 델타를 계산 |
| `InsideAnnotation` | 정밀 시맨틱 경계 수집; 측정이 어노테이션된 테스트 경계에 정렬됨 |
| `OutsideSnapshot` | 외부 스크랩; 세션 내부가 아닌 외부 소스에서 메트릭 수집 |

---

## 게이트 모델

두 가지 게이트 모델 구성 요소가 모두 완성되었습니다.

**임계치 검사** (완료): `sli-summary.json`의 각 메트릭 결과가 `policy.yaml`의 임계치 규칙과 비교됩니다. `fail_on`에 `threshold_miss`가 포함된 경우 임계치 미달 시 `gate_result`가 `FAIL`로 설정됩니다.

**회귀 감지** (완료): `--baseline`이 제공된 경우, 각 메트릭 결과가 저장된 기준선 값과 비교됩니다. 변화량이 `tolerance_percent`를 초과하면 회귀로 플래그됩니다. `fail_on`에 `regression_detected`가 포함된 경우 회귀 감지 시 `gate_result`가 `FAIL`로 설정됩니다.

---

## 보안 기본값

kube-slint의 기본 계측 경로는 네임스페이스 스코프입니다:

- 생성 RBAC는 ServiceAccount, Role, RoleBinding을 사용합니다.
- 기본 경로에는 ClusterRoleBinding이 필요하지 않습니다.
- curl pod는 pod 내부에 mount된 ServiceAccount token을 읽습니다.
- command/error 출력은 일반적인 token/secret 형태를 redaction합니다.
- 계측이 불충분한 경우 `NO_GRADE`가 1급 gate 결과로 노출됩니다.
- `ServiceURLFormat`은 curl pod 생성 전에 검증됩니다 — 외부 host, 지원하지
  않는 scheme, 잘못된 형식의 service/namespace 값은 기본적으로 거부됩니다
  (해제하려면 `DangerouslyAllowExternalMetricsURL` 참고).
- `kube-system`/`kube-public`/`kube-node-lease`는 기본적으로 계측 대상
  namespace로 거부됩니다 (해제하려면 `DangerouslyAllowKubeSystemNamespace`
  참고).

전체 default-deny 정책과 dangerous option 목록은 `docs/security-model.md`를
참고하세요.

---

## CI 통합

### GitHub Composite Action (권장)

워크플로우에 추가할 수 있습니다. 현재 action은 Go CLI를 source에서 실행하므로
먼저 Go를 설정해야 합니다:

```yaml
jobs:
  e2e:
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      # ... artifacts/sli-summary.json을 생성하는 E2E 스텝 ...

      - name: slint-gate
        uses: HeaInSeo/kube-slint/.github/actions/slint-gate@main
        with:
          measurement-summary: artifacts/sli-summary.json   # 기본값
          policy:              .slint/policy.yaml            # 기본값
          fail-on:             FAIL_OR_NOGRADE               # NEVER | FAIL | FAIL_OR_WARN | FAIL_OR_NOGRADE | FAIL_WARN_OR_NOGRADE
```

**입력값**

| 입력 | 기본값 | 설명 |
|---|---|---|
| `measurement-summary` | `artifacts/sli-summary.json` | sli-summary.json 경로 |
| `policy` | `.slint/policy.yaml` | 정책 YAML 경로 |
| `baseline` | `` | 선택적 기준선 경로 |
| `output` | `slint-gate-summary.json` | 게이트 결과 출력 경로 |
| `fail-on` | `FAIL_OR_NOGRADE` | `NEVER`\|`FAIL`\|`FAIL_OR_WARN`\|`FAIL_OR_NOGRADE`\|`FAIL_WARN_OR_NOGRADE` |
| `github-step-summary` | `true` | Markdown 요약 테이블 출력 여부 |
| `upload-artifact` | `true` | 게이트 결과 아티팩트 업로드 여부 |

**출력값**: `gate-result`, `evaluation-status`, `summary-path`

장기 CI에서는 가능하면 `main` 대신 tag 또는 SHA에 pin하십시오.

### 직접 실행

```yaml
- name: slint-gate 평가
  run: |
    go run ./cmd/slint-gate \
      --measurement-summary artifacts/sli-summary.json \
      --policy .slint/policy.yaml \
      --github-step-summary

- name: 게이트 결과 확인
  run: |
    result=$(jq -r '.gate_result' slint-gate-summary.json)
    [ "$result" != "FAIL" ] || exit 1
```

---

## kind 클러스터 예제

외부 클러스터 없이 kind 로컬 클러스터로 전체 흐름을 체험할 수 있는 예제가 `examples/kind-hello-operator/`에 포함되어 있습니다.

```sh
cd examples/kind-hello-operator
./setup.sh   # kind 클러스터 생성 (클러스터만)

# setup.sh 출력의 Next steps를 따라 이미지 빌드/배포 후:
go test -tags kind ./e2e/ -v
```

`setup.sh`는 kind 클러스터 생성만 수행합니다. Docker 이미지 빌드, kind load, kubectl apply 등은 스크립트 실행 후 출력되는 Next steps 안내를 따르십시오. 상세 안내는 `examples/kind-hello-operator/README.md`를 참조하십시오.

---

## 기준선 관리

기준선은 회귀 비교를 위해 이전 게이트의 메트릭 결과를 저장합니다.

**기준선 업데이트**

```sh
make baseline-update-prepare BASELINE_SUMMARY=/path/to/sli-summary.json
```

**승인 요구사항:** 기준선 변경은 일반 기능 PR이 아닌 별도의 승인 PR에 포함되어야 합니다. 이는 성능이 저하된 실행 결과가 자동으로 회귀 기준점을 초기화하는 것을 방지합니다.

**기준선 없는 첫 실행:** `--baseline`을 지정하지 않으면 회귀 감지를 건너뛰고 `gate_result`가 `WARN`(비차단)으로 설정됩니다. 이는 온보딩 후 첫 실행에서 예상되는 동작입니다.

---

## 로컬 개발 및 테스트

```sh
# 린터 실행
./bin/golangci-lint run --timeout=10m --config=.golangci.yml ./...

# 단위 테스트 실행
go test ./...

# 모듈 일관성 확인
go mod tidy
git diff --exit-code

# slint-gate CLI 빌드
make slint-gate

# 로컬 요약으로 기준선 업데이트
make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json
```

---

## 프로젝트 문서

- 제품/품질 로드맵: `docs/quality-roadmap.md`
- 구현 handoff: `docs/quality-roadmap-implementation-handoff.md`
- 보안 모델: `docs/security-model.md`
- 게이트 계약: `docs/gate-contract.md`
- 테스트 전략: `docs/test-strategy.md`
- 릴리즈/DevEx 계획: `docs/release-devex-plan.md`

---

## 라이선스

Copyright 2026.

Apache License, Version 2.0에 따라 배포됩니다.
자세한 내용은 http://www.apache.org/licenses/LICENSE-2.0 를 참조하십시오.

본 소프트웨어는 관련 법령이나 서면 동의에 의해 별도로 명시되지 않는 한, 명시적 또는 묵시적 어떠한 종류의 보증도 없이 "있는 그대로" 배포됩니다.
