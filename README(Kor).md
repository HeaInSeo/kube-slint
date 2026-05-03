# kube-slint

쿠버네티스 오퍼레이터를 위한 shift-left operational SLI 가드레일 라이브러리입니다.

> **전환 안내:** kube-slint는 독립 실행형 오퍼레이터가 아닙니다. 임베드 가능한 Go 라이브러리 및 CLI 툴체인입니다. 이전 문서에서는 독립 오퍼레이터 모델을 언급했으나, 해당 모델은 폐기되었습니다. 현재 설계는 SLI 수집을 E2E 테스트 세션에 직접 내장하고, Go CLI 바이너리(`cmd/slint-gate`)를 통해 CI를 게이팅합니다.

---

## 정체성과 범위

### kube-slint가 하는 것

- E2E 테스트 세션 중 실행 중인 오퍼레이터에서 운영 SLI 메트릭(reconcile 비율, 워크큐 깊이, REST 클라이언트 오류)을 수집합니다.
- 수집된 메트릭을 선언적 정책(`policy.yaml`)과 비교하여 게이트 결과를 산출합니다.
- 저장된 기준선(baseline)과 비교하여 회귀를 감지합니다.
- CI 소비 및 감사를 위한 구조화된 JSON 아티팩트(`sli-summary.json`, `slint-gate-summary.json`)를 작성합니다.
- `--github-step-summary`를 통해 GitHub Actions에 마크다운 스텝 요약을 렌더링합니다.

### kube-slint가 하지 않는 것

- kube-slint는 correctness 테스트 프레임워크가 아닙니다. 오퍼레이터가 올바르게 동작하는지 단언하지 않습니다.
- kube-slint는 모니터링 또는 알림 시스템이 아닙니다. 연속적인 프로덕션 메트릭이 아닌, 테스트 실행 시점의 가드레일 결과를 산출합니다.
- kube-slint는 계측 실패 시 E2E 테스트를 실패시키지 않습니다. 메트릭 스크랩에 실패하면 미계측으로 기록되지만 테스트 세션은 계속됩니다(핵심 계약 참조).

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
import "github.com/HeaInSeo/kube-slint/test/e2e/harness"

sess := harness.NewSession(harness.SessionConfig{
    Namespace:             "my-operator-system",
    MetricsServiceName:    "my-operator-controller-manager-metrics-service",
    ArtifactsDir:          "artifacts",
    Specs:                 harness.DefaultV3Specs(),
    TLSInsecureSkipVerify: true,
    CurlImage:             "my-registry/curlimages/curl:latest",
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

`DefaultV3Specs()`(별칭: `BaselineV3Specs()`)는 kubebuilder로 생성된 오퍼레이터를 위해 설계된 프리셋 스펙 세트를 반환합니다. 다음 항목을 포함합니다:

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
specs := harness.DefaultV3Specs()
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
import "github.com/HeaInSeo/kube-slint/test/e2e/harness"

sess := harness.NewSession(harness.SessionConfig{
    // 오퍼레이터가 실행 중인 대상 네임스페이스
    Namespace: "my-operator-system",

    // /metrics를 노출하는 쿠버네티스 서비스 이름
    MetricsServiceName: "my-operator-controller-manager-metrics-service",

    // sli-summary.json이 작성될 디렉토리
    ArtifactsDir: "artifacts",

    // SLI 스펙 세트 — 프리셋 또는 커스텀 사용
    Specs: harness.DefaultV3Specs(),

    // 실클러스터 설정
    TLSInsecureSkipVerify: true,
    CurlImage:             "my-registry/curlimages/curl:latest",
})

sess.Start()
// ... 여기에 E2E 시나리오 실행 ...
sess.End(ctx)
```

**RBAC 주의사항:** 하네스는 curl 기반 페처를 사용하여 메트릭 엔드포인트를 스크랩하기 위한 임시 파드를 생성합니다. 오퍼레이터의 ServiceAccount는 대상 네임스페이스에서 `pods: create` 권한을 보유해야 합니다.

**출력:** `artifacts/sli-summary.json`은 `sess.End(ctx)` 호출 시 작성됩니다. 이 파일이 slint-gate CLI의 입력이 됩니다.

---

### 3. 결과 게이팅 (slint-gate CLI)

게이트 CLI는 `cmd/slint-gate`에 위치한 Go 바이너리입니다. 레거시 `hack/slint_gate.py` Python 스크립트를 대체하며, 해당 스크립트는 참조용으로만 보관됩니다.

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
| `--github-step-summary` | false | `$GITHUB_STEP_SUMMARY`에 마크다운 작성 (GitHub Actions용) |

**종료 동작:** 바이너리는 항상 0으로 종료합니다. CI 워크플로우는 `jq`로 출력 JSON의 `gate_result`를 검사하여 실패를 판단합니다.

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
| `WARN` | 비차단 이슈 (예: 기준선 없는 첫 실행, 신뢰도 최솟값 미달 등) |
| `FAIL` | 정책 위반 — 임계치 미달 또는 회귀 감지; CI 실패 |
| `NO_GRADE` | 평가 불가 — 입력 파일 누락 또는 손상 |

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

## CI 통합

### GitHub Composite Action (권장)

워크플로우에 한 줄로 추가할 수 있습니다:

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
          fail-on:             FAIL                          # FAIL | FAIL_OR_WARN
```

**입력값**

| 입력 | 기본값 | 설명 |
|---|---|---|
| `measurement-summary` | `artifacts/sli-summary.json` | sli-summary.json 경로 |
| `policy` | `.slint/policy.yaml` | 정책 YAML 경로 |
| `baseline` | `` | 선택적 기준선 경로 |
| `output` | `slint-gate-summary.json` | 게이트 결과 출력 경로 |
| `fail-on` | `FAIL` | `FAIL` 또는 `FAIL_OR_WARN` |
| `github-step-summary` | `true` | Markdown 요약 테이블 출력 여부 |
| `upload-artifact` | `true` | 게이트 결과 아티팩트 업로드 여부 |

**출력값**: `gate-result`, `evaluation-status`, `summary-path`

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

## 라이선스

Copyright 2026.

Apache License, Version 2.0에 따라 배포됩니다.
자세한 내용은 http://www.apache.org/licenses/LICENSE-2.0 를 참조하십시오.

본 소프트웨어는 관련 법령이나 서면 동의에 의해 별도로 명시되지 않는 한, 명시적 또는 묵시적 어떠한 종류의 보증도 없이 "있는 그대로" 배포됩니다.
