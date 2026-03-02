# kube-slint

`kube-slint`는 쿠버네티스 오퍼레이터의 오퍼레이셔널 SLI(Service Level Indicator, 서비스 수준 지수)를 추적하고 신뢰성을 검증하기 위한 순수 Go 프레임워크이자 관측성(Observability) 스택입니다.

> **중요:** 이 저장소는 독립적으로 실행되는 오퍼레이터(Standalone Operator)에서 **라이브러리 및 테스트 하네스 프레임워크**로 전환되었습니다. 기존의 `cmd/main.go` 및 `controller-runtime` 매니저 실행 구조는 완전히 제거되었습니다.

## 핵심 기능

- **SLI 선언적 스펙 정의 (`pkg/slo/spec`)**: Churn Rate(변동률), Convergence Time(수렴 시간) 등의 메트릭을 정의하고 강제할 수 있습니다.
- **E2E 테스트 하네스 (`test/e2e/harness`)**: 쿠버네티스 클러스터 내부에서 실행되는 통합 테스트 환경입니다. 시간에 따른 SLI를 평가하고, 신뢰성/엄격성 점수를 반영하여 정밀하게 포맷팅된 JSON 보고서(`summary.json`)를 생성합니다.
- **Orphan Sweeper (고아 리소스 정리)**: 테스트 인프라 환경에서 이전 런(run-id)의 자원들을 정리할 수 있도록 `report-only` 모드와 안전한 `delete` 모드를 제공하여 상태 오염을 방지합니다.

## 사용 방법

이 프로젝트는 현재 두 가지 주요 개념으로 나뉘어 구성되어 있습니다.

1. **오퍼레이터 계측 및 테스트 통합 (Go 라이브러리)**
2. **관측성 스택 배포 (Kustomize)**

### 1. 오퍼레이터 계측 및 테스트 통합 (Go Library)

개발 중인 오퍼레이터 내부나 E2E 테스트 코드에서 `pkg/slo` 기반의 하네스 라이브러리를 활용할 수 있습니다:

```sh
go get github.com/HeaInSeo/kube-slint@latest
```

**실클러스터 연동 옵션 (Real-cluster Integration):**
단순 로컬 테스트가 아닌 **실제 클러스터** 환경에 통합할 때는 자체 서명(Self-signed) 인증서 에러를 무시하거나, 프라이빗 curl 이미지를 바라보도록 `SessionConfig`를 다듬어 줄 수 있습니다.

```go
sess := harness.NewSession(harness.SessionConfig{
    Namespace: "my-operator-system",
    MetricsServiceName: "my-operator-metrics",
    Specs: mySpecs,
    
    // -- 실클러스터 연동 옵션 (Real-cluster Integration Knobs) --
    // 자체 서명 인증서 무시 (x509: certificate signed by unknown authority 방지)
    TLSInsecureSkipVerify: true, 
    // 프록시/프라이빗 이미지 레지스트리 (Docker-hub rate-limit 대비)
    CurlImage: "my-private-registry.com/curlimages/curl:latest",
})
```

> **RBAC 등 권한 주의:** `kube-slint`의 메트릭 수집기(Fetcher)는 메트릭을 긁어오기 위해 일회용 파드를 생성합니다. 따라서 이 라이브러리를 실행하는 매니저의 `ServiceAccount`에는 타겟 네임스페이스 내에 `pods`를 `create` 할 수 있는 RBAC 롤이 반드시 부여되어 있어야 합니다.

> **참고:** `kube-slint`의 Go 코드는 메트릭을 **계산하고, 평가하여 JSON 결과를 리포팅**하는 역할을 담당합니다. 그 메트릭들을 시각화하거나 수집하도록 돕는 인프라 프로비저닝은 하단 Kustomize 스택의 몫입니다.

### 2. 관측성 스택 배포 (Kustomize)

이곳에 정의된 Kustomize 리소스들은 `kube-slint` 메트릭을 모니터링하기 위해 필요한 프로메테우스 태그, 레코딩 룰(Record Rules), 대시보드 등을 제공합니다.

**원격 리소스 설치 (권장 사항)**  
현재 작업 중인 오퍼레이터 저장소의 Kustomize overlay 내부에 관측성 스택을 직접 불러와 사용할 수 있습니다.

> **주의:** `?ref=main`과 같은 브랜치 참조를 사용하지 마십시오. 재현 가능하고 불변하는 빌드를 보장하기 위해 반드시 **특정 태그나 커밋 SHA**를 지정해야 합니다.

소비자 저장소의 최상위 `kustomization.yaml`을 생성하십시오. 저희 베이스 스택은 특정 네임스페이스를 강제하지 않는 **"Zero-Assumption" 전략**을 취하고 있으므로, overlay 단에서 반영할 대상을 `namespace` 필드로 명시해야 합니다:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# (필수) 원격 스택이 배포될 네임스페이스를 주입합니다.
namespace: your-target-namespace

resources:
  # 구체적인 태그나 커밋 SHA 지정
  - github.com/HeaInSeo/kube-slint//config/default?ref=<tag or commitSHA>
```

> **ServiceMonitors 주의사항 (명시적 패치 오버라이드 전략):** 리소스 원격 가져오기(`github.com/...`) 문법 자체는 정상 동작합니다. 하지만 현재 `config/samples/prometheus` 리소스에는 `kube-slint` 특정 라벨(`app.kubernetes.io/name: kube-slint`)이 하드코딩되어 있습니다. 이를 그대로 사용하면 프로메테우스가 여러분의 오퍼레이터를 스크랩하지 못하는 **조용한 장애(Silent Failure)** 가 발생합니다.
> 
> 단기 권장 완화책으로서, 소비자 저장소 측에서 **명시적 로컬 오버라이드(Kustomize Patch)** 를 통해 `spec.selector.matchLabels`를 여러분의 오퍼레이터 이름으로 덮어씌워야 합니다. (이 구조적 한계의 근본적 개선은 당분간 보류(Deferred) 상태로 남습니다.)
> *(공식 튜토리얼과 패치 파일 예시는 [`test/consumer-onboarding/kustomize-remote-consumer/`](test/consumer-onboarding/kustomize-remote-consumer/README.md)를 참고하십시오.)*

---

## 로컬 개발 및 검증

더 이상 백그라운드 서비스(데몬)로 동작하지 않으므로, 일반적인 Go 프로젝트의 검증 표준을 따릅니다.

### 개발 및 테스트 명령어

개발과 수정을 진행한 이후에는 반드시 다음 명령어들을 실행하여 정합성을 확인해야 합니다. **Push 전 `go mod tidy`에 의한 변경(diff) 차이가 없어야 CI(lint/test)를 무사히 통과할 수 있습니다.**

- `bin/golangci-lint run --timeout=10m --config=.golangci.yml ./...` : 정적 분석 린트 검사
- `go test ./...` : 단위/통합 테스트 (E2E 하네스 시뮬레이션 포함)
- `go mod tidy` : 누락되거나 불필요한 의존성 모듈 정리
- `git diff --exit-code` : `go.mod/go.sum` 및 전체 의존성 무결성 체크

> 과거에 사용하던 `run`, `docker-build`, `deploy`, `install` 등의 빌드 배포 명령어들은 오작동 방지를 위해 안내 메시지만 출력하는 no-ops 껍데기로 남겨져 있습니다.

---

## 라이선스

Copyright 2026.

Apache License 2.0 하에 배포됩니다.
