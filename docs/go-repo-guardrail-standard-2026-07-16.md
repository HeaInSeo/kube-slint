# HeaInSeo Go 저장소 린트/테스트/가드레일/GitHub Actions 전수 조사

작성일: 2026-07-16

대상 루트: `/opt/go/src/github.com/HeaInSeo/`

대상 저장소: `NodeVault`, `podbridge5`, `JUMI`, `bori`, `sori`, `NodeSentinel`,
`NodePalette`, `artifact-handoff`, `node-artifact-runtime`, `spawner`, `tori`,
`dag-go`

주의: `kube-slint`는 이 표에서 “비교 대상 repo”가 아니라, 다른 저장소에 적용될 수
있는 가드레일 family로 다룬다. 즉 `golangci-lint`, `CodeQL`, `govulncheck`처럼
별도 G-ID를 가진 검사/정책 도구 축이다.

이 문서는 현재 각 저장소에 실제로 어떤 린트, 테스트, 보안 스캔, GitHub Actions
가드레일이 적용되어 있는지 파악하기 위한 조사 문서다.

향후 이 문서가 정본 기준서가 될 수 있으므로, 모든 항목에 고정 ID를 붙인다.
예를 들어 나중에 “`spawner`에 `G073`을 CI로 올려라”처럼 정확히 지시할 수 있어야
한다.

## 0. 먼저 보는 전체 체크판

이 장만 먼저 읽으면 문서 전체 구조를 잡을 수 있다.

아주 쉽게 말하면 이 문서는 아래 질문에 답하기 위한 표다.

- “이 repo는 CI에서 무엇을 자동으로 검사하고 있나?”
- “이 repo는 테스트가 어느 정도 촘촘한가?”
- “이 repo에 새로 보강하려면 어떤 번호를 추가하면 되나?”
- “어떤 repo는 강하고, 어떤 repo는 빈약한가?”

처음 읽는 사람은 모든 번호를 외울 필요가 없다. 아래처럼만 보면 된다.

1. 먼저 repo 이름을 찾는다.
   예: `node-artifact-runtime`을 보고 싶으면 `0.2 repo별 빠른 체크 테이블`에서 먼저 찾는다.
2. 그 repo의 “강한 축”과 “약한 축”을 읽는다.
   예: `PID1 container smoke는 있지만 build/vet/security/vuln은 Local 중심`이라고 이해한다.
3. 더 자세히 보고 싶으면 오른쪽의 `정확한 전체 ID 위치`로 내려간다.
   예: `3.9 node-artifact-runtime`으로 내려가 실제 `G001`, `G166`, `G373` 같은 ID를 확인한다.

헷갈릴 때는 이렇게 생각하면 된다.

- `G번호`: 이미 정의된 체크 항목이다. “이걸 추가해”라고 지시할 수 있다.
- `KOC번호`: Kubernetes Operator churn SLI 후보이다. 아직 적용이 아니라 검토 후보이다.
- `DPC번호`: 데이터 플레인 app churn SLI 후보이다. 아직 적용이 아니라 검토 후보이다.
- `CI`: 자동으로 막는다.
- `Local`: 명령은 있지만 자동으로 막는지는 약하다.
- `Observe`: 결과는 보지만 실패시키지는 않는다.

읽는 순서:

1. `0.1 전체 번호표`에서 번호 범위가 무엇을 뜻하는지 본다.
2. `0.2 repo별 빠른 체크 테이블`에서 각 repo의 현재 강한 축과 약한 축을 본다.
3. 필요한 repo의 정확한 ID 전체 목록은 `3. 저장소별 적용 ID 목록`으로 내려가 확인한다.
4. 특정 번호의 뜻이 궁금하면 `2. 전체 항목 카탈로그`에서 찾는다.

### 0.1 전체 번호표

이 표는 “무엇을 체크해야 하는가”를 번호 범위로 묶은 전체 지도다. 세부 ID 하나하나는
아래 `2. 전체 항목 카탈로그`에 있다.

| 번호 | 체크할 것 | 쉽게 말하면 |
|---|---|---|
| G001-G017 | GitHub Actions 실행 구조 | CI가 언제, 어떤 조건에서 돌고, artifact를 남기는지 |
| G018-G030 | CodeQL / SARIF | Go CodeQL 분석이 실제 build를 타고 결과를 업로드하는지 |
| G031-G043 | 취약점 / 보안 스캔 | govulncheck, gosec, Semgrep, 보안 report가 있는지 |
| G044-G056 | golangci-lint 실행/설치 | lint 설정, 실행 방식, 버전 고정, generated 예외 |
| G057-G088 | golangci-lint 내부 린터 | 실제 어떤 lint rule이 켜져 있는지 |
| G089-G104 | golangci-lint 세부 설정 | depguard, govet, gosec, complexity 같은 세부 정책 |
| G105-G109 | formatter / 코드 형태 | gofmt, goimports, fmt drift 검사 |
| G110-G118 | build / module 상태 | build, build tag, nested module, go mod drift |
| G119-G135 | 테스트 실행 방식 / coverage | go test, race, shuffle, coverage, threshold |
| G136-G144 | 테스트 성격 family | 정상/실패/회귀/계약/스모크/통합/성능 테스트의 큰 분류 |
| G145-G152 | protobuf / generated contract | proto lint, breaking check, generate drift |
| G153-G162 | Kubernetes manifest / cluster 검증 | kube-linter, kubeconform, kind smoke, K8s contract |
| G163-G180 | runtime / VM / 제품 smoke / release | VM, registry, CLI, release, benchmark 흐름 |
| G181-G194 | GitHub Actions 세부 실행 구조 | permissions, matrix, needs, manual inputs, runner 운영 세부 |
| G195-G203 | repo 고유 quality / readiness | repo-specific quality, readiness, release guardrail |
| G204-G217 | golangci 실행 세부 | action version, timeout, generated/test exception 세부 |
| G218-G235 | depguard 정책 세부 | import boundary, K8s/runtime/library boundary |
| G236-G257 | coverage / golden / protobuf 세부 | package exclude, threshold, golden, fixture, proto 세부 |
| G258-G280 | Kubernetes/runtime/benchmark 세부 | kind artifact, registry, release build, benchmark 세부 |
| G281-G327 | kube-slint / 실패 경로 세부 | SLI 측정/gate/coverage governance와 실패 유형 분류 |
| G328-G333 | fuzz 테스트 | fuzz target, corpus, CI fuzz, artifact |
| G334-G350 | 후보 승격 / 운영 세부 감사 | golden update, smoke artifact, CodeQL query, Action drift |
| G351-G362 | operator behavior 테스트 | operator가 CR을 받았을 때 실제 행동을 검증하는 테스트 |
| G363-G370 | CI 운영 세부 / test double | self-hosted runner, go vet, artifact always, httptest/sqlmock/testify |
| G371-G390 | 운영 거버넌스 후보 | CODEOWNERS, release note, provenance, SBOM, branch protection 등 |
| G391-G440 | Kubernetes Operator 표준 후보 | envtest, CRD, RBAC, webhook, finalizer, upgrade, uninstall 등 |
| G441-G460 | 컨테이너 데이터 플레인 / PID1 후보 | signal, child reap, zombie, process group, stdout drain 등 |
| KOC-001-KOC-032 | Kubernetes/operator churn SLI 후보 | reconcile, workqueue, API server, child resource churn 지표 후보 |
| DPC-001-DPC-060 | 데이터 플레인 app churn SLI 후보 | request/job/artifact/cache/process churn 지표 후보 |

### 0.2 repo별 빠른 체크 테이블

이 표는 “지금 repo별로 무엇이 강하고, 무엇을 조심해야 하는가”를 빠르게 보는 표다.
정확한 전체 G-ID 목록은 각 repo의 `3.x` 절을 기준으로 한다.

| repo | 현재 강한 축 | 현재 약하거나 주의할 축 | 정확한 전체 ID 위치 |
|---|---|---|---|
| NodeVault | CodeQL, govulncheck hard gate, golangci 세부 린트, protobuf, kube-linter, NodeVault SLI gate | SLI는 Go test 직접 assert 중심이고, slint-gate CLI gate는 아님 | `3.1 NodeVault` |
| podbridge5 | VM runtime/integration, build tag, race, rich golangci, runtime smoke | govulncheck는 observe이고, nested/runtime 환경 의존성이 큼 | `3.2 podbridge5` |
| JUMI | workflow 폭 넓음, Semgrep, quality guardrail, registry/remote smoke, kube-slint SLI/policy | 많은 workflow가 있어 운영 복잡도가 높고, 일부 보안 검사는 observe | `3.3 JUMI` |
| bori | Kubernetes/operator behavior, kind smoke, kube-linter/kubeconform, kube-slint summary artifact | security/vuln hard gate는 상대적으로 약함 | `3.4 bori` |
| sori | lint/depguard, release workflow, race, OCI/library boundary, runtime smoke | security는 observe 중심이고, 일부 build/test target은 Local | `3.5 sori` |
| NodeSentinel | actionlint, govulncheck hard gate, race/shuffle/coverage threshold, protobuf/K8s contract, Trivy contract test | operator/kind behavior 체계는 별도 확인 안 됨 | `3.6 NodeSentinel` |
| NodePalette | 작은 repo지만 CI build/vet/race/coverage/K8s contract가 잘 있음 | govulncheck는 observe, 고급 runtime/operator 축은 없음 | `3.7 NodePalette` |
| artifact-handoff | protobuf contract, buf breaking/generate drift, artifact/domain tests, coverage threshold | security/vuln은 Local/Partial, runtime smoke는 약함 | `3.8 artifact-handoff` |
| node-artifact-runtime | PID1 container smoke, release notes/provenance, runtime helper tests | build/vet/security/vuln은 Local 중심, PID1 artifact 보존은 제한적 | `3.9 node-artifact-runtime` |
| spawner | security observe, lint/depguard, race, lifecycle/failure path tests | lifecycle/race 전용 target 일부는 Local/Partial | `3.10 spawner` |
| tori | core CI, security observe, buf/proto, sqlmock, historical skip marker, contract tests | core scope 중심이라 full runtime/transport 검증은 제한적 | `3.11 tori` |
| dag-go | benchmark, coverage Pages, depguard boundary, race, performance history 일부 | security는 observe, 일부 benchmark 비교는 Local | `3.12 dag-go` |

### 0.3 빠른 읽기 가이드

이 문서는 단순한 목록이 아니라, 저장소별 품질 체계를 같은 언어로 비교하기 위한
기준표다. 읽을 때는 아래 순서가 가장 쉽다.

1. 먼저 `2. 전체 항목 카탈로그`에서 G-ID의 의미를 확인한다.
   예를 들어 `G122`는 race test이고, `G070`은 golangci-lint 안에서 gosec을
   켰다는 뜻이다.
2. 그 다음 `3. 저장소별 적용 ID 목록`에서 각 repo가 어떤 G-ID를 갖고 있는지 본다.
   여기에는 CI에서 강제되는 항목과 Local/Partial 항목이 분리되어 있다.
3. `4-10`장의 비교표와 버전 감사표로 같은 계열의 항목을 repo 간에 빠르게 비교한다.
   예를 들어 실패 경로 테스트는 `G137`만 보면 너무 넓으므로, `G306-G325`를 같이
   봐야 실제 방어 범위를 알 수 있다.
4. 마지막으로 `11. 조사상 중요한 주의점`을 읽는다.
   이 장은 이 문서를 적용 지침으로 사용할 때 오해하기 쉬운 부분을 정리한다.

### 이 문서에서 “적용됨”의 의미

“적용됨”은 보통 세 단계로 나뉜다.

- `CI`: GitHub Actions에서 실행되어 실패하면 PR/push가 막힌다. 가장 강한 상태다.
- `Local`: Makefile, script, config에는 있지만 CI gate로 확인되지는 않았다. 사람이
  실행할 수는 있지만 자동 강제력은 약하다.
- `Partial`: 일부 module, package, build tag, workflow에만 적용된다. 없는 것보다는
  낫지만 repo 전체를 보호한다고 보면 안 된다.

따라서 어떤 repo에 “`G122`가 있다”고만 말하면 부족하다. `CI G122`인지,
`Local G122`인지, 특정 package에만 적용된 `Partial G122`인지까지 같이 봐야 한다.

### 학습용으로 읽는 법

처음 보는 항목은 세 가지 질문으로 이해하면 쉽다.

- 이 항목은 무엇을 막는가?
- 실패하면 개발자는 무엇을 고치게 되는가?
- CI에서 강제해야 하는가, 아니면 관찰 artifact만 남겨도 되는가?

예를 들어 `govulncheck`는 취약한 Go dependency나 표준 라이브러리 사용을 찾는다.
하지만 모든 repo에서 처음부터 hard fail로 두면 기존 취약점 때문에 개발이 막힐 수
있다. 그래서 어떤 repo는 `G036` hard fail이고, 어떤 repo는 `G037` observe로 시작할
수 있다. 이 문서는 그런 차이를 숫자로 기록하기 위한 기준이다.

## 1. 표기 규칙

- `Gxxx`: 고정 가드레일 ID. 나중에 새 항목이 추가되어도 기존 ID 의미를 바꾸지 않는다.
- `CI`: GitHub Actions에서 실행되는 것으로 확인됨.
- `Local`: Makefile/config/script에는 있으나 정규 GitHub Actions gate로는 확인하지 못함.
- `Observe`: 결과는 남기지만 실패를 강제하지 않거나 관찰 목적 workflow로 확인됨.
- `Partial`: 특정 package, build tag, module, scope에만 적용됨.
- `Unknown`: workflow/Makefile/config 수준 조사만으로는 세부 적용 여부가 불명확함.

표의 각 열은 다음 의미다.

- `ID`: 나중에 작업 지시나 이슈에서 그대로 사용할 고정 번호다.
- `항목`: 사람이 부를 이름이다. 도구 이름, 테스트 방식, 정책 종류가 들어간다.
- `확인 기준`: repo에서 무엇을 찾으면 이 항목이 있다고 판단할지 적은 기준이다.
- `쉬운 설명`: 해당 항목이 왜 필요한지 한 문장으로 풀어쓴 설명이다.

이 문서의 목적은 “좋다/나쁘다”를 즉시 판정하는 것이 아니다. 먼저 각 repo의 현재
상태를 같은 단위로 관찰하고, 이후 repo 성격에 맞게 어떤 항목을 승격할지 정하기
위한 것이다.

### 1.1 번호 granularity 읽는 법

이 문서의 번호는 모두 같은 크기의 개념이 아니다. 일부는 아주 작은 도구 설정이고,
일부는 여러 세부 검사를 묶는 넓은 family다. 그래서 번호를 읽을 때는 아래 4단계로
구분한다.

| 단계 | 뜻 | 예시 | 읽는 법 |
|---|---|---|---|
| L1 Family | 넓은 테스트/가드레일 family | `G137` 실패 경로 테스트, `G281` kube-slint SLI 측정 harness | “이 축이 있다”는 큰 분류다. 단독으로는 세부 수준이 부족할 수 있다. |
| L2 Capability | 독립 적용 가능한 기능/검사 | `G122` race test, `G171` kube-slint SLI gate, `G391` envtest | repo에 추가하라고 지시하기 좋은 기본 단위다. |
| L3 Implementation Detail | 같은 capability 안의 구현 차이 | `G154` kube-linter action, `G155` kube-linter CLI, `G262` kube-linter action | 실행 방식, 설치 방식, artifact 방식처럼 세부 차이를 구분한다. |
| L4 Domain Sub-ID | 특정 domain의 하위 후보/지표 | `KOC-008`, `DPC-052` | 전역 G-ID는 아니며, SLI 후보나 세부 검토 후보를 지칭하기 위한 보조 번호다. |

이 구분이 중요한 이유는 간단하다. 예를 들어 “`G137`을 추가하라”는 말은 너무 넓다.
실제로는 `G314 timeout/deadline failure`, `G323 leak/deadlock/cleanup failure path`
같은 세부 항목을 골라야 한다. 반대로 `G154`와 `G155`는 둘 다 kube-linter지만,
하나는 GitHub Action 방식이고 하나는 CLI 방식이므로 둘을 같은 항목으로 합치면
운영 차이가 사라진다.

평탄화 원칙:

- 기존 번호는 의미를 바꾸거나 재사용하지 않는다.
- 넓은 번호는 “family”로 남기고, 부족한 부분은 하위 세부 번호로 보완한다.
- 특정 repo의 도메인 SLI 이름은 전역 G-ID로 올리지 않는다. `KOC-*`, `DPC-*` 같은
  하위 번호로 둔다.
- 중복처럼 보이는 항목은 먼저 “정말 같은가”를 본다. 실행 위치, 실패 강도, artifact
  보존 여부가 다르면 별도 번호로 유지한다.

## 2. 전체 항목 카탈로그

### 2.1 GitHub Actions 실행 구조

이 그룹은 “검사가 실제로 언제, 어떤 조건으로 자동 실행되는가”를 본다. 같은 테스트가
있더라도 PR에서 자동으로 돌지 않으면 보호력이 낮다. 반대로 너무 넓게 실행하면 CI가
느려지고 개발자가 우회하고 싶어질 수 있으므로 trigger, path filter, concurrency,
artifact 보존 여부를 함께 봐야 한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G001 | GitHub Actions workflow | `.github/workflows/*` 존재 | 자동 검사 파일이 있는지 |
| G002 | push trigger | `on: push` | push 때 자동 실행되는지 |
| G003 | pull_request trigger | `on: pull_request` | PR 때 자동 실행되는지 |
| G004 | schedule trigger | `on: schedule` | 정기적으로 자동 실행되는지 |
| G005 | workflow_dispatch trigger | `on: workflow_dispatch` | 사람이 수동 실행할 수 있는지 |
| G006 | path filter | `paths`, `paths-ignore` | 관련 파일 변경 때만 실행되도록 제한하는지 |
| G007 | concurrency | `concurrency:` | 같은 branch/ref의 중복 실행을 줄이는지 |
| G008 | latest checkout/setup-go | `actions/checkout@v6`, `actions/setup-go@v6` | 최신 계열 GitHub Action을 쓰는지 |
| G009 | older checkout/setup-go | `checkout@v4`, `setup-go@v5` 등 | 구버전 계열이 남아 있는지 |
| G010 | artifact upload | `actions/upload-artifact` | coverage/log/report를 남기는지 |
| G011 | artifact download | `actions/download-artifact` | 이전 job 결과를 받아 쓰는지 |
| G012 | GitHub Pages publish | `peaceiris/actions-gh-pages`, benchmark Pages 등 | coverage/benchmark를 Pages로 게시하는지 |
| G013 | environment gate | `environment:` | 특정 GitHub environment 승인/secret을 쓰는지 |
| G014 | secret validation | secret 존재 확인 step | secret 누락을 명확히 실패시키는지 |
| G015 | SSH agent setup | `webfactory/ssh-agent` 등 | 원격 VM 접근을 위해 SSH agent를 구성하는지 |
| G016 | known_hosts setup | `ssh-keyscan`, known_hosts 처리 | 원격 SSH 대상 host key를 준비하는지 |
| G017 | actionlint | `actionlint` 실행 | workflow 문법/표현 오류를 검사하는지 |

### 2.2 CodeQL / SARIF

CodeQL은 GitHub의 정적 보안 분석이다. Go repo에서는 단순히 workflow만 있어서는
부족하고, CodeQL이 실제 Go package를 build해서 분석할 수 있어야 한다. 특히 build
tag, nested module, local replace가 있는 repo는 `build-mode: manual`과 repo-specific
build command가 중요하다.

SARIF는 분석 결과 파일 형식이다. 기본 원칙은 `analyze@v4`의 자동 업로드를 유지하는
것이다. 수동 SARIF upload나 필터는 generated source 노이즈가 실제로 증명된 repo에서만
예외적으로 검토한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G018 | CodeQL workflow | `codeql.yml` 존재 | CodeQL 보안 분석 workflow가 있는지 |
| G019 | CodeQL init v4 | `github/codeql-action/init@v4` | CodeQL 초기화 액션 |
| G020 | CodeQL analyze v4 | `github/codeql-action/analyze@v4` | CodeQL 분석 액션 |
| G021 | CodeQL Go manual build mode | `build-mode: manual` | Go 파일을 분석하려고 직접 build를 걸었는지 |
| G022 | CodeQL config file | `config-file: ./.github/codeql/codeql-config.yml` | CodeQL 설정 파일을 따로 쓰는지 |
| G023 | CodeQL root `go build ./...` | CodeQL job에서 `go build ./...` | root module 전체를 build하는지 |
| G024 | CodeQL build tags | CodeQL build에 `-tags` 사용 | build tag가 필요한 repo에서 tag를 반영하는지 |
| G025 | CodeQL repo-specific build | `go build ./config ...`, 특정 `cmd/...` 등 | 전체가 아니라 의도한 scope만 build하는지 |
| G026 | CodeQL nested module build | `cd tools/... && go build`, `hack/...` build | nested module도 분석 build에 포함하는지 |
| G027 | CodeQL local replacement preparation | `Prepare local replacements` | local replace/sibling repo 준비가 필요한지 |
| G028 | CodeQL automatic SARIF upload | `analyze@v4` 기본 업로드 | 별도 manual SARIF upload 없이 자동 업로드하는지 |
| G029 | CodeQL manual SARIF upload | `upload-sarif` | raw/filter/manual SARIF 구조를 쓰는지 |
| G030 | CodeQL SARIF filter | custom SARIF filter script | generated alert 노이즈를 별도 필터링하는지 |

### 2.3 취약점 / 보안 스캔

이 그룹은 “코드가 지금 알려진 취약점이나 위험한 패턴을 포함하는가”를 본다.
`govulncheck`는 Go 생태계 취약점 데이터베이스와 실제 call graph를 함께 사용한다.
`gosec`과 Semgrep은 코드 패턴을 본다. 취약점/보안 스캔은 처음 도입할 때 hard fail이
적절한지, observe로 시작해야 하는지 repo별 판단이 필요하다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G031 | `govulncheck` 설치 | `go install golang.org/x/vuln/cmd/govulncheck` | Go 취약점 검사 도구를 설치하는지 |
| G032 | `govulncheck` 실행 | `govulncheck ./...` 또는 package scope | Go 취약점 검사를 실행하는지 |
| G033 | `govulncheck` build tags | `govulncheck -tags ...` | build tag 환경까지 반영하는지 |
| G034 | `govulncheck` JSON report | `-format json` | machine-readable report를 만드는지 |
| G035 | `govulncheck` exception gate | repo-specific exception checker | 허용 목록에 없는 취약점이면 실패시키는지 |
| G036 | `govulncheck` hard fail | `continue-on-error` 없음 | 취약점 scan 실패가 CI 실패로 이어지는지 |
| G037 | `govulncheck` observe | `continue-on-error`, `security-observe`, summary 저장 | 결과만 관찰하는지 |
| G038 | `govulncheck` report artifact | artifact로 report 업로드 | 취약점 결과를 다운로드 가능하게 남기는지 |
| G039 | `gosec` hard lint | `gosec` 결과가 lint 실패로 이어짐 | 보안 lint를 실패 기준으로 쓰는지 |
| G040 | `gosec` observe lint | `lint-security` report/summary | 보안 lint를 관찰로 쓰는지 |
| G041 | Semgrep scan | `semgrep scan --config .semgrep/rules` | custom 보안/정책 rule을 검사하는지 |
| G042 | Semgrep rule test | `semgrep --test .semgrep/rules` | Semgrep rule 자체의 fixture를 테스트하는지 |
| G043 | security report artifact | security reports artifact | 보안 lint/vuln 결과를 artifact로 남기는지 |

### 2.4 golangci-lint 실행/설치

이 그룹은 golangci-lint 자체를 어떻게 실행하는지 본다. 중요한 점은 “린터가 있다”가
아니라 “같은 버전, 같은 설정, 같은 scope로 반복 실행되는가”다. CI와 로컬이 다른
버전을 쓰면 개발자 환경에서는 통과하지만 CI에서는 실패하는 문제가 생긴다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G044 | `.golangci.yml` config | `.golangci.yml` 또는 `.golangci.yaml` | 통합 린트 설정 파일이 있는지 |
| G045 | golangci-lint GitHub Action | `golangci/golangci-lint-action` | GitHub Action으로 린트를 돌리는지 |
| G046 | `make lint` workflow | workflow에서 `make lint` | repo Makefile의 lint 계약을 CI에서 쓰는지 |
| G047 | lint config verify | `golangci-lint config verify` | 린트 설정 자체를 검사하는지 |
| G048 | local pinned golangci install | Makefile에서 특정 version install | 로컬/CI 도구 버전을 고정하는지 |
| G049 | checksum verified golangci install | checksum file 확인 | 다운로드한 린트 바이너리를 검증하는지 |
| G050 | source install golangci | `go install github.com/golangci/...` | Go toolchain으로 린터를 설치하는지 |
| G051 | golangci build tags | `--build-tags`, config `build-tags` | build tag가 필요한 lint를 처리하는지 |
| G052 | golangci `--fix` target | `make lint-fix`, `--fix` | 자동 수정 경로가 있는지 |
| G053 | generated exclusions | `exclusions.generated` | generated file에 린트 예외를 두는지 |
| G054 | test file lint relaxations | `_test.go` exclusions | 테스트 파일에는 일부 규칙을 완화하는지 |
| G055 | path-based lint exclusions | `paths`, `path:` rules | vendor/generated/specific file 예외를 두는지 |
| G056 | max issues unbounded | `max-issues-per-linter: 0` | 린트 결과를 중간에 잘라내지 않는지 |

### 2.5 golangci-lint 내부 린터

golangci-lint는 여러 린터를 한 번에 실행하는 wrapper다. 그래서 `golangci-lint를 쓴다`
는 말만으로는 부족하다. 실제로 어떤 내부 린터를 켰는지가 품질 수준을 결정한다.
예를 들어 `errcheck`는 에러 무시를 잡고, `bodyclose`는 HTTP body close 누락을 잡고,
`depguard`는 package 의존성 경계를 강제한다.

| ID | 린터 | 확인 기준 | 무엇을 잡는가 |
|---|---|---|---|
| G057 | `bodyclose` | `enable: bodyclose` | HTTP response body 미닫힘 |
| G058 | `copyloopvar` | `enable: copyloopvar` | loop variable capture |
| G059 | `depguard` | `enable: depguard` | import boundary 위반 |
| G060 | `dupl` | `enable: dupl` | 중복 코드 |
| G061 | `errcheck` | `enable: errcheck` | error 반환값 무시 |
| G062 | `errorlint` | `enable: errorlint` | error wrapping/comparison 문제 |
| G063 | `exhaustive` | `enable: exhaustive` | enum/switch 누락 |
| G064 | `funlen` | `enable: funlen` | 너무 긴 함수 |
| G065 | `gocheckcompilerdirectives` | `enable: gocheckcompilerdirectives` | compiler directive 형식 오류 |
| G066 | `gocognit` | `enable: gocognit` | 인지 복잡도 |
| G067 | `goconst` | `enable: goconst` | 반복 literal 상수화 후보 |
| G068 | `gocritic` | `enable: gocritic` | diagnostic/performance/style 문제 |
| G069 | `gocyclo` | `enable: gocyclo` | cyclomatic complexity |
| G070 | `gosec` | `enable: gosec` | 보안 위험 패턴 |
| G071 | `govet` in golangci | `enable: govet` | Go vet 분석을 golangci에서 실행 |
| G072 | `ineffassign` | `enable: ineffassign` | 효과 없는 assignment |
| G073 | `lll` | `enable: lll` | 긴 줄 |
| G074 | `misspell` | `enable: misspell` | 오타 |
| G075 | `nakedret` | `enable: nakedret` | naked return |
| G076 | `nilerr` | `enable: nilerr` | error 위치에서 nil 반환 |
| G077 | `noctx` | `enable: noctx` | context 없는 HTTP 요청 |
| G078 | `nolintlint` | `enable: nolintlint` | 부정확한 nolint 주석 |
| G079 | `prealloc` | `enable: prealloc` | slice pre-allocation 후보 |
| G080 | `revive` | `enable: revive` | style/quality 규칙 |
| G081 | `rowserrcheck` | `enable: rowserrcheck` | `sql.Rows.Err()` 누락 |
| G082 | `sqlclosecheck` | `enable: sqlclosecheck` | SQL rows/stmt close 누락 |
| G083 | `staticcheck` | `enable: staticcheck` | 강한 Go 정적 분석 |
| G084 | `tparallel` | `enable: tparallel` | 테스트 병렬화 패턴 |
| G085 | `unconvert` | `enable: unconvert` | 불필요한 type conversion |
| G086 | `unparam` | `enable: unparam` | 불필요한 parameter |
| G087 | `unused` | `enable: unused` | 사용되지 않는 코드 |
| G088 | `wrapcheck` | `enable: wrapcheck` | 외부 error context wrapping 누락 |

### 2.6 golangci-lint 세부 설정

이 그룹은 린터의 민감도와 예외 규칙을 본다. 같은 린터를 켜도 threshold, 제외 규칙,
depguard 정책에 따라 실제 강도는 크게 달라진다. 좋은 설정은 “많이 잡는 것”만이 아니라
repo 구조에 맞게 noisy한 부분을 설명 가능하게 제한하는 것이다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G089 | `govet` nilness | `govet.enable: nilness` | nil 관련 vet 분석 |
| G090 | `govet` shadow | `govet.enable: shadow` | 변수 shadowing 검사 |
| G091 | `govet` copylocks | `govet.enable: copylocks` | lock/atomic 포함 값 복사 검사 |
| G092 | `govet` enable-all | `govet.enable-all: true` | vet analyzer를 넓게 켬 |
| G093 | `gosec` exclusions | `gosec.excludes` | repo 의도상 필요한 보안 예외 |
| G094 | `gosec` G101 config | `gosec.config.G101` | secret 탐지 임계치/패턴 조정 |
| G095 | `gocritic` tags | `gocritic.enabled-tags` | diagnostic/performance/style tag 사용 |
| G096 | `gocritic` disabled checks | `gocritic.disabled-checks` | repo 특성상 noisy check 제외 |
| G097 | `revive` custom rules | `revive.rules` | revive 규칙을 명시적으로 설정 |
| G098 | `depguard` K8s boundary | `k8s.io/**`, `sigs.k8s.io/**` deny | Kubernetes 의존성 경계 강제 |
| G099 | `depguard` service/transport boundary | service/transport deny | core가 상위 계층을 import하지 못하게 함 |
| G100 | `depguard` allowlist | `allow:` 사용 | 허용 import를 명시 |
| G101 | complexity threshold | `gocyclo`, `gocognit` threshold | 복잡도 기준 설정 |
| G102 | line length threshold | `lll.line-length` | 줄 길이 기준 설정 |
| G103 | errcheck exclusions | `errcheck.exclude-functions` | Close/Remove 등 예외 |
| G104 | staticcheck exclusions | `staticcheck.checks` with negative checks | 특정 staticcheck 제외 |

### 2.7 formatter / 코드 형태

formatter는 코드 스타일 논쟁을 줄이는 가장 기본적인 가드레일이다. `gofmt`는 Go 기본
형태를 맞추고, `goimports`는 import 정렬과 사용하지 않는 import 정리를 함께 한다.
포맷은 보통 hard fail로 두어도 부담이 낮고 효과가 크다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G105 | `gofmt` formatter | `formatters.enable: gofmt` 또는 `make fmt` | Go 기본 포맷 |
| G106 | `goimports` formatter | `formatters.enable: goimports` | import 정리 |
| G107 | local import prefix | `goimports.local-prefixes` | repo import 정렬 기준 |
| G108 | fmt target | `make fmt` | 포맷 실행 target |
| G109 | fmt-check target | `make fmt-check` | 포맷 drift 검사 target |

### 2.8 build / module 상태

이 그룹은 “repo가 실제로 build 가능한 상태인가”를 본다. 테스트가 일부 package만
통과해도 전체 build가 깨져 있으면 사용자 입장에서는 실패다. nested module, build tag,
local replace가 있는 repo는 단순 `go build ./...`만으로 충분하지 않을 수 있다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G110 | root `go build ./...` | `go build ./...` | 전체 package build |
| G111 | binary build | `go build -o bin/... ./cmd/...` | 실제 바이너리 build |
| G112 | repo-specific build scope | 특정 package/cmd만 build | 의도된 scope만 build |
| G113 | build tags | `go build -tags ...` | build tag 반영 |
| G114 | CGo/system deps setup | apt install 등 system dependency | CGo/library 필요 조건 준비 |
| G115 | nested module build | root 밖 module build | tools/hack module build |
| G116 | `go mod tidy` drift check | `go mod tidy` 후 diff | go.mod/go.sum drift 방지 |
| G117 | GOPROXY/offline build/test | `GOPROXY=off` | 의존성 network drift 방지 |
| G118 | local replacement setup | sibling/local module 준비 | local replace를 CI에서 맞춤 |

### 2.9 테스트 실행 방식

이 그룹은 테스트를 어떤 방식으로 실행하는지 본다. 같은 테스트라도 race, shuffle,
no-cache, repeated 실행을 추가하면 잡히는 문제가 달라진다. coverage는 테스트가 충분한지
보는 보조 지표지만, coverage 숫자가 높다고 실패 경로가 충분하다는 뜻은 아니다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G119 | `go test` 기본 실행 | `go test`, `make test` | 기본 테스트 |
| G120 | `go test ./...` 전체 테스트 | `go test ./...` | 전체 package 테스트 |
| G121 | scope-limited test | 특정 package list | 일부 package만 테스트 |
| G122 | race test | `-race` | data race 탐지 |
| G123 | shuffle test | `-shuffle=on` | 테스트 순서 의존성 탐지 |
| G124 | no-cache test | `-count=1` | 캐시 없이 실행 |
| G125 | repeated test | `-count=N` | 반복으로 flaky/race 탐지 |
| G126 | short test | `-short` | 긴 테스트 제외 |
| G127 | build-tagged test | `go test -tags ...` | 특정 build tag 테스트 |
| G128 | compile-only tagged test | runtime/integration tag vet/build | 실행 어려운 tag 조합의 컴파일 확인 |
| G129 | coverage basic | `-cover` | coverage 측정 |
| G130 | coverage profile | `-coverprofile` | coverage 파일 생성 |
| G131 | coverage report text | `go tool cover -func` | 사람이 읽는 coverage summary |
| G132 | coverage artifact | coverage artifact 업로드 | coverage 결과 보존 |
| G133 | coverage threshold | threshold 미만 실패 | 테스트 양이 줄어드는 것을 막음 |
| G134 | coverage HTML | `go tool cover -html` | HTML coverage 생성 |
| G135 | coverage Pages publish | Pages에 coverage 게시 | coverage를 웹에서 확인 |

### 2.10 테스트 성격

이 그룹은 테스트의 “목적”을 분류한다. 실행 명령만 보면 `go test` 하나로 보이지만,
그 안에는 정상 경로, 실패 경로, 회귀, 계약, 스모크, 성능 테스트가 섞여 있다. repo를
상향 평준화하려면 단순 테스트 개수보다 어떤 성격의 테스트가 비어 있는지를 봐야 한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G136 | 골든 패스 단위 테스트 | 정상 입력/정상 상태 unit test | 제대로 썼을 때 되는지 |
| G137 | 실패 경로 테스트 | invalid/missing/error/timeout test | 문제가 생겼을 때 안전하게 실패하는지 |
| G138 | 회귀 테스트 | `test-regression`, bug-specific test | 고친 버그가 다시 생기지 않는지 |
| G139 | 골든 파일/fixture 테스트 | golden/fixture compare/update | 출력물이 기대 파일과 같은지 |
| G140 | 계약 테스트 | API/proto/K8s/cross-repo contract | 외부와의 약속이 깨지지 않는지 |
| G141 | 스모크 테스트 | smoke workflow/target | 켜지고 기본 동작이 되는지 |
| G142 | 통합 테스트 | integration/kind/VM/registry | 여러 구성요소가 같이 동작하는지 |
| G143 | race/lifecycle 테스트 | race/lifecycle/repeated/cancel cleanup | 동시성/lifecycle 문제를 보는지 |
| G144 | 성능/벤치마크 테스트 | benchmark workflow/target | 성능 변화를 기록하는지 |

### 2.10b 실패 경로 테스트 세부 유형

`G137`은 실패 경로 테스트가 있다는 큰 분류다. 실제로 어떤 실패를 다루는지는
아래 세부 ID로 판단한다.

실패 경로 테스트는 “나쁜 입력을 넣었을 때 에러가 난다”만 확인하면 부족하다. 좋은 실패
테스트는 에러가 적절한 타입/메시지로 전파되는지, cleanup이 되는지, retry가 멈추는지,
보안상 위험한 입력을 차단하는지까지 본다. 이 표는 각 repo의 방어 범위를 비교하기 위해
실패 유형을 더 잘게 쪼갠 것이다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G306 | malformed/invalid parse failure | `InvalidJSON`, `InvalidYAML`, `MalformedJSON`, invalid binary | 깨진 입력을 거부하는지 |
| G307 | missing/not-found failure | `Missing*`, `NotFound`, file not found | 없는 리소스/파일/ID를 안전하게 처리하는지 |
| G308 | duplicate/conflict failure | `Duplicate*`, `Conflict`, collision | 중복/충돌을 잡는지 |
| G309 | empty/nil required input failure | `Empty*`, `Nil*`, missing required field | 필수 값이 비었을 때 막는지 |
| G310 | negative/out-of-range failure | `Negative*`, invalid size/time/index/range | 음수나 범위 밖 값을 막는지 |
| G311 | unsupported/unknown value failure | `Unsupported*`, `Unknown*`, wrong enum/schema | 모르는 값이나 지원하지 않는 값을 막는지 |
| G312 | auth/permission/security rejection | 401/permission/forbidden/reject unsafe/security policy | 권한/보안 위반을 막는지 |
| G313 | remote/http/registry/server failure | server error, registry error, HTTP error, unreachable | 외부 서비스 실패를 전파하는지 |
| G314 | timeout/deadline failure | `Timeout`, deadline exceeded | 시간 초과를 실패로 처리하는지 |
| G315 | context cancellation failure | `Cancel`, `Canceled`, `Cancelled`, context cancel | 취소 신호를 처리하는지 |
| G316 | upstream/dependency failure propagation | builder/store/finalize/notify/resolve failure propagation | 하위 의존성 실패를 삼키지 않는지 |
| G317 | store/db/io failure | store error, DB query error, file IO error | 저장소/DB/파일 오류를 처리하는지 |
| G318 | lifecycle failed-event propagation | failed event, failed node, failed run, failure reason | lifecycle 실패 상태를 정확히 전파하는지 |
| G319 | no-data/no-op edge case | no match, no data, zero, no output, no runner | 데이터가 없을 때 오동작하지 않는지 |
| G320 | schema/contract validation failure | schema version, manifest contract, binding contract | 계약 위반을 막는지 |
| G321 | unsafe path/digest/remote-source rejection | path escape, digest mismatch, signed URL/query, disallowed host | 위험한 경로/원격 입력을 막는지 |
| G322 | retry/recovery failure path | retry budget exhausted, retry after unknown outcome | 재시도/복구 경로가 안전한지 |
| G323 | leak/deadlock/cleanup failure path | no leak, no deadlock, cleanup, subprocess kill | 자원 누수/교착/cleanup 실패를 막는지 |
| G324 | bad fixture/matrix failure | `BadFixture`, invalid fixture matrix | fixture 기반 실패 경로를 모아 검증하는지 |
| G325 | policy/admission rejection | policy reject, reserved label, invalid policy, admission reject | 정책상 허용되지 않는 입력을 막는지 |

### 2.11 protobuf / generated contract

protobuf를 쓰는 repo에서는 schema가 곧 외부 계약이다. generated file이 최신인지,
schema 변경이 하위 호환성을 깨는지, 외부 proto import 경계가 유지되는지를 확인해야
한다. generated code는 자동 생성됐다는 이유만으로 무조건 무시하면 안 된다. repo에
커밋되고 실제 바이너리에 포함된다면 제품 코드의 일부로 다뤄야 한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G145 | `buf.yaml` | `buf.yaml` 존재 | protobuf lint 설정 |
| G146 | `buf.gen.yaml` | `buf.gen.yaml` 존재 | protobuf generate 설정 |
| G147 | Buf lint via action | `bufbuild/buf-action` | Action으로 Buf lint |
| G148 | Buf lint via CLI | `buf lint` | CLI로 Buf lint |
| G149 | Buf breaking check | `buf breaking` | schema 호환성 깨짐 방지 |
| G150 | protobuf generate | `buf generate`, `protoc-gen-go` | generated code 생성 |
| G151 | protobuf/generated drift check | generate 후 diff | generated code가 최신인지 |
| G152 | external proto import guardrail | proto import boundary test | 외부 API import 경계 확인 |

### 2.12 Kubernetes manifest / cluster 검증

Kubernetes 관련 repo는 Go 코드가 통과해도 manifest가 잘못되면 실제 배포에서 실패한다.
이 그룹은 manifest lint, schema validation, kind 기반 smoke를 분리해 본다. `kube-linter`
는 운영상 위험한 manifest 패턴을 찾고, `kubeconform`은 schema 적합성을 본다. kind smoke는
실제 cluster 가까운 환경에서 기본 동작을 확인한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G153 | `.kube-linter.yaml` | config file 존재 | kube-linter 설정 |
| G154 | kube-linter action | `stackrox/kube-linter-action` | Action으로 manifest lint |
| G155 | kube-linter CLI | `kube-linter lint`, `make kube-lint` | CLI로 manifest lint |
| G156 | kubeconform | `kubeconform` | Kubernetes schema validation |
| G157 | generated manifest drift | `generate-check` | 생성된 manifest drift 확인 |
| G158 | kind install | workflow에서 kind 설치 | CI에서 kind cluster 준비 |
| G159 | kind boot smoke | boot smoke workflow/script | operator 기동/metrics 확인 |
| G160 | kind functional smoke | functional smoke workflow/script | 실제 리소스 생성 확인 |
| G161 | kind digest smoke | digest smoke workflow/script | digest 기반 동작 확인 |
| G162 | Kubernetes contract test | `test/k8s`, `test-k8s` | K8s/data-plane 계약 테스트 |

### 2.13 runtime / VM / 제품 smoke

이 그룹은 repo의 제품 성격에 가까운 실행 검증이다. 라이브러리 repo는 단위 테스트가
중심일 수 있지만, runtime 도구나 배포 도구는 실제 VM, container, registry, 원격 환경에서
기본 동작을 확인해야 한다. smoke test는 깊은 검증보다 “사용자가 바로 만나는 기본 경로가
살아 있는가”를 빠르게 보는 용도다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G163 | VM runtime test | VM runtime workflow/target | VM에서 runtime 기본 동작 |
| G164 | VM runtime integration | VM integration workflow/target | VM에서 통합 동작 |
| G165 | VM artifact upload | VM logs artifact | 원격 테스트 로그 보존 |
| G166 | PID 1 container smoke | `smoke-pid1-container` | 컨테이너 PID 1 실행 확인 |
| G167 | registry smoke | registry smoke workflow/script | registry push/pull/sync 확인 |
| G168 | CLI smoke | CLI smoke target/script | CLI 기본 실행 확인 |
| G169 | remote smoke | remote smoke script/workflow | 원격 환경 smoke |
| G170 | cross-repo runtime contract | sibling repo tests in one target | 여러 repo 계약 확인 |
| G171 | kube-slint SLI gate | `make slint`, slint workflow | SLI 기반 운영 회귀 gate |
| G172 | SLI artifact upload | SLI artifact | SLI 결과 보존 |

### 2.14 품질 정책 / 릴리스 / 성능

이 그룹은 repo 고유의 품질 정책과 릴리스 안정성을 본다. 일반 린터로 표현하기 어려운
프로젝트 규칙은 별도 quality script로 둔다. 릴리스 workflow는 “tag를 찍었을 때 실제
사용 가능한 산출물이 만들어지는가”를 보장한다. benchmark는 성능 회귀를 기록하거나
차단하는 데 사용한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G173 | repo quality script | quality guardrail script | repo 고유 품질 정책 |
| G174 | license check | license guardrail | 라이선스 정책 |
| G175 | dependency policy | depguard/license/dependency script | 의존성 정책 |
| G176 | release workflow | release.yml | 릴리스 자동화 |
| G177 | multi-platform release build | GOOS/GOARCH matrix/loop | 여러 플랫폼 바이너리 |
| G178 | benchmark run | `go test -bench` | benchmark 실행 |
| G179 | benchmark publish | benchmark action/Pages | benchmark 결과 게시 |
| G180 | benchmark regression compare | `bench-compare` | baseline 대비 성능 비교 |

### 2.15 GitHub Actions 세부 설정

이 그룹은 workflow의 운영 품질을 본다. 같은 테스트라도 timeout이 없으면 멈춘 job이 오래
남고, permissions가 넓으면 불필요한 권한을 가진다. matrix, needs, cache, Go version
선택도 CI의 안정성과 재현성에 영향을 준다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G181 | Ubuntu runner | `runs-on: ubuntu-*` | Linux CI runner 사용 |
| G182 | job timeout | `timeout-minutes:` | 멈춘 job을 자동 종료 |
| G183 | Go version file 사용 | `go-version-file: go.mod` | go.mod 기준 Go 버전 사용 |
| G184 | 고정 Go version 사용 | `go-version: ...` | workflow에 Go 버전 직접 지정 |
| G185 | Go cache 사용 | `cache: true` 또는 setup-go 기본 cache | Go module/build cache 사용 |
| G186 | minimal permissions | `permissions:` 명시 | workflow 권한을 명시적으로 제한 |
| G187 | contents read permission | `contents: read` | checkout/analysis용 최소 repo 읽기 권한 |
| G188 | security-events write permission | `security-events: write` | CodeQL 결과 업로드 권한 |
| G189 | pages write permission | `pages: write` 또는 gh-pages token | Pages 게시 권한 |
| G190 | pull request scoped permissions | `pull-requests:` 등 | PR 관련 권한 명시 |
| G191 | job dependency | `needs:` | job 간 순서/의존성 명시 |
| G192 | matrix strategy | `strategy.matrix` | 여러 언어/버전/환경 matrix |
| G193 | manual workflow inputs | `workflow_dispatch.inputs` | 수동 실행 입력값 제공 |
| G194 | scheduled runtime validation | schedule로 runtime/smoke 실행 | 정기 runtime drift 검사 |

### 2.16 artifact 세부 항목

artifact는 실패를 나중에 해석하기 위한 증거다. CI가 실패했는데 로그나 report가 남지
않으면 원인 파악이 느려진다. coverage, vulnerability, smoke, SLI, benchmark 결과는
가능하면 사람이 읽는 요약과 machine-readable 원본을 함께 남기는 것이 좋다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G195 | coverage raw profile artifact | `coverage.out`, `cover.out` 업로드 | coverage 원본 profile 보존 |
| G196 | coverage text report artifact | `coverage.txt` 업로드 | 사람이 읽는 coverage 결과 보존 |
| G197 | coverage HTML artifact | `index.html` 등 업로드 | HTML coverage 보존 |
| G198 | vuln JSON artifact | `govulncheck.json` 업로드 | 취약점 raw JSON 보존 |
| G199 | security summary artifact | `lint-security-summary`, `govulncheck*.summary` | 관찰 결과 요약 보존 |
| G200 | smoke log artifact | smoke log directory 업로드 | smoke 실패 분석 로그 보존 |
| G201 | VM log artifact | VM runtime log 업로드 | 원격 VM 실패 분석 로그 보존 |
| G202 | SLI result artifact | SLI JSON/report 업로드 | 운영 지표 결과 보존 |
| G203 | benchmark output artifact/data | `bench-output.txt`, benchmark data | benchmark 결과 보존 |

### 2.17 golangci-lint 실행 세부 설정

이 그룹은 golangci-lint의 실행 형태를 더 세밀하게 본다. config v2, timeout, generated
exclusion, test file relax는 린트 결과의 신뢰도와 개발자 경험을 좌우한다. 테스트 파일은
일부 완화가 필요할 수 있지만, 완화 범위가 넓으면 실제 문제를 숨길 수 있다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G204 | golangci config v2 | `version: "2"` | golangci-lint v2 config 형식 |
| G205 | default linters disabled | `linters.default: none` | 켜는 린터만 명시 |
| G206 | allow parallel runners | `allow-parallel-runners: true` | 병렬 lint runner 허용 |
| G207 | golangci timeout 5m | `timeout: 5m` | 5분 제한 |
| G208 | golangci timeout 10m | `timeout: 10m` | 10분 제한 |
| G209 | golangci no explicit timeout | timeout 미확인 | timeout이 설정되지 않았거나 기본값 |
| G210 | generated lax exclusion | `exclusions.generated: lax` | generated 파일 완화 |
| G211 | vendor/third_party exclusions | `vendor`, `third_party` paths | 외부 vendored code 제외 |
| G212 | examples/builtin exclusions | `examples`, `builtin` paths | 예제/내장 샘플 제외 |
| G213 | protobuf file lint relax | `*.pb.go`, `*_grpc.pb.go` 예외 | generated protobuf lint 완화 |
| G214 | test file security relax | test file에서 `gosec` 제외 | 테스트 hardcoded 값 등 허용 |
| G215 | test file complexity relax | test file에서 complexity/dupl/lll 완화 | 테스트 가독성/중복 허용 |
| G216 | depguard test relax | test file에서 depguard 완화 | 테스트 helper import 허용 |
| G217 | local-prefix goimports | `local-prefixes` 명시 | import grouping 기준 |

### 2.18 gosec 세부 예외

보안 린트 예외는 특히 조심해야 한다. 예외가 있다는 사실 자체가 나쁜 것은 아니지만,
왜 필요한지 설명 가능해야 한다. 예외는 repo 전체가 아니라 가능한 좁은 path나 rule에만
적용하는 것이 좋다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G218 | gosec secret-rule custom config | gosec secret rule config | secret 탐지 패턴/entropy 조정 |
| G219 | gosec decompression rule 제외 | gosec decompression rule exclude | decompression bomb 류 경고 제외 |
| G220 | gosec integer-conversion rule 제외 | gosec integer conversion rule exclude | integer conversion 경고 제외 |
| G221 | gosec variable-command rule 제외 | gosec variable command rule exclude | variable command execution 경고 제외 |
| G222 | gosec file-path rule 제외 | gosec file path rule exclude | file path taint 경고 제외 |
| G223 | gosec file-permission rule 제외 | gosec file permission rule exclude | file permission 경고 제외 |
| G224 | gosec path-specific relax | 특정 파일/path에서 gosec 제외 | repo 의도상 필요한 파일만 완화 |
| G225 | gosec no explicit exclusions | gosec 예외 없음 | 보안 린트 예외가 명시되지 않음 |

### 2.19 depguard 세부 정책

depguard는 import 구조를 정책으로 강제한다. 이 항목은 architecture boundary를 지키는 데
유용하다. 예를 들어 core library가 Kubernetes runtime package를 직접 import하기 시작하면
나중에 재사용성과 테스트성이 떨어질 수 있다. depguard는 이런 경계 침식을 초기에 막는다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G226 | Kubernetes import deny | `k8s.io/**` deny | K8s 직접 의존성 차단 |
| G227 | controller-runtime import deny | `sigs.k8s.io/**` deny | controller-runtime 계열 차단 |
| G228 | backend-only K8s boundary | only backend may import K8s | K8s 의존성을 backend로 제한 |
| G229 | pure OCI/library boundary | pure OCI/library 설명 | 라이브러리 순수성 유지 |
| G230 | product core K8s independence | product core K8s-independent | core와 K8s 분리 |
| G231 | runtime tool K8s independence | runtime tool K8s-independent | container runtime tool과 K8s 분리 |
| G232 | pkg K8s independence | `pkg/` no k8s | library package와 K8s adapter 분리 |
| G233 | core service boundary | core no service import | core가 service layer를 import하지 않음 |
| G234 | core transport boundary | core/cmd no transport import | core/cmd가 transport adapter를 import하지 않음 |
| G235 | allowlist purity boundary | depguard allowlist | 허용 import만 명시 |

### 2.20 테스트/coverage 세부 설정

이 그룹은 테스트를 더 신뢰할 수 있게 만드는 세부 옵션이다. race, shuffle, no-cache,
반복 실행은 flaky test와 동시성 문제를 찾는 데 도움이 된다. coverage threshold는 테스트
양의 하한선을 만들지만, threshold만으로 테스트 품질을 판단하면 안 된다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G236 | covermode atomic | `-covermode=atomic` | race와 함께 안전한 coverage mode |
| G237 | coverpkg explicit | `-coverpkg` | coverage 대상 package를 명시 |
| G238 | coverage package exclude | grep -v, explicit package list | coverage에서 generated/bench 등 제외 |
| G239 | coverage merge | coverage profile merge | 여러 테스트 profile 병합 |
| G240 | coverage threshold 70 | `COVERAGE_THRESHOLD=70` 또는 70% gate | 70% 기준 |
| G241 | configurable coverage threshold | `COVERAGE_THRESHOLD ?=` | 환경/Make 변수로 기준 조정 가능 |
| G242 | coverage threshold awk gate | `awk` 비교 gate | shell에서 수치 비교 |
| G243 | race on all packages | `go test -race ./...` | 전체 package race |
| G244 | race on core packages | `go test -race $(PKGS_CORE)` | 일부 핵심 package race |
| G245 | lifecycle repeated race | lifecycle `-race -count=N` | 반복 lifecycle race |
| G246 | test cache disabled | `-count=1` | 캐시 영향 제거 |
| G247 | shuffle enabled | `-shuffle=on` | 순서 의존성 감지 |
| G248 | integration build tags | `integration` tag | 통합 테스트 scope |
| G249 | runtime build tags | `runtime` tag | runtime 테스트 scope |
| G250 | slint build tags | `slint` tag | kube-slint 테스트 scope |
| G251 | short test mode | `-short` | 느린 테스트 제외 |
| G252 | regression target | `test-regression`, regression script | 회귀 테스트 명시 target |
| G253 | golden update target | `UPDATE_GOLDEN=1`, fixture update | golden/fixture 갱신 경로 |
| G254 | smoke target separate from test | `make smoke*` | 기본 test와 smoke 분리 |

### 2.21 protobuf/Kubernetes/runtime 세부 항목

이 그룹은 앞의 protobuf, Kubernetes, runtime 항목을 실제 구현 단위로 더 나눈 것이다.
설치 방식, checksum 검증, generated drift, kind artifact 보존처럼 작은 차이가 CI의
재현성과 디버깅 속도에 영향을 준다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G255 | Buf action | `bufbuild/buf-action@v1` | Buf를 Action으로 실행 |
| G256 | Buf setup action | `bufbuild/buf-setup-action@v1` | Buf CLI setup |
| G257 | protoc-gen-go install | `go install .../protoc-gen-go` | Go protobuf generator 설치 |
| G258 | buf breaking against main | `buf breaking --against '.git#branch=main'` | main 기준 schema 호환성 |
| G259 | buf breaking fallback | `--against '.git#ref=HEAD~1'` | main 비교 실패 시 fallback |
| G260 | generate then diff | generate 후 `git diff` | generated drift 검사 |
| G261 | kube-linter checksum install | kube-linter checksum 검증 설치 | kube-linter binary 검증 |
| G262 | kube-linter action | `stackrox/kube-linter-action@v1` | kube-linter Action |
| G263 | kube-linter CLI install | `make kube-linter`, `go install` | kube-linter CLI 준비 |
| G264 | kubeconform latest tar install | kubeconform tar download | kubeconform 설치 |
| G265 | kind binary install | kind download/chmod/move | kind CLI 설치 |
| G266 | kind artifact upload | `artifacts/kind*` upload | kind smoke 결과 보존 |
| G267 | SSH-based VM workflow | SSH key/agent/known_hosts | SSH 기반 VM 검증 |
| G268 | podman/unshare runtime test | `podman unshare` | rootless/runtime 환경 테스트 |
| G269 | container smoke script | container smoke script | 컨테이너 실행 스모크 |
| G270 | registry sync smoke script | registry sync smoke script | registry 동기화 스모크 |
| G271 | remote publish/preflight scripts | `preflight`, `publish`, remote scripts | 원격 publish/smoke 준비 |

### 2.22 릴리스/성능 세부 항목

릴리스와 성능 항목은 평소 PR 테스트와 성격이 다르다. 릴리스는 사용자가 받는 바이너리와
아카이브가 제대로 만들어지는지 확인하고, 성능 항목은 시간이 지나며 느려지는 변화를
기록한다. 성능 결과는 한 번의 숫자보다 추세가 중요하다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G272 | release test rerun | release workflow에서 test 실행 | 릴리스 전에 테스트 재확인 |
| G273 | trimpath build | `go build -trimpath` | 재현성/경로 제거 build |
| G274 | CGO disabled release build | `CGO_ENABLED=0` | static-friendly release build |
| G275 | GOOS/GOARCH loop | OS/ARCH 반복 build | 다중 플랫폼 산출물 |
| G276 | benchmark-action | `benchmark-action/github-action-benchmark` | benchmark 전용 Action |
| G277 | benchmark benchtime fixed | `-benchtime=3s` | benchmark 시간 고정 |
| G278 | benchmark memory metrics | `-benchmem` | allocation/memory 지표 |
| G279 | benchmark baseline compare script | `scripts/bench_compare.sh` | baseline 대비 비교 |
| G280 | performance history file | `PERFORMANCE_HISTORY.md` | 성능 기준 기록 파일 |

### 2.23 kube-slint / slint-gate 가드레일 세부 항목

이 섹션은 `kube-slint`를 repo로 비교하기 위한 항목이 아니다. 다른 repo가
`kube-slint` 또는 `slint-gate`를 품질 가드레일로 사용하는지를 판단하기 위한
항목이다.

`kube-slint` 계열 항목은 일반 unit test와 다르게 운영 지표를 shift-left로 당겨오는
가드레일이다. 즉 “코드가 맞는가”보다는 “운영 관점에서 느려지거나 실패율이 올라가지
않는가”를 테스트/CI 단계에서 확인한다. 이 그룹은 SLI를 어떻게 측정하는지, 정책으로
평가하는지, artifact를 남기는지, coverage gap을 감시하는지를 나눠 본다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G281 | kube-slint SLI 측정 harness 사용 | `pkg/slint`, `test/slint`, `sli-summary.json` 생성 | 운영 SLI를 테스트/CI 단계에서 측정하는지 |
| G282 | kube-slint custom SLI spec 사용 | repo-specific `SLISpec`, `ComputeSpec`, `BaselineV3Specs` 등 | repo 고유 SLI를 정의했는지 |
| G283 | kube-slint SnapshotFetcher 사용 | `SnapshotFetcher`, `MetricsFetcher`, `fetch.SnapshotFetcher` 구현 | point/snapshot metric source를 붙였는지 |
| G284 | kube-slint WindowFetcher 사용 | `WindowFetcher`, range/window fetcher 구현 | range/window SLI source를 붙였는지 |
| G285 | kube-slint Prometheus/curlpod source 사용 | curlpod, Prometheus point scrape | cluster 내부 metric source를 사용했는지 |
| G286 | kube-slint JSON/expvar source 사용 | `jsonendpoint`, expvar/JSON fetcher | Prometheus가 아닌 HTTP JSON source를 사용했는지 |
| G287 | kube-slint promrange source 사용 | `promrange`, Prometheus `query_range` | Prometheus range query를 사용했는지 |
| G288 | kube-slint policy file 사용 | `.slint/policy.yaml`, `config/slint/policy.yaml` | SLI 결과를 정책으로 판정하는지 |
| G289 | slint-gate CLI 사용 | `go run ./cmd/slint-gate`, `slint-gate` binary | SLI summary를 gate로 평가하는지 |
| G290 | slint-gate GitHub composite action 사용 | `.github/actions/slint-gate` 또는 remote action | GitHub Actions에서 slint-gate action을 쓰는지 |
| G291 | slint-gate workflow gate | `.github/workflows/slint-gate.yml` 또는 repo-specific gate workflow | PR/push/manual에서 slint-gate를 실행하는지 |
| G292 | slint-gate baseline 비교 | `--baseline`, baseline summary | 이전 baseline 대비 회귀를 보는지 |
| G293 | slint-gate threshold 정책 | policy thresholds | 절대 임계값을 보는지 |
| G294 | slint-gate coverage governance | `coverage.required`, `coverage_gap` | 측정된 SLI가 정책에 빠졌는지 감시하는지 |
| G295 | slint-gate strict exit behavior | `--exit-on`, `exit-on: FAIL_OR_NOGRADE` | FAIL/NO_GRADE 등을 CI 실패로 승격하는지 |
| G296 | slint-gate summary artifact | `slint-gate-summary.json` artifact | gate 결과를 artifact로 남기는지 |
| G297 | kube-slint SLI artifact | `sli-summary.json`, `sli-summary*.json` artifact | 원본 SLI 측정 결과를 artifact로 남기는지 |
| G298 | kube-slint kind demo/consumer validation | kind demo, hello-operator consumer demo | 실제 consumer 흐름에서 측정→gate가 되는지 |
| G299 | kube-slint quality guardrail script | `hack/quality-guardrails.sh` | kube-slint 제품 계약/문서/보안 drift를 막는지 |
| G300 | kube-slint roadmap/status guardrail | `docs/project-status.yaml`, roadmap-status workflow | machine-readable 상태 문서를 검증/렌더링하는지 |
| G301 | kube-slint custom Semgrep guardrails | `.semgrep/rules` for kube-slint patterns | kube-slint 보안/안정성 회귀 패턴을 Semgrep로 막는지 |
| G302 | kube-slint generated CI snippet guardrail | `ci github-actions` generator tests | consumer용 CI snippet이 깨지지 않는지 |
| G303 | kube-slint quickstart/wizard guardrail | `quickstart`, `wizard`, onboarding tests | 온보딩 UX가 계속 동작하는지 |
| G304 | kube-slint dataplane analyzer guardrail | `analyze-dataplane`, `pkg/dataplane`, `pkg/report` | Kubernetes dataplane manifest 위험을 분석하는지 |
| G305 | kube-slint summary schema bad-fixture tests | bad fixture tests under `pkg/gate/testdata` | 잘못된 summary/policy가 PASS로 오인되지 않는지 |
| G326 | kube-slint static JSON fixture source | JSON fixture + custom/static `fetch.SnapshotFetcher` | live scrape가 아니라 저장된 JSON fixture를 SLI 입력으로 쓰는지 |
| G327 | kube-slint in-test assertion gate | `go test` 안에서 SLI value를 직접 assert | `slint-gate` CLI 없이 테스트 코드가 SLI threshold를 직접 판정하는지 |

#### 2.23.1 repo별 관측 SLI 세부 목록

이 표는 `G281-G327`의 하위 근거다. 여기의 SLI ID는 repo별 운영 도메인에 묶인
수치 지표이므로 전역 G-ID로 승격하지 않는다. 대신 “어떤 repo가 kube-slint를 쓴다”를
넘어서, 실제로 무엇을 보고 있는지 확인하기 위한 하위 항목으로 남긴다.

해석 기준:

- `측정 SLI`: kube-slint summary에 들어가는 개별 SLI result ID다.
- `정책 SLI`: slint-gate policy threshold가 직접 참조하는 SLI다.
- `직접 테스트 SLI`: kube-slint summary 밖에서 Go test가 직접 측정/assert하는 운영 지표다.
- 같은 대표 ID라도 repo마다 SLI 의미가 다르다. 그래서 SLI 이름 자체를 전역 번호로 만들지
  않고, repo별 하위 목록으로 관리한다.

| repo | SLI 세부 항목 | 성격 | 확인 소스 | 쉬운 설명 |
|---|---|---|---|---|
| `kube-slint` | `reconcile_total_delta` | 기본 preset 측정 SLI / 정책 예시 | `pkg/slint/presets.go`, `.slint/policy.yaml` | controller-runtime reconcile 총량이 window 안에서 얼마나 움직였는지 |
| `kube-slint` | `reconcile_success_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | 성공 reconcile delta |
| `kube-slint` | `reconcile_error_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | error reconcile delta |
| `kube-slint` | `workqueue_adds_total_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | workqueue add delta |
| `kube-slint` | `workqueue_retries_total_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | workqueue retry delta |
| `kube-slint` | `workqueue_depth_end` | 기본 preset 측정 SLI / 정책 예시 | `pkg/slint/presets.go`, `.slint/policy.yaml` | window 종료 시점 queue depth |
| `kube-slint` | `rest_client_requests_total_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | client-go REST request 총량 delta |
| `kube-slint` | `rest_client_429_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | API server throttling 신호 |
| `kube-slint` | `rest_client_5xx_delta` | 기본 preset 측정 SLI | `pkg/slint/presets.go` | API server/client-go 5xx 신호 |
| `NodeVault` | `reconcile_fast_delta` | 측정 SLI + Go test 직접 assert + 정책 SLI | `test/slint/nodevault_slint_test.go`, `.slint/policy.yaml` | 빠른 reconcile loop가 살아 있고 과도하게 돌지 않는지 |
| `NodeVault` | `reconcile_slow_delta` | 측정 SLI + Go test 직접 assert | `test/slint/nodevault_slint_test.go` | 테스트 window에서 느린 reconcile loop가 불필요하게 돌지 않는지 |
| `NodeVault` | `reconcile_error_delta` | 측정 SLI + Go test 직접 assert + 정책 SLI | `test/slint/nodevault_slint_test.go`, `.slint/policy.yaml` | reconcile error가 발생하지 않는지 |
| `NodeVault` | `build_failure_delta` | 측정 SLI + Go test 직접 assert + 정책 SLI | `test/slint/nodevault_slint_test.go`, `.slint/policy.yaml` | build disabled window에서 build failure counter가 움직이지 않는지 |
| `NodeVault` | startup latency | 직접 테스트 SLI | `test/slint/nodevault_slint_test.go` | 프로세스 시작부터 `/healthz` ready까지 5초 budget 안에 들어오는지 |
| `NodeVault` | gRPC Ping latency/error | 직접 테스트 SLI | `test/slint/nodevault_slint_test.go` | 관측 window 동안 gRPC Ping이 실패하지 않고 200ms 안에 끝나는지 |
| `JUMI` | `jumi_jobs_created_delta`, `jumi_jobs_created_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | JUMI가 smoke window에서 Kubernetes Job을 만들었는지 |
| `JUMI` | `jumi_artifacts_registered_delta`, `jumi_artifacts_registered_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | JUMI output artifact가 AH에 등록됐는지 |
| `JUMI` | `jumi_input_resolve_requests_delta`, `jumi_input_resolve_requests_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | consumer node 시작 전에 input artifact resolve가 일어났는지 |
| `JUMI` | `jumi_input_remote_fetch_delta`, `jumi_input_remote_fetch_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | remote-fetch resolution decision이 관측됐는지 |
| `JUMI` | `jumi_input_materializations_delta`, `jumi_input_materializations_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | resolved input materialization이 발생했는지 |
| `JUMI` | `jumi_sample_runs_finalized_delta`, `jumi_sample_runs_finalized_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | sample run finalize가 AH를 통해 완료됐는지 |
| `JUMI` | `jumi_gc_evaluate_requests_delta`, `jumi_gc_evaluate_requests_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | GC 평가 요청이 발생했는지 |
| `JUMI` | `jumi_cleanup_backlog_objects_end` | 측정 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go` | window 종료 시점 JUMI cleanup backlog 수 |
| `JUMI` | `ah_artifacts_registered_delta`, `ah_artifacts_registered_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | AH inventory가 artifact를 받았는지 |
| `JUMI` | `ah_resolve_requests_delta`, `ah_resolve_requests_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | AH resolve 요청이 들어왔는지 |
| `JUMI` | `ah_fallback_delta`, `ah_fallback_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | AH fallback transition 관측값 |
| `JUMI` | `ah_gc_backlog_bytes_end`, `ah_gc_backlog_bytes_smoke` | 측정 SLI + 정책 SLI | `tools/kubeslint-smoke-summary/spec_profiles.go`, `policy/devspace/*.yaml` | AH GC backlog bytes가 smoke/live 기준 이하인지 |
| `JUMI` | `k8s_namespace_jobs_total_delta_churn`, `k8s_namespace_pods_total_delta_churn` | derived churn SLI | `tools/kubeslint-smoke-summary/main.go` | namespace 전체 Job/Pod object churn delta |
| `JUMI` | `k8s_jobs_for_run_churn`, `k8s_pods_for_run_churn` | derived churn SLI | `tools/kubeslint-smoke-summary/main.go` | smoke run label이 붙은 Job/Pod가 남아 있는지 |
| `JUMI` | `k8s_failed_jobs_end_churn`, `k8s_active_jobs_end_churn` | derived churn SLI | `tools/kubeslint-smoke-summary/main.go` | 종료 시점 failed/active Job 잔존 여부 |
| `JUMI` | `ah_lifecycle_finalized_smoke`, `ah_retention_window_active_smoke` | derived lifecycle SLI | `tools/kubeslint-smoke-summary/main.go` | AH sample-run lifecycle이 finalize되고 retention window에 들어갔는지 |
| `JUMI` | `ah_retained_artifact_bytes_smoke`, `ah_artifact_metadata_complete_smoke`, `ah_inventory_lifecycle_bytes_match_smoke` | derived inventory/provenance SLI | `tools/kubeslint-smoke-summary/main.go` | AH inventory metadata와 lifecycle retained bytes가 서로 맞는지 |
| `bori` | `reconcile_total_delta`, `reconcile_success_delta`, `reconcile_error_delta` | 기본 preset 측정 SLI | `test/e2e/helpers_test.go`, kube-slint `DefaultSpecs()` | bori e2e smoke에서 controller-runtime reconcile 흐름을 기본 preset으로 측정 |
| `bori` | `workqueue_adds_total_delta`, `workqueue_retries_total_delta`, `workqueue_depth_end` | 기본 preset 측정 SLI | `test/e2e/helpers_test.go`, kube-slint `DefaultSpecs()` | bori workqueue churn/depth를 기본 preset으로 측정 |
| `bori` | `rest_client_requests_total_delta`, `rest_client_429_delta`, `rest_client_5xx_delta` | 기본 preset 측정 SLI | `test/e2e/helpers_test.go`, kube-slint `DefaultSpecs()` | bori controller의 Kubernetes API request와 throttling/5xx 신호 |
| `bori` | `bori_reconcile_delta` | bori 전용 SLI | `test/e2e/.slint/policy.yaml` | `BoriDataPlane` reconcile delta |
| `bori` | `bori_reconcile_errors_delta` | bori 전용 SLI | `test/e2e/.slint/policy.yaml` | `BoriDataPlane` reconcile error delta |
| `bori` | `bori_workqueue_depth_end` | bori 전용 SLI | `test/e2e/.slint/policy.yaml` | `BoriDataPlane` queue depth 종료값 |
| `bori` | `bori_rest_errors_delta` | bori 전용 SLI | `test/e2e/.slint/policy.yaml` | 5xx REST client request delta |

#### 2.23.2 repo별 SLI 검토 후보

이 표는 현재 적용 근거가 확인된 항목이 아니라, repo 성격상 앞으로 넣어볼 만한 SLI 후보를
분리해 적은 것이다. 사용자가 검토하기 쉽도록 `제안`으로 표시한다. 따라서 이 표의 항목은
repo별 적용 ID 목록에 넣지 않는다.

선정 기준은 단순하다. 각 repo가 실제로 운영 중 문제가 될 수 있는 “느려짐, 실패율 증가,
backlog 증가, orphan 누적, 외부 의존성 실패, 보안/검증 누락”을 수치로 볼 수 있는지를
기준으로 잡았다.

| repo | 제안 SLI 후보 | 상태 | 왜 보면 좋은가 |
|---|---|---|---|
| `NodeVault` | `validation_scan_records_delta` | 제안 | scan record ingestion이 smoke window에서 실제로 들어오는지 확인 |
| `NodeVault` | `catalog_index_write_errors_delta` | 제안 | catalog/index 저장 실패가 생기면 검증 결과가 쌓이지 않으므로 조기 감지 필요 |
| `NodeVault` | `tool_scan_critical_findings_end` | 제안 | Trivy/scan 결과 중 critical finding 수를 promotion gate 후보로 볼 수 있음 |
| `NodeVault` | `webhook_rejects_delta` | 제안 | admission/webhook 정책이 의도대로 거부를 발생시키는지 관찰 |
| `NodeVault` | `grpc_request_latency_p95_ms` | 제안 | 현재 Ping max latency는 test 내부 assert라 summary에 남지 않음. p95로 남기면 추세 비교 가능 |
| `podbridge5` | `image_build_duration_ms` | 제안 | 이미지 build runtime 회귀를 가장 직접적으로 보여줌 |
| `podbridge5` | `image_save_archive_bytes` | 제안 | 저장된 tar/tar.gz artifact 크기 급증을 감지 |
| `podbridge5` | `buildah_failures_delta` | 제안 | user namespace/capability/storage driver 문제를 count로 관찰 |
| `podbridge5` | `runtime_capability_skip_delta` | 제안 | 환경 capability 부족으로 skip되는 테스트가 늘어나는지 확인 |
| `JUMI` | `jumi_run_duration_p95_ms` | 제안 | Job 생성 수뿐 아니라 end-to-end run latency 회귀를 볼 수 있음 |
| `JUMI` | `jumi_node_retry_delta` | 제안 | DAG node retry가 늘면 사용자는 성공해도 불안정성을 겪음 |
| `JUMI` | `jumi_fast_fail_to_terminal_ms` | 제안 | fast-fail이 실제로 빠르게 terminal state로 수렴하는지 확인 |
| `JUMI` | `jumi_gc_deleted_artifacts_delta` | 제안 | GC evaluate 요청뿐 아니라 실제 삭제/정리 효과를 확인 |
| `JUMI` | `jumi_orphan_jobs_end` | 제안 | smoke 후 Job orphan이 남는지 직접 확인 |
| `bori` | `bori_revision_promotion_latency_ms` | 제안 | BoriRevision promotion이 느려지는지 관찰 |
| `bori` | `bori_verified_revisions_delta` | 제안 | slint-gate 결과가 Verified 상태로 이어지는지 확인 |
| `bori` | `bori_failed_promotions_delta` | 제안 | promotion 실패율 증가를 operator 관점에서 측정 |
| `bori` | `bori_condition_transition_latency_ms` | 제안 | CR condition이 기대 상태로 바뀌는 데 걸리는 시간 |
| `bori` | `bori_digest_mismatch_rejects_delta` | 제안 | digest 기반 release identity가 실제로 잘못된 promotion을 막는지 확인 |
| `sori` | `sori_publish_duration_ms` | 제안 | dataset publish/packaging latency 회귀 감지 |
| `sori` | `sori_manifest_validation_failures_delta` | 제안 | dataset metadata/manifest 계약 위반 증가 감지 |
| `sori` | `sori_chunk_upload_retries_delta` | 제안 | chunked publish retry 증가를 통해 storage/network 불안정성 확인 |
| `sori` | `sori_remote_fetch_bytes_delta` | 제안 | 원격 fetch 경로가 예상보다 많은 데이터를 당기는지 확인 |
| `NodeSentinel` | `nodesentinel_validation_runs_delta` | 제안 | validation worker가 smoke window에서 실제로 실행됐는지 확인 |
| `NodeSentinel` | `nodesentinel_l5b_not_available_delta` | 제안 | trivy-operator/VulnerabilityReport 부재가 얼마나 자주 fallback되는지 확인 |
| `NodeSentinel` | `nodesentinel_scan_record_submit_errors_delta` | 제안 | NodeVault scan record submit 실패를 조기 감지 |
| `NodeSentinel` | `nodesentinel_worker_poll_latency_ms` | 제안 | worker polling이 느려지면 검증 결과 지연으로 이어짐 |
| `NodePalette` | `nodepalette_render_duration_ms` | 제안 | UI/asset/render 계열이면 render latency 회귀가 체감 품질에 직접 영향 |
| `NodePalette` | `nodepalette_config_load_errors_delta` | 제안 | 설정 로딩 실패를 smoke에서 빠르게 확인 |
| `artifact-handoff` | `ah_artifacts_registered_delta` | 제안 | artifact inventory 등록량은 handoff 핵심 liveness |
| `artifact-handoff` | `ah_resolve_requests_delta` | 제안 | consumer resolve 요청이 실제로 처리되는지 확인 |
| `artifact-handoff` | `ah_resolve_failures_delta` | 제안 | allowlist/digest/state 문제로 resolve가 실패하는지 확인 |
| `artifact-handoff` | `ah_gc_blocked_delta` | 제안 | GC가 retention/lease/policy 때문에 막히는 빈도 확인 |
| `artifact-handoff` | `ah_orphan_artifacts_end` | 제안 | smoke 후 orphan artifact 누적 여부 확인 |
| `node-artifact-runtime` | `nan_materialization_duration_ms` | 제안 | runtime helper가 artifact를 준비하는 시간이 늘어나는지 확인 |
| `node-artifact-runtime` | `nan_remote_fetch_failures_delta` | 제안 | remote fetch 실패가 command 실패로 번지기 전 감지 |
| `node-artifact-runtime` | `nan_supervisor_timeout_delta` | 제안 | PID1/supervisor timeout 증가 확인 |
| `node-artifact-runtime` | `nan_process_group_leak_end` | 제안 | 종료 후 process group 잔류 여부를 수치화 |
| `spawner` | `spawner_runs_started_delta` | 제안 | dispatcher/runtime이 실제 run을 시작했는지 확인 |
| `spawner` | `spawner_run_failures_delta` | 제안 | run failure 증가를 smoke에서 감지 |
| `spawner` | `spawner_attempt_retries_delta` | 제안 | recovery/retry churn이 늘어나는지 확인 |
| `spawner` | `spawner_queue_depth_end` | 제안 | dispatch backlog가 남는지 확인 |
| `tori` | `tori_sync_duration_ms` | 제안 | NAS/shared fixture 동기화 latency 회귀 감지 |
| `tori` | `tori_catalog_write_failures_delta` | 제안 | catalog write 실패는 데이터 신뢰성에 직접 영향 |
| `tori` | `tori_partial_write_recovery_delta` | 제안 | partial write/corruption recovery가 발생하는지 확인 |
| `tori` | `tori_db_lock_wait_ms` | 제안 | DB/file lock 대기로 느려지는 현상 감지 |
| `dag-go` | `dag_run_duration_ms` | 제안 | DAG 전체 실행 latency 회귀 감지 |
| `dag-go` | `dag_node_failures_delta` | 제안 | node 실패 증가 확인 |
| `dag-go` | `dag_retry_attempts_delta` | 제안 | retry policy가 과도하게 발동하는지 확인 |
| `dag-go` | `dag_goroutines_end` | 제안 | run 종료 후 goroutine leak 여부 확인 |
| `dag-go` | `dag_deadlock_timeout_delta` | 제안 | deadlock/timeout 성격의 실패를 별도 관찰 |

#### 2.23.3 Kubernetes/operator churn SLI 후보

이 표는 특히 Kubernetes Operator에서 중요한 churn 관점의 SLI 후보를 따로 모은 것이다.
churn은 “작동은 하지만 너무 자주 흔들리는 상태”를 뜻한다. 예를 들어 reconcile이 계속
반복되거나, workqueue retry가 늘거나, Job/Pod가 불필요하게 생성/삭제되거나, finalizer
때문에 terminating 리소스가 쌓이는 경우다.

이 항목들도 전역 G-ID로 만들지 않는다. 대신 `G421-G426` 같은 operator guardrail 번호를
구현할 때 어떤 SLI를 쓸 수 있는지 보여주는 하위 후보 목록으로 둔다.

번호 읽는 법:

- `KOC`: Kubernetes Operator Churn
- 예: `KOC-008 workqueue_depth_end`는 “operator queue backlog를 보는 churn SLI 후보”라는 뜻이다.
- `G421` 같은 대표 가드레일을 실제로 구현할 때 `KOC-*` 중 어떤 지표를 쓸지 고르면 된다.

| 하위 ID | churn SLI 후보 | 주 적용 repo 유형 | 연결되는 대표 G-ID | 쉬운 의미 |
|---|---|---|---|---|
| KOC-001 | `reconcile_total_delta` | 모든 controller-runtime operator | G421, G423 | window 안 reconcile 총량이 갑자기 늘면 불필요한 재처리나 watch 폭주 가능성 |
| KOC-002 | `reconcile_success_delta` | 모든 controller-runtime operator | G421 | 성공 reconcile이 과도하게 많으면 상태가 안정되지 않고 계속 다시 도는 신호일 수 있음 |
| KOC-003 | `reconcile_error_delta` | 모든 controller-runtime operator | G409, G420, G421 | error reconcile 증가는 retry storm과 연결될 수 있음 |
| KOC-004 | `reconcile_result_requeue_delta` | controller-runtime operator | G420, G421 | requeue가 과도하면 API server와 workqueue churn이 증가 |
| KOC-005 | `reconcile_latency_p95_ms` | 모든 operator | G423 | reconcile 횟수는 같아도 각 reconcile이 느려지는 회귀를 잡음 |
| KOC-006 | `workqueue_adds_total_delta` | 모든 operator | G421 | queue에 들어오는 작업량이 늘어나는지 확인 |
| KOC-007 | `workqueue_retries_total_delta` | 모든 operator | G420, G421 | 실패/충돌/외부 의존성 문제로 retry가 늘어나는지 확인 |
| KOC-008 | `workqueue_depth_end` | 모든 operator | G421 | window 종료 시 queue backlog가 남는지 확인 |
| KOC-009 | `workqueue_longest_running_processor_seconds_end` | 모든 operator | G421, G423 | 특정 reconcile worker가 오래 잡혀 병목이 되는지 확인 |
| KOC-010 | `rest_client_requests_total_delta` | Kubernetes API를 많이 쓰는 operator | G422 | API server 요청량 증가 감지 |
| KOC-011 | `rest_client_429_delta` | Kubernetes API를 많이 쓰는 operator | G422 | client-side throttling/API server pressure 감지 |
| KOC-012 | `rest_client_5xx_delta` | Kubernetes API를 많이 쓰는 operator | G409, G422 | API server 또는 네트워크 불안정성 감지 |
| KOC-013 | `child_resources_created_delta` | child Deployment/Job/Pod/Secret 생성 operator | G407, G408, G424 | reconcile 때 불필요하게 child resource를 계속 만드는지 확인 |
| KOC-014 | `child_resources_updated_delta` | child resource 관리 operator | G408, G421 | spec drift가 없어도 update가 반복되는지 확인 |
| KOC-015 | `child_resources_deleted_delta` | cleanup/finalizer 있는 operator | G403, G417, G424 | 삭제 churn이 비정상적으로 늘어나는지 확인 |
| KOC-016 | `orphan_child_resources_end` | ownerRef/finalizer 쓰는 operator | G404, G424 | ownerRef 누락이나 cleanup 실패로 orphan이 남는지 확인 |
| KOC-017 | `stuck_terminating_resources_end` | finalizer/cleanup 있는 operator | G403, G425 | 삭제 중 멈춘 CR/Pod/Job이 있는지 확인 |
| KOC-018 | `finalizer_add_delta` | finalizer 쓰는 operator | G403 | finalizer가 과도하게 추가되거나 반복 patch되는지 확인 |
| KOC-019 | `finalizer_remove_delta` | finalizer 쓰는 operator | G403, G417 | cleanup 후 finalizer 제거가 실제로 일어나는지 확인 |
| KOC-020 | `status_updates_delta` | status condition 쓰는 operator | G405, G426 | condition/status patch가 불필요하게 반복되는지 확인 |
| KOC-021 | `status_conflict_retries_delta` | status patch가 많은 operator | G402, G405 | resourceVersion conflict로 재시도가 늘어나는지 확인 |
| KOC-022 | `condition_transition_delta` | 상태 전환이 중요한 operator | G405, G426 | condition이 지나치게 자주 바뀌는지 확인 |
| KOC-023 | `condition_stale_seconds_end` | 장기 실행/비동기 operator | G405, G426 | condition이 오래 갱신되지 않는 silent failure 감지 |
| KOC-024 | `k8s_events_warning_delta` | 모든 operator | G406 | Warning event가 증가하는지 확인 |
| KOC-025 | `admission_rejects_delta` | webhook 있는 operator | G410 | 잘못된 CR이 실제로 admission에서 거부되는지 확인 |
| KOC-026 | `webhook_latency_p95_ms` | webhook 있는 operator | G410, G411, G412 | admission latency가 API server UX에 영향을 주는지 확인 |
| KOC-027 | `leader_election_changes_delta` | HA operator | G413 | leader가 자주 바뀌면 reconcile 안정성이 흔들림 |
| KOC-028 | `lease_renew_errors_delta` | HA operator | G413, G437 | leader election lease 갱신 실패 감지 |
| KOC-029 | `namespace_scope_miss_delta` | multi-namespace operator | G414, G415 | watch해야 할 namespace의 CR/child resource를 놓치는지 확인 |
| KOC-030 | `install_upgrade_apply_errors_delta` | 배포 manifest/Helm/Kustomize operator | G416, G428 | upgrade 중 apply/schema 오류 감지 |
| KOC-031 | `uninstall_leftover_resources_end` | operator lifecycle 검증 | G417, G424 | uninstall 후 남은 리소스 수 확인 |
| KOC-032 | `api_object_churn_score` | 복합 operator | G421, G424, G425 | create/update/delete/retry/orphan을 합친 단일 churn score 후보 |

#### 2.23.4 데이터 플레인 app churn SLI 후보

데이터 플레인 앱의 churn은 Kubernetes object churn과 다르다. 여기서는 사용자의 실제
작업을 처리하는 과정에서 request, job, artifact, cache, temp file, queue, retry,
worker, connection, session 같은 리소스가 불필요하게 늘거나 반복되는지를 본다.

이 항목도 “현재 적용”이 아니라 “검토 후보”다. 특히 `JUMI`, `artifact-handoff`,
`node-artifact-runtime`, `spawner`, `sori`, `tori`, `dag-go`처럼 데이터 처리/실행/전달
경로가 있는 repo에서 유용하다.

번호 읽는 법:

- `DPC`: Data Plane Churn
- 예: `DPC-022 artifact_resolve_failures_delta`는 “artifact resolve 실패 churn을 보는 SLI 후보”라는 뜻이다.
- 데이터 플레인 app에 SLI를 추가할 때 `DPC-*` 번호로 후보를 선택하고, 실제 metric/spec/policy 이름은 repo에 맞게 정하면 된다.

| 하위 ID | churn SLI 후보 | 주 적용 repo 유형 | 우선 검토 repo | 쉬운 의미 |
|---|---|---|---|---|
| DPC-001 | `requests_total_delta` | HTTP/gRPC API app | JUMI, artifact-handoff, spawner | 처리량 기준선과 비정상 요청 폭증을 같이 볼 수 있음 |
| DPC-002 | `request_errors_delta` | HTTP/gRPC API app | JUMI, artifact-handoff, NodeVault | 성공률은 유지되는 듯 보여도 error churn이 늘 수 있음 |
| DPC-003 | `request_latency_p95_ms` | HTTP/gRPC API app | JUMI, artifact-handoff, NodeVault | 요청 수보다 사용자 체감 회귀를 더 잘 보여줌 |
| DPC-004 | `jobs_started_delta` | job/pipeline 실행 app | JUMI, spawner, dag-go | 실제 작업 시작량이 기대와 맞는지 확인 |
| DPC-005 | `jobs_completed_delta` | job/pipeline 실행 app | JUMI, spawner, dag-go | 시작된 작업이 완료까지 가는지 확인 |
| DPC-006 | `jobs_failed_delta` | job/pipeline 실행 app | JUMI, spawner, dag-go | 실패 job churn 증가 감지 |
| DPC-007 | `jobs_retried_delta` | retry 있는 실행 app | JUMI, spawner, dag-go | 성공해도 내부 retry가 많아지는 불안정성 확인 |
| DPC-008 | `jobs_active_end` | long-running job app | JUMI, spawner | smoke 종료 시 active job이 남는지 확인 |
| DPC-009 | `jobs_orphan_end` | job cleanup 있는 app | JUMI, spawner | owner/run 관계가 끊긴 job 누적 확인 |
| DPC-010 | `queue_depth_end` | dispatcher/worker queue app | spawner, JUMI, artifact-handoff | 처리 뒤 backlog가 남는지 확인 |
| DPC-011 | `queue_enqueued_delta` | queue 기반 app | spawner, artifact-handoff | 입력 churn 증가 확인 |
| DPC-012 | `queue_dequeued_delta` | queue 기반 app | spawner, artifact-handoff | worker가 실제로 소비하는지 확인 |
| DPC-013 | `queue_retries_delta` | queue 기반 app | spawner, artifact-handoff | 재시도 churn 증가 확인 |
| DPC-014 | `worker_busy_ratio_end` | worker pool app | spawner, JUMI | worker 포화 여부 확인 |
| DPC-015 | `worker_restarts_delta` | supervisor/worker app | node-artifact-runtime, spawner | worker/process 재시작이 늘어나는지 확인 |
| DPC-016 | `goroutines_end` | Go service/app | dag-go, spawner, artifact-handoff | 작업 후 goroutine이 누수되는지 확인 |
| DPC-017 | `open_files_end` | file-heavy app | sori, tori, node-artifact-runtime | file descriptor 누수 감지 |
| DPC-018 | `temp_files_created_delta` | materialization/cache app | node-artifact-runtime, sori, tori | 임시 파일 생성량이 비정상적으로 늘어나는지 확인 |
| DPC-019 | `temp_files_leftover_end` | materialization/cache app | node-artifact-runtime, sori, tori | cleanup 실패로 임시 파일이 남는지 확인 |
| DPC-020 | `artifact_registered_delta` | artifact inventory app | artifact-handoff, JUMI | 산출물이 inventory에 기록되는지 확인 |
| DPC-021 | `artifact_resolve_requests_delta` | artifact handoff app | artifact-handoff, JUMI | consumer resolve 경로가 동작하는지 확인 |
| DPC-022 | `artifact_resolve_failures_delta` | artifact handoff app | artifact-handoff, JUMI | digest/policy/state 문제로 resolve 실패가 늘어나는지 확인 |
| DPC-023 | `artifact_materializations_delta` | artifact materialization app | JUMI, node-artifact-runtime | artifact가 실제 실행 환경으로 준비되는지 확인 |
| DPC-024 | `artifact_materialization_failures_delta` | artifact materialization app | node-artifact-runtime, JUMI | remote fetch/storage/path 문제 감지 |
| DPC-025 | `artifact_bytes_materialized_delta` | artifact materialization app | node-artifact-runtime, sori | 데이터 이동량 급증을 감지 |
| DPC-026 | `cache_hits_delta` | cache 있는 app | sori, tori, artifact-handoff | cache가 실제로 도움이 되는지 확인 |
| DPC-027 | `cache_misses_delta` | cache 있는 app | sori, tori, artifact-handoff | cache miss 증가로 latency/IO가 늘어나는지 확인 |
| DPC-028 | `cache_evictions_delta` | cache 있는 app | sori, tori | eviction churn 증가 확인 |
| DPC-029 | `gc_evaluate_requests_delta` | GC/retention app | artifact-handoff, JUMI | GC 판단이 실제로 실행되는지 확인 |
| DPC-030 | `gc_deleted_objects_delta` | GC/retention app | artifact-handoff, JUMI, tori | GC가 평가만 하고 삭제하지 못하는지 확인 |
| DPC-031 | `gc_blocked_objects_end` | GC/retention app | artifact-handoff, JUMI | lease/retention/policy 때문에 삭제가 막힌 객체 수 |
| DPC-032 | `retained_bytes_end` | storage/retention app | artifact-handoff, JUMI, sori | 보관 데이터가 기준 이상 커지는지 확인 |
| DPC-033 | `orphan_artifacts_end` | artifact lifecycle app | artifact-handoff, JUMI | owner/run 없이 남은 artifact 확인 |
| DPC-034 | `manifest_validation_failures_delta` | manifest/catalog app | sori, tori, artifact-handoff | schema/contract 위반 증가 감지 |
| DPC-035 | `catalog_write_failures_delta` | catalog/index app | tori, NodeVault, artifact-handoff | catalog/index 저장 실패 감지 |
| DPC-036 | `db_lock_wait_p95_ms` | SQLite/DB/file lock app | tori, artifact-handoff | lock 대기로 처리 지연이 생기는지 확인 |
| DPC-037 | `partial_write_recovery_delta` | 파일/DB 쓰기 app | tori, sori, node-artifact-runtime | partial write 복구가 발생하는지 확인 |
| DPC-038 | `remote_fetch_requests_delta` | remote fetch app | sori, node-artifact-runtime, JUMI | 원격 fetch churn 확인 |
| DPC-039 | `remote_fetch_failures_delta` | remote fetch app | sori, node-artifact-runtime, JUMI | remote source/network 실패 확인 |
| DPC-040 | `remote_fetch_bytes_delta` | remote fetch app | sori, node-artifact-runtime | 데이터 이동량 급증 확인 |
| DPC-041 | `process_timeouts_delta` | subprocess/supervisor app | node-artifact-runtime, spawner | timeout 증가 감지 |
| DPC-042 | `process_kills_delta` | subprocess/supervisor app | node-artifact-runtime, spawner | SIGKILL/강제 종료 증가 감지 |
| DPC-043 | `process_group_leaks_end` | subprocess/supervisor app | node-artifact-runtime | 종료 후 자식 프로세스 누수 확인 |
| DPC-044 | `run_state_transitions_delta` | state machine app | spawner, dag-go, JUMI | state transition churn이 비정상적으로 늘어나는지 확인 |
| DPC-045 | `invalid_state_transitions_delta` | state machine app | spawner, dag-go | 허용되지 않는 상태 전이가 발생하는지 확인 |
| DPC-046 | `deadlock_timeout_delta` | DAG/concurrency app | dag-go, spawner | deadlock/timeout 계열 실패를 별도 감지 |
| DPC-047 | `memory_alloc_bytes_delta` | memory-sensitive app | dag-go, JUMI, spawner | 실행량 대비 메모리 할당 급증 확인 |
| DPC-048 | `heap_objects_end` | long-running app | spawner, artifact-handoff | heap object 누수 후보 확인 |
| DPC-049 | `churn_score` | 복합 데이터 플레인 app | JUMI, artifact-handoff, spawner, dag-go | retry, failure, orphan, backlog, leftover를 합친 단일 검토 지표 후보 |
| DPC-050 | `child_processes_spawned_delta` | subprocess/container app | node-artifact-runtime, spawner | 실행 중 child process가 얼마나 생성되는지 확인 |
| DPC-051 | `child_processes_reaped_delta` | subprocess/container app | node-artifact-runtime, spawner | 종료된 child process를 실제로 wait/reap하는지 확인 |
| DPC-052 | `zombie_processes_end` | PID1/container supervisor app | node-artifact-runtime | window 종료 시 zombie process가 남는지 확인 |
| DPC-053 | `orphan_processes_end` | PID1/container supervisor app | node-artifact-runtime, spawner | parent가 죽은 뒤 떠도는 orphan process가 남는지 확인 |
| DPC-054 | `process_group_members_end` | process group 관리 app | node-artifact-runtime | 종료 후 process group에 남은 프로세스가 있는지 확인 |
| DPC-055 | `sigterm_graceful_shutdown_ms` | PID1/supervisor app | node-artifact-runtime, spawner | SIGTERM 뒤 정상 종료까지 걸리는 시간 |
| DPC-056 | `sigkill_fallback_delta` | PID1/supervisor app | node-artifact-runtime, spawner | graceful shutdown 실패 후 SIGKILL까지 간 횟수 |
| DPC-057 | `stdout_stderr_drain_errors_delta` | subprocess IO app | node-artifact-runtime, spawner | child stdout/stderr pipe 처리 실패 감지 |
| DPC-058 | `exit_code_mapping_failures_delta` | command wrapper app | node-artifact-runtime | child exit code가 termination summary/status로 잘 반영되는지 확인 |
| DPC-059 | `pid1_reaped_zombies_total_delta` | PID1/container supervisor app | node-artifact-runtime | PID1 역할로 zombie를 수거한 누적 횟수 |
| DPC-060 | `shutdown_leftover_processes_end` | container runtime helper app | node-artifact-runtime, spawner | shutdown 후 남은 프로세스 수 |

### 2.24 fuzz 테스트

fuzz 테스트는 사람이 미리 생각한 입력만 넣는 일반 테스트와 다르다. 테스트 엔진이 많은
입력 조합을 자동으로 만들어 parser, decoder, policy evaluator, path 처리, schema 처리처럼
입력 공간이 넓은 코드에서 panic, hang, 잘못된 accept/reject를 찾는다.

Go에서는 보통 `func FuzzXxx(f *testing.F)` 형태의 fuzz target을 만들고,
`go test -fuzz=FuzzXxx`로 실행한다. CI에서는 무한히 오래 돌릴 수 없으므로
`-fuzztime=30s`처럼 짧은 smoke fuzz로 두거나, nightly/scheduled workflow로 길게 돌리는
방식을 쓴다.

이번 최종 검수 기준으로 대상 12개 repo에서는 repo 자체의 `func Fuzz...` target,
`go test -fuzz=...`, `testdata/fuzz` corpus가 확인되지 않았다. `NodeVault`, `podbridge5`의
`go.sum`에 `go-fuzz-headers` dependency가 있고 `NodeVault/vendor` 아래 third-party fuzz
코드가 있지만, 이것은 대상 repo가 자체 fuzz guardrail을 운영한다는 의미로 보지 않는다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G328 | Go native fuzz target | `func FuzzXxx(f *testing.F)` | repo 자체에 fuzz 진입점이 있는지 |
| G329 | fuzz seed corpus | `testdata/fuzz/FuzzXxx/*` | 실패/흥미 입력을 seed로 보존하는지 |
| G330 | local fuzz target | `make fuzz`, `go test -fuzz=...` | 사람이 로컬에서 fuzz를 실행할 수 있는지 |
| G331 | CI fuzz smoke | GitHub Actions에서 짧은 `-fuzztime` 실행 | PR/CI에서 짧게 fuzz 회귀를 보는지 |
| G332 | scheduled/deep fuzz | schedule/nightly fuzz workflow | 긴 시간 fuzz를 정기 실행하는지 |
| G333 | fuzz artifact/corpus 보존 | crash corpus, fuzz logs artifact | fuzz 실패 입력과 로그를 보존하는지 |

### 2.25 후보 승격 항목 / 운영 세부 감사

이 그룹은 이전 버전 문서의 “다음 조사에서 추가로 번호화할 후보”였던 항목을 실제
작업 지시가 가능한 단위로 승격한 것이다. 기존 항목과 겹치는 큰 분류가 있더라도,
운영에서는 “있다/없다”보다 “어떤 방식으로 있다”가 중요하다.

예를 들어 `G253`은 golden update target이라는 큰 항목이다. 하지만 실제 적용 판단에서는
명시적인 update command가 있는지, 그 command가 guardrail에서 언급되는지, baseline이
자동 생성되는지까지 나눠야 나중에 repo별로 같은 수준을 요구할 수 있다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G334 | explicit golden update command | `UPDATE_GOLDEN=1`, `make update-golden`, 명시 fixture update command | golden 파일을 갱신하는 공식 명령이 있는지 |
| G335 | golden update guardrail/docs | golden update command를 quality guardrail, README, docs, Makefile help에서 안내/검증 | golden 갱신 절차가 숨겨져 있지 않은지 |
| G336 | snapshot/baseline fixture freeze | `testdata`, snapshot, baseline fixture를 기대값으로 고정 비교 | 현재 동작을 fixture로 잠가 회귀를 잡는지 |
| G337 | baseline auto-create/update behavior | baseline missing 시 생성, update script, comparison baseline 저장 | baseline이 없거나 바뀔 때 갱신 흐름이 있는지 |
| G338 | smoke log artifact | smoke 실행 로그를 파일로 남김 | 실패했을 때 마지막 실행 로그를 볼 수 있는지 |
| G339 | Kubernetes diagnostic artifact | pod logs, events, describe, conditions, manifest 등을 artifact로 남김 | K8s smoke 실패 원인을 CI에서 재현 없이 볼 수 있는지 |
| G340 | smoke result/manifest artifact | smoke 결과 JSON, manifest JSON, termination JSON, summary 파일 | smoke가 무엇을 검증했고 어디서 실패했는지 구조화해 남기는지 |
| G341 | remote/VM fetched smoke artifact | SSH/VM/container 내부 로그를 CI runner로 복사해 보존 | 원격 환경에서만 생긴 실패 증거를 가져오는지 |
| G342 | smoke artifact upload gate | smoke 산출물을 `actions/upload-artifact`로 업로드 | 로컬 파일로만 남기지 않고 Actions artifact로 보존하는지 |
| G343 | CodeQL vendor paths-ignore | `codeql-config.yml` 또는 workflow에서 `paths-ignore: vendor/**` | vendored third-party 코드를 기본 분석 범위에서 제외하는지 |
| G344 | CodeQL explicit query suite | CodeQL config에 `queries:` 명시 | default query가 아니라 분석 suite를 명시적으로 선택했는지 |
| G345 | CodeQL security-extended suite | `security-extended` query suite | 기본보다 넓은 보안 query를 켰는지 |
| G346 | CodeQL security-and-quality suite | `security-and-quality` query suite | 보안뿐 아니라 품질 query까지 넓혔는지 |
| G347 | checkout/setup-go v6-only | key workflow에서 `checkout@v6`, `setup-go@v6`만 확인 | 핵심 Action major version이 최신 계열로 정리됐는지 |
| G348 | mixed checkout/setup-go major versions | `checkout@v4/v6`, `setup-go@v5/v6` 혼재 | workflow별 Action major가 섞여 drift가 있는지 |
| G349 | older artifact action v4 present | `upload-artifact@v4`, `download-artifact@v4` 존재 | artifact Action 구버전 계열이 남아 있는지 |
| G350 | older golangci-lint action major present | `golangci/golangci-lint-action@v7` 등 최신 major보다 낮은 Action | lint Action wrapper major가 뒤처져 있는지 |

### 2.26 operator behavior 테스트

이 그룹은 Kubernetes Operator repo에서 특히 중요한 “동작 계약” 테스트를 따로 번호화한
것이다. 일반 unit test는 함수 결과를 보는 경우가 많지만, operator behavior test는
controller가 reconcile loop에서 어떤 상태를 만들고, 어떤 condition을 기록하고, 어떤
리소스를 생성/갱신/스킵하는지를 본다.

쉽게 말하면 “코드 조각이 맞는가”보다 “사용자가 CR을 넣었을 때 operator가 약속한
행동을 하는가”를 확인하는 테스트다. `bori`처럼 operator 방향으로 개발되는 repo에서는
이 축을 unit/smoke/integration과 분리해서 봐야 한다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G351 | operator test strategy document | `docs/testing/operator-test-strategy.md` 등 계층형 operator 테스트 전략 문서 | operator 테스트가 어떤 층으로 나뉘는지 명시했는지 |
| G352 | fake-client controller behavior test | controller-runtime fake client로 `Reconcile()` 호출 | 실제 cluster 없이 controller 동작 분기를 검증하는지 |
| G353 | reconcile status/condition patch behavior | status patch, `conditions`, `observedGeneration` assertion | reconcile 결과가 CR status에 맞게 기록되는지 |
| G354 | finalizer/deletion behavior test | finalizer 추가/삭제, deletion timestamp 처리 test | 삭제 흐름과 cleanup 계약을 검증하는지 |
| G355 | reconcile skip/idempotency behavior | observedGeneration/releaseGeneration match, idempotent/no-update test | 이미 처리한 리소스를 불필요하게 다시 처리하지 않는지 |
| G356 | policy violation condition behavior | `ViolationError`, `Violation=True`, `Degraded=True` assertion | 정책 위반이 runtime error가 아니라 condition으로 표현되는지 |
| G357 | cross-resource watch/status aggregation behavior | secondary watch, `activeDataPlanes`, related resource status aggregation | 여러 CR 사이의 상태 전파가 맞는지 |
| G358 | Ginkgo/Gomega operator E2E suite | `Describe`, `It`, `BeforeSuite`, `AfterSuite` 기반 operator suite | operator 시나리오를 사람이 읽기 쉬운 행동 단위로 표현하는지 |
| G359 | async controller assertion | `Eventually`, `Consistently`로 reconcile 결과 대기/비발생 확인 | 비동기 controller 특성을 테스트가 제대로 기다리는지 |
| G360 | kind operator behavior smoke | kind에서 CR 생성 후 finalizer/status/condition/metrics 확인 | 실제 API server 위에서 operator 기본 행동을 보는지 |
| G361 | functional reconcile-cycle behavior | fixture 주입 후 Runner.Run, revision 생성, release status까지 확인 | 성공 reconcile cycle 전체가 이어지는지 |
| G362 | digest/image behavior test | image digest 입력이 status/revision/deployed image에 반영되는지 | 이미지 digest 같은 핵심 배포 의미가 보존되는지 |

### 2.27 재검수로 추가 식별한 CI 운영 세부 항목

이번 전수 재검수에서 workflow/Makefile을 다시 대조하면서, 기존 큰 항목만으로는
repo별 운영 차이가 충분히 드러나지 않는 세부 항목을 추가했다.

이 그룹은 테스트 종류 자체보다 “CI가 얼마나 운영 가능하게 짜여 있는가”를 본다.
예를 들어 artifact upload가 있어도 실패한 job에서 upload step이 건너뛰면 디버깅
증거가 남지 않는다. `if: always()`는 그런 차이를 구분하기 위한 항목이다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G363 | self-hosted runner 사용 | `runs-on: [self-hosted, ...]` | GitHub-hosted가 아니라 관리 중인 runner에서 실행되는지 |
| G364 | direct `go vet` gate | workflow 또는 Makefile에서 `go vet` 직접 실행 | golangci의 `govet`과 별도로 Go vet을 독립 gate로 돌리는지 |
| G365 | always artifact upload | artifact upload step에 `if: always()` | 테스트가 실패해도 report/log artifact를 남기는지 |
| G366 | setup-go cache disabled | `actions/setup-go`의 `cache: false` | 의도적으로 Go cache를 끄는 workflow가 있는지 |
| G367 | local Go cache/temp isolation | `GOCACHE`, `GOTMPDIR`, `GOMODCACHE`를 repo별 `/tmp/...`로 지정 | runner의 전역 Go cache/tmp 오염을 줄이는지 |

### 2.28 테스트 더블 / 테스트 라이브러리 세부 항목

이 그룹은 테스트가 외부 시스템을 어떻게 대체하거나 관찰하는지를 본다. 같은 unit test라도
순수 함수만 검증하는 테스트와, HTTP server/DB/Kubernetes API를 test double로 세워서
계약을 검증하는 테스트는 성격이 다르다.

테스트 더블 항목은 “실제 운영 환경과 완전히 같다”는 뜻이 아니다. 대신 외부 의존성을
작고 빠르게 흉내 내서, 실패 경로와 프로토콜 계약을 더 많이 검증한다는 뜻이다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G368 | `httptest` HTTP test double | `net/http/httptest`, `httptest.NewServer`, `httptest.NewRecorder` | HTTP server/client 계약을 로컬 test double로 검증하는지 |
| G369 | SQL mock test double | `github.com/DATA-DOG/go-sqlmock` | DB 없이 SQL query/transaction 동작을 검증하는지 |
| G370 | `testify` assertion helper | `github.com/stretchr/testify/assert`, `require` | 표준 `testing` 외 assertion helper를 사용하는지 |

### 2.29 문서 11-13장 반영 / 운영 거버넌스 후보 승격

이 그룹은 11-13장의 주의사항과 다음 조사 후보를 받아들여, 로컬 파일로 확인 가능한 것은
정식 G-ID로 승격하고, GitHub 서버 설정처럼 로컬 파일만으로 확정할 수 없는 것은
“향후 조사 가능한 고정 ID”로 만든 것이다.

중요한 구분은 “문서/코드 계약”과 “CI hard gate”가 다르다는 점이다. 예를 들어
`Trivy VulnerabilityReport`를 처리하는 코드와 테스트가 있어도, 그것이 곧 Trivy image
scan workflow나 Cosign signing gate가 있다는 뜻은 아니다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G371 | CODEOWNERS file | `.github/CODEOWNERS`, `CODEOWNERS` | 코드 소유자/리뷰 책임을 파일로 선언했는지 |
| G372 | dependency update automation config | `.github/dependabot.yml`, `renovate.json` | dependency update PR 자동화를 설정했는지 |
| G373 | changelog/release notes file | `CHANGELOG.md`, `docs/RELEASE*.md`, `docs/RELEASE_NOTES*.md` | 릴리스 변경사항을 사람이 읽는 문서로 남기는지 |
| G374 | release note/changelog guardrail | workflow/script가 changelog 또는 release note 존재/변경을 검사 | 릴리스 설명 누락을 CI나 script가 막는지 |
| G375 | artifact provenance manifest model | `pkg/provenance`, `ArtifactManifest`, lineage/input provenance tests | 산출물의 digest, 위치, 입력 lineage를 구조화해 기록하는지 |
| G376 | Trivy/VulnerabilityReport contract | docs/proto/API에 `trivy`, `VulnerabilityReport`, scan record 계약 명시 | 이미지 취약점 scan 결과를 어떤 계약으로 받는지 정의했는지 |
| G377 | Trivy/VulnerabilityReport implementation test | `runL5b`, `parseTrivySummary`, `ToolScanRecord` ingestion tests | Trivy/VulnerabilityReport 또는 scan record 처리 코드가 테스트되는지 |
| G378 | conditional environment/capability skip | `t.Skip`/`t.Skipf` for missing env, KUBECONFIG, platform capability | 환경이 없을 때 테스트가 명시적으로 skip되는지 |
| G379 | historical/quarantine skip marker | historical anchor, retired baseline, quarantine reason in `t.Skip` | 의도적으로 비활성화한 테스트 이유가 코드에 남아 있는지 |
| G380 | container image scan/signing CI | Trivy/Grype/Cosign 등 image scan/sign workflow | 컨테이너 이미지 자체를 CI에서 scan/sign하는지 |
| G381 | SBOM/SLSA/provenance CI artifact | SBOM 생성, SLSA provenance, attestation upload/signing | 릴리스/이미지 공급망 증거를 CI artifact로 남기는지 |
| G382 | GitHub required checks/branch protection | GitHub branch protection/ruleset required checks 설정 | merge 전에 통과해야 할 check를 GitHub 서버 설정으로 강제하는지 |
| G383 | skip expiry/issue tracking | `t.Skip`/quarantine 문구에 issue URL, owner, expiry date, 재검토 날짜 포함 | skip이 영구 방치되지 않도록 누가 언제 다시 볼지 남기는지 |
| G384 | quarantine test registry | quarantine/flaky test 목록 파일 또는 별도 quarantine workflow | 불안정 테스트를 그냥 빼지 않고 별도 목록과 실행 경로로 관리하는지 |
| G385 | branch protection audit artifact | ruleset/branch protection 조회 결과를 문서나 artifact로 보존 | GitHub UI 설정을 나중에 감사할 수 있게 기록하는지 |
| G386 | release readiness checklist | tag/release 전 체크리스트 문서 또는 script | 릴리스 전에 통과해야 할 수동/자동 확인을 한곳에 모으는지 |
| G387 | SARIF filter justification | SARIF filter 사용 이유, 제외 범위, 제품 source alert 미은폐 근거 문서 | CodeQL 결과를 필터링할 때 왜 안전한지 설명하는지 |
| G388 | generated-code review policy | generated code를 추적/리뷰/필터링하는 기준 문서 | generated라는 이유만으로 무시하지 않고 어떤 것은 source처럼 보는지 정하는지 |
| G389 | security alert triage SLA | `SECURITY.md` 또는 보안 triage 문서에 owner/SLA/처리 흐름 명시 | 취약점 알림을 누가 언제까지 판단할지 정하는지 |
| G390 | CI failure triage runbook | CI 실패 유형별 원인, 재실행 기준, 담당자, artifact 확인법 문서 | CI가 깨졌을 때 매번 감으로 보지 않도록 조사 순서를 정리하는지 |

### 2.30 Kubernetes Operator 표준 테스트 / 가드레일 후보

이 그룹은 Kubernetes Operator repo에서 추가로 번호화할 필요가 있는 테스트와 가드레일이다.
일부는 현재 특정 repo에 이미 비슷한 형태로 있을 수 있지만, 이 문서에서는 “앞으로 적용
여부를 판단할 수 있는 표준 단위”로 먼저 고정한다.

Operator는 일반 Go 서비스와 다르다. API server, CRD schema, RBAC, admission webhook,
reconcile loop, status condition, finalizer, leader election, uninstall/upgrade 같은
운영 계약이 함께 맞아야 한다. 그래서 unit test만 많아도 실제 cluster에서는 깨질 수 있다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G391 | envtest API server/controller test | `envtest.Environment`, `setup-envtest`, controller-runtime envtest | fake client가 아니라 실제 API server/etcd 위에서 controller를 검증하는지 |
| G392 | CRD schema structural validation | CRD가 structural schema 조건을 만족하는지 검사 | CRD가 Kubernetes API server에서 안정적으로 저장/검증될 수 있는지 |
| G393 | CRD OpenAPI required/default validation | required/default/enum/min/max validation test | 잘못된 CR spec이 API server 단계에서 막히는지 |
| G394 | CRD conversion webhook test | conversion webhook, multi-version CRD conversion test | v1alpha1/v1beta1/v1 등 버전 변환이 안전한지 |
| G395 | CRD backward compatibility test | old CR fixture apply/read/update test | 예전 버전 사용자가 만든 CR이 새 controller에서도 깨지지 않는지 |
| G396 | CRD generated deepcopy drift check | `deepcopy-gen`, `zz_generated.deepcopy.go`, generate 후 diff | CRD type 변경 후 generated deepcopy가 최신인지 |
| G397 | controller RBAC least-privilege check | `controller-gen rbac`, rbac diff, kubectl auth can-i matrix | controller 권한이 너무 넓거나 부족하지 않은지 |
| G398 | manager startup smoke | manager binary starts with config/env and exposes health/ready | operator process가 기본 설정으로 부팅되는지 |
| G399 | health/readiness endpoint test | `/healthz`, `/readyz`, manager health probe test | Kubernetes probe가 실제로 동작하는지 |
| G400 | metrics endpoint test | `/metrics` scrape, controller-runtime metrics presence | Prometheus가 scrape할 operator metrics가 나오는지 |
| G401 | reconcile idempotency test | 같은 CR을 반복 reconcile해도 불필요한 write/event가 없는지 | reconcile은 반복 호출되므로 같은 입력에 안정적인지 |
| G402 | reconcile conflict retry test | optimistic lock conflict, resourceVersion conflict retry | status/spec update 충돌 시 안전하게 재시도하는지 |
| G403 | finalizer add/remove behavior test | deletionTimestamp, finalizers, cleanup assertion | 삭제 시 외부 리소스 정리 후 finalizer가 빠지는지 |
| G404 | ownerReference/garbage-collection test | child resource ownerRef, controllerRef assertion | CR 삭제 시 하위 리소스가 자연스럽게 정리되는지 |
| G405 | status condition contract test | `type`, `status`, `reason`, `message`, `observedGeneration` assertion | 사용자가 CR 상태를 보고 판단할 수 있게 condition 계약을 지키는지 |
| G406 | event emission contract test | Kubernetes Event reason/message assertion | 중요한 실패/전환이 `kubectl describe`에서 보이는지 |
| G407 | spec-to-child-resource render test | CR spec이 Deployment/Service/Job 등 child manifest로 정확히 변환되는지 | 사용자가 입력한 spec이 실제 workload 설정에 반영되는지 |
| G408 | drift correction test | child resource를 일부러 바꾼 뒤 reconcile이 복구하는지 | 사람이 수동 수정한 drift를 operator가 바로잡는지 |
| G409 | external dependency failure test | registry/API/storage unavailable fixture | 외부 의존성이 실패해도 condition/event/retry가 예측 가능한지 |
| G410 | admission webhook validation test | validating webhook allow/deny table test | 잘못된 CR을 admission 단계에서 거부하는지 |
| G411 | admission webhook mutation test | mutating webhook defaulting/normalization test | 누락 필드 default나 normalization이 기대대로 되는지 |
| G412 | webhook TLS/cert readiness test | cert-manager/self-signed cert, webhook service endpoint check | webhook 인증서/서비스 문제로 admission이 막히지 않는지 |
| G413 | leader election behavior test | leader election enabled startup, lease object check | HA 배포에서 한 manager만 active reconcile하는지 |
| G414 | multi-namespace watch test | watch namespace list, cache scope test | 여러 namespace 또는 제한 namespace 감시가 의도대로 동작하는지 |
| G415 | cluster-scoped vs namespaced mode test | ClusterRole/Role, namespace-scoped install mode | cluster-wide 설치와 namespace 한정 설치를 구분 검증하는지 |
| G416 | install/upgrade smoke test | `kubectl apply`, Helm/Kustomize upgrade, old-to-new manifests | 기존 설치를 새 버전으로 올려도 CR/상태가 유지되는지 |
| G417 | uninstall cleanup test | delete manifests/CRD, leftover resource scan | 제거 후 webhook, role, deployment, CR, finalizer가 남지 않는지 |
| G418 | CR deletion during reconcile test | reconcile 중 CR 삭제 race test | 처리 중 삭제되어도 panic/리소스 누수가 없는지 |
| G419 | controller panic recovery test | panic path, recover/log/requeue behavior | 한 reconcile panic이 manager 전체 장애로 번지지 않는지 |
| G420 | rate-limit/backoff behavior test | retry/backoff interval assertion | 실패 시 너무 빠르게 재시도해 API server를 두드리지 않는지 |
| G421 | workqueue saturation guardrail | queue depth/retry SLI threshold, stress smoke | queue backlog가 쌓이는 회귀를 CI에서 감지하는지 |
| G422 | API server request budget guardrail | rest_client 429/5xx/qps SLI threshold | operator가 API server를 과도하게 호출하지 않는지 |
| G423 | reconcile latency SLI guardrail | reconcile duration p95/p99 SLI threshold | reconcile이 느려지는 회귀를 운영 지표로 막는지 |
| G424 | orphan child resource guardrail | orphan Deployment/Job/Pod/Secret scan | ownerRef/finalizer 문제로 orphan 리소스가 남는지 확인 |
| G425 | stuck terminating resource guardrail | terminating CR/Pod/Job age scan | finalizer나 cleanup 실패로 삭제가 멈추는지 감지 |
| G426 | condition freshness guardrail | condition `lastTransitionTime`, stale condition age check | 상태가 오래 갱신되지 않는 silent failure를 찾는지 |
| G427 | CRD examples apply test | docs/examples/samples CR을 실제 apply dry-run/server-side validate | README/sample YAML이 실제로 API server에 들어가는지 |
| G428 | Helm chart/Kustomize render validation | `helm template`, `kustomize build`, schema/kubeconform | 배포 패키지가 유효한 Kubernetes manifest를 만드는지 |
| G429 | OLM bundle validation | `operator-sdk bundle validate`, catalog/bundle test | OLM 배포용 bundle이 올바른지 |
| G430 | operator release image digest pinning | manager image digest pin, mutable tag guardrail | 릴리스 manifest가 mutable tag가 아니라 digest로 고정되는지 |
| G431 | Pod security context guardrail | runAsNonRoot, readOnlyRootFilesystem, capabilities drop 검사 | operator Pod가 과도한 권한으로 뜨지 않는지 |
| G432 | NetworkPolicy presence/egress guardrail | NetworkPolicy manifest, allowed egress review | operator 통신 범위를 명시적으로 제한하는지 |
| G433 | secret/config redaction test | log output에 token/password/secret 미노출 test | reconcile 실패 로그에 민감정보가 새지 않는지 |
| G434 | namespace cleanup safety test | cleanup code가 target namespace 밖 리소스를 지우지 않는지 | 잘못된 cleanup으로 cluster-wide 피해가 나지 않게 하는지 |
| G435 | disaster recovery/restart test | manager restart 후 in-flight CR 상태 복구 | operator 재시작 뒤에도 작업 상태를 이어서 수습하는지 |
| G436 | concurrent CR reconcile test | 여러 CR 동시 생성/수정 stress test | 여러 사용자가 동시에 CR을 만들 때 race나 starvation이 없는지 |
| G437 | clock/time skew tolerance test | lease/deadline/TTL/expiry time skew fixture | 시간 차이 때문에 lease나 만료 판단이 깨지지 않는지 |
| G438 | dry-run/server-side apply compatibility | server-side dry-run, SSA field manager conflict test | kubectl/server-side apply 사용자와 충돌하지 않는지 |
| G439 | CR status subresource enforcement test | spec update와 status update 경로 분리 확인 | status가 spec update 경로로 섞여 들어가지 않는지 |
| G440 | operator runbook/troubleshooting doc | common conditions, events, logs, recovery steps 문서 | 운영자가 장애 때 무엇을 봐야 하는지 문서화했는지 |

### 2.31 컨테이너 데이터 플레인 / PID1 / process supervision 테스트 후보

이 그룹은 컨테이너 안에서 실제 command, worker, subprocess를 실행하는 데이터 플레인 app에
필요한 테스트와 가드레일이다. 특히 `node-artifact-runtime`, `spawner`처럼 “프로세스를
시작하고, 종료시키고, 결과를 요약하고, artifact를 남기는” repo에서 중요하다.

컨테이너에서는 앱이 PID1이 되는 경우가 많다. PID1은 일반 프로세스와 다르게 signal 처리,
child process reap, zombie 수거, process group cleanup을 제대로 해야 한다. 이 부분이
약하면 테스트는 PASS인데 실제 컨테이너에는 zombie process, orphan process, temp file,
깨진 termination summary가 남을 수 있다.

| ID | 항목 | 확인 기준 | 쉬운 설명 |
|---|---|---|---|
| G441 | PID1 signal handling test | PID1 모드에서 SIGTERM/SIGINT 수신 후 shutdown assertion | 컨테이너가 종료 신호를 받았을 때 정상 정리되는지 |
| G442 | child process reap test | child process 종료 후 `Wait`/reap 확인 | zombie process가 남지 않도록 자식 종료를 수거하는지 |
| G443 | zombie process regression test | intentionally short-lived child, `/proc` zombie scan | 종료된 자식이 zombie 상태로 남지 않는지 |
| G444 | orphan process cleanup test | parent/child 분리 fixture, orphan scan | 부모가 먼저 죽어도 떠도는 자식 프로세스를 정리하는지 |
| G445 | process group termination test | child가 grandchild를 만들고 process group kill 확인 | 하위 프로세스 트리 전체가 종료되는지 |
| G446 | graceful then force kill test | SIGTERM grace period 후 SIGKILL fallback 확인 | 정상 종료가 안 될 때 강제 종료까지 가는지 |
| G447 | timeout termination summary test | timeout 발생 시 status/exitCode/reason summary assertion | timeout이 사용자에게 명확한 결과로 기록되는지 |
| G448 | exit code propagation test | child exit code가 wrapper exit/status에 반영되는지 | 실제 command 실패가 성공으로 오인되지 않는지 |
| G449 | stdout/stderr drain test | large stdout/stderr child process, pipe drain assertion | 로그 pipe가 막혀 child가 hang되지 않는지 |
| G450 | log redaction for child output | child output에 secret 포함 fixture, redaction assertion | command 로그에 token/password가 새지 않는지 |
| G451 | working directory cleanup test | temp workdir/materialization dir cleanup assertion | 실행 후 임시 디렉터리가 남지 않는지 |
| G452 | file descriptor leak test | repeated child runs, open fd count comparison | 실행 반복 후 fd가 누수되지 않는지 |
| G453 | process environment isolation test | env allowlist/denylist, secret env stripping | child process에 불필요한 환경변수가 전달되지 않는지 |
| G454 | uid/gid/user switching test | runAs user/group, permission failure fixture | 컨테이너 내부 권한 모델이 기대대로 적용되는지 |
| G455 | read-only filesystem behavior test | read-only rootfs 또는 unwritable path fixture | writable path 가정 때문에 컨테이너에서 깨지지 않는지 |
| G456 | cgroup/resource limit behavior test | memory/cpu limit fixture, OOM/timeout summary | 리소스 제한 상황을 명확히 감지하고 기록하는지 |
| G457 | init wrapper compatibility test | tini/dumb-init/no-init mode comparison | init wrapper 유무에 따라 signal/reap 동작이 깨지지 않는지 |
| G458 | concurrent subprocess cleanup test | 여러 child 동시 실행 후 cleanup assertion | 병렬 실행 시 process group/zombie cleanup이 안전한지 |
| G459 | interrupted artifact consistency test | 실행 중 interrupt 후 partial artifact/summary 상태 확인 | 중단된 실행이 성공 artifact처럼 보이지 않는지 |
| G460 | container supervision runbook | PID1, signal, timeout, zombie, cleanup troubleshooting 문서 | 운영자가 프로세스 잔류/종료 문제를 조사할 수 있는지 |

## 3. 저장소별 적용 ID 목록

### 3.1 NodeVault

확인 소스: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`,
`.golangci.yml`, `.kube-linter.yaml`, `buf.yaml`, `Makefile`

CI:

- G001, G002, G003, G004, G007, G008, G010
- G181, G182, G183, G185, G186, G187, G188, G191, G192, G363, G364, G365
- G018, G019, G020, G021, G022, G023, G024, G028
- G031, G032, G033, G034, G035, G036, G038
- G044, G045, G048, G049
- G204, G205, G206, G207, G210, G211, G212, G213, G215, G217
- G057, G058, G060, G061, G062, G064, G065, G067, G068, G069, G070, G071, G072, G073, G074, G075, G077, G078, G079, G080, G083, G085, G086, G087
- G089, G090, G095, G096, G097, G103
- G218, G224
- G105, G106, G107
- G110, G111, G113, G114
- G119, G120, G122, G127, G129, G130, G132
- G236, G243, G250
- G136, G138, G140, G141, G143, G368, G376, G377, G378
- G145, G147, G153, G155
- G255, G261, G263
- G171, G172
- G281, G282, G283, G286, G297, G327

Local/Partial:

- G047, G052, G108, G131, G133, G148, G150
- G195, G198, G202, G240, G241, G242

메모:

- `govulncheck`는 exception gate가 있어 hard gate 성격이다.
- SLI/kube-slint 측정이 CI에 있고, `slint-gate` CLI가 아니라 Go test assertion으로 gate한다.

### 3.2 podbridge5

확인 소스: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`,
`.github/workflows/vm-runtime-test.yml`, `.golangci.yml`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G007, G008, G009, G010, G014, G015, G016
- G181, G182, G183, G185, G186, G187, G188, G192, G193, G194, G363, G364, G365
- G018, G019, G020, G021, G022, G023, G024, G026, G028
- G031, G032, G033, G037
- G044, G045, G051
- G204, G205, G208, G214, G215, G224
- G057, G061, G063, G068, G070, G071, G072, G074, G076, G077, G079, G080, G081, G082, G083, G084, G086, G087, G088
- G092, G093, G094, G095, G096, G097, G103, G104
- G218, G219, G220, G221, G223
- G110, G113, G114, G115
- G119, G120, G122, G127, G128, G129, G130, G132
- G236, G239, G243, G248, G249
- G136, G141, G142, G143, G378
- G163, G164, G165
- G201, G267, G268

Local/Partial:

- G052, G108, G131, G133
- G195, G241

Observe:

- G037

메모:

- `govulncheck`는 `continue-on-error: true`라 hard gate가 아니다.
- VM workflow는 runtime test와 integration test를 분리한다.

### 3.3 JUMI

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/security-observe.yml`, `.github/workflows/semgrep.yml`,
`.github/workflows/kube-linter.yml`, `.github/workflows/quality-guardrails.yml`,
`.github/workflows/registry-sync-smoke.yml`, `.github/workflows/sprint-baseline.yml`,
`.github/workflows/codeql.yml`, `.golangci.yml`, `.semgrep/rules`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G006, G007, G008, G009, G010
- G181, G182, G183, G184, G185, G186, G187, G188, G192, G193, G194, G365, G367
- G018, G019, G020, G021, G022, G023, G026, G027, G028
- G031, G032, G037, G040, G041, G042, G043
- G044, G046, G048, G050
- G204, G205, G206, G210
- G059, G061, G069, G071, G072, G074, G080, G083, G087
- G098, G101, G097
- G226, G227, G228
- G105, G106, G108
- G110, G115, G116, G118
- G119, G121, G129, G130, G131, G132, G133
- G236, G237, G238, G241, G242, G253
- G136, G138, G139, G140, G141, G142, G375
- G155
- G167, G169, G170
- G199, G270, G271
- G173, G174, G175
- G281, G282, G283, G288, G289, G293, G297, G326

Local/Partial:

- G042, G109, G150, G364

Observe:

- G037, G040

메모:

- Semgrep scan과 rule test가 모두 있다.
- cross-repo baseline과 remote smoke가 있어 workflow 표면이 넓다.

### 3.4 bori

확인 소스: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`,
`.github/workflows/golangci-lint.yaml`, `.github/workflows/kubelint.yaml`,
`.github/workflows/kubeconform.yaml`, `.github/workflows/generate-check.yaml`,
`.github/workflows/kind-boot-smoke.yml`, `.github/workflows/kind-functional-smoke.yml`,
`.github/workflows/kind-digest-smoke.yml`, `.github/workflows/vm-integration.yml`,
`.golangci.yml`, `.kube-linter.yaml`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G006, G008, G009, G010, G013
- G181, G182, G183, G184, G185, G186, G187, G188, G192, G193, G363, G365, G366
- G018, G019, G020, G021, G022, G023, G028
- G044, G045
- G209
- G061, G071, G072, G080, G083, G087
- G097
- G110, G111, G117
- G119, G120
- G136, G138, G140, G141, G142
- G351, G352, G353, G354, G355, G356, G357, G358, G359, G360, G361, G362
- G153, G154, G156, G157, G158, G159, G160, G161
- G262, G264, G265, G266
- G164, G165
- G201, G267
- G281, G285, G288, G289, G296, G297

Local/Partial:

- G042, G108, G150

메모:

- Kubernetes/kind/operator 관련 가드레일이 강하다.
- operator behavior test는 fake-client controller test, Ginkgo/Gomega kind behavior
  smoke, functional/digest scenario까지 분리되어 있다.
- kube-slint는 kind/VM smoke에서 summary-only/비치명 관찰 성격으로 붙어 있다.
- security/vuln 쪽은 CI 기준으로는 약하게 보인다.

### 3.5 sori

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/security-observe.yml`, `.github/workflows/release.yml`,
`.github/workflows/codeql.yml`, `.golangci.yml`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G006, G008, G010
- G181, G182, G183, G185, G186, G187, G188, G192, G193, G367
- G018, G019, G020, G021, G022, G025, G028
- G031, G032, G037, G040, G043
- G044, G046, G050, G052
- G204, G205, G206, G210, G214, G215, G216
- G058, G059, G060, G061, G066, G067, G069, G071, G072, G073, G074, G075, G079, G080, G083, G085, G086, G087
- G091, G098, G101, G102, G103
- G226, G227, G229
- G105, G106, G108
- G116
- G119, G122, G126, G129, G130, G131, G132
- G236, G238, G243, G251
- G136, G141, G142, G368, G370, G378
- G167, G168
- G176, G177
- G199, G272, G273, G274, G275

Local/Partial:

- G042, G109, G110, G111, G133, G364

Observe:

- G037, G040

메모:

- pure OCI library 경계를 `depguard`로 강제한다.
- release workflow가 별도로 있다.

### 3.6 NodeSentinel

확인 소스: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`,
`.golangci.yml`, `buf.yaml`, `Makefile`

CI:

- G001, G002, G003, G004, G007, G008, G009, G010, G017
- G181, G182, G183, G185, G186, G187, G188, G192, G364, G365
- G018, G019, G020, G021, G022, G025, G028
- G031, G032, G036
- G044, G045, G047, G048, G049, G052
- G204, G205, G206, G208, G210, G211, G213, G214, G217
- G057, G058, G061, G062, G065, G068, G070, G071, G072, G074, G077, G080, G081, G082, G083, G085, G086, G087
- G089, G090, G093, G094, G095, G096, G097, G103
- G218, G220, G224
- G105, G106, G107, G109
- G110, G116
- G119, G120, G122, G123, G124, G129, G130, G131, G132, G133
- G236, G240, G242, G243, G246, G247
- G136, G140, G143, G368, G376, G377
- G145, G147, G162
- G195, G255

Local/Partial:

- G150

메모:

- actionlint가 명시적으로 있다.
- coverage threshold와 race/shuffle 테스트가 CI에 있다.

### 3.7 NodePalette

확인 소스: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`,
`.golangci.yml`, `Makefile`

CI:

- G001, G002, G003, G004, G007, G008, G010
- G181, G182, G183, G185, G186, G187, G188, G192, G363, G364, G365
- G018, G019, G020, G021, G022, G023, G028
- G031, G032, G037
- G044, G045, G048, G049, G052
- G204, G205, G206, G208, G210, G211, G214, G217
- G057, G058, G061, G062, G065, G068, G070, G071, G072, G074, G077, G080, G083, G085, G086, G087
- G089, G090, G094, G095, G096, G097, G103
- G218
- G105, G106, G107, G108
- G110
- G119, G120, G122, G129, G130, G131, G132, G133
- G236, G240, G242, G243
- G136, G140, G143, G368
- G162
- G195

Local/Partial:

- 없음으로 확인

Observe:

- G037

메모:

- 작은 repo지만 build/vet/race/coverage/vuln/K8s contract가 CI에 있다.
- `govulncheck`는 `continue-on-error: true`라 hard gate가 아니라 observe 성격이다.

### 3.8 artifact-handoff

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/proto-contract.yml`, `.github/workflows/codeql.yml`,
`.golangci.yml`, `buf.yaml`, `buf.gen.yaml`, `Makefile`

CI:

- G001, G002, G003, G004, G006, G008, G009, G010
- G181, G182, G183, G185, G186, G187, G188, G192, G367
- G018, G019, G020, G021, G022, G023, G028
- G044, G046, G048, G049
- G204, G205, G206, G210
- G059, G061, G070, G071, G072, G074, G080, G083, G087
- G098, G097
- G226, G227, G230
- G105, G106
- G110
- G119, G121, G129, G130, G131, G132, G133
- G236, G238, G241, G242
- G136, G138, G140, G142, G368
- G145, G146, G148, G149, G150, G151
- G256, G257, G258, G259, G260

Local/Partial:

- G031, G032, G040, G042, G108, G364

메모:

- protobuf 계약 검사가 가장 강한 축이다.
- security/vuln target은 있으나 정규 CI gate로는 확인되지 않는다.

### 3.9 node-artifact-runtime

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/smoke.yml`, `.github/workflows/codeql.yml`, `.golangci.yml`,
`Makefile`

CI:

- G001, G002, G003, G004, G005, G006, G008, G009, G010
- G181, G182, G183, G185, G186, G187, G188, G192, G193, G367
- G018, G019, G020, G021, G022, G025, G028
- G044, G046, G048, G049
- G204, G205, G206, G210
- G059, G061, G070, G071, G072, G074, G080, G083, G087
- G093, G098, G097
- G221, G222, G226, G227, G231
- G105, G106
- G119, G129, G130, G131, G132, G133
- G236, G238, G241, G242
- G136, G141, G142, G368, G373, G375, G378
- G166
- G200, G269

Local/Partial:

- G031, G032, G040, G042, G108, G110, G111, G364

메모:

- PID 1 container smoke가 별도 workflow로 있다.
- build/vet/security/vuln은 Makefile에는 있으나 정규 CI gate로는 약하다.

### 3.10 spawner

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/security-observe.yml`, `.github/workflows/codeql.yml`,
`.golangci.yml`, `Makefile`

CI:

- G001, G002, G003, G004, G006, G007, G008, G010
- G181, G182, G183, G185, G186, G187, G188, G192, G367
- G018, G019, G020, G021, G022, G023, G028
- G031, G032, G037, G040, G043
- G044, G046, G050, G052
- G204, G205, G206, G210, G211, G212, G215, G216
- G058, G059, G060, G061, G066, G067, G069, G071, G072, G073, G074, G075, G079, G080, G083, G085, G086, G087
- G091, G098, G101, G102
- G226, G227, G232
- G105, G106
- G110, G116
- G119, G122, G129, G130, G131, G132
- G236, G238, G243
- G136, G138, G143
- G199

Local/Partial:

- G042, G108, G125, G133, G364
- G241, G242, G244, G245

Observe:

- G037, G040

메모:

- lifecycle/race 전용 target은 있으나 CI regular gate로는 확인되지 않는다.

### 3.11 tori

확인 소스: `.github/workflows/core-ci.yml`, `.github/workflows/security-observe.yml`,
`.github/workflows/codeql.yml`, `.golangci.yml`, `buf.yaml`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G007, G008, G010
- G181, G182, G183, G185, G186, G187, G188, G192
- G018, G019, G020, G021, G022, G025, G028
- G031, G032, G037, G040, G043
- G044, G046, G048, G049
- G204, G205, G206, G210
- G059, G061, G071, G072, G074, G080, G083, G087
- G098, G099, G097
- G226, G227, G233, G234
- G105, G106
- G112
- G119, G121, G122
- G136, G139, G140, G141, G369, G378, G379
- G145, G147, G152
- G199, G255

Local/Partial:

- G042, G108, G364

Observe:

- G037, G040

메모:

- core scope 중심이다. full runtime/transport scope가 아니라 `config/db/rules/block/cmd` 중심이다.

### 3.12 dag-go

확인 소스: `.github/workflows/test.yml`, `.github/workflows/lint.yml`,
`.github/workflows/security-observe.yml`, `.github/workflows/bench.yml`,
`.github/workflows/codeql.yml`, `.golangci.yml`, `Makefile`

CI:

- G001, G002, G003, G004, G005, G006, G007, G008, G009, G010, G011, G012
- G181, G182, G183, G185, G186, G187, G188, G189, G191, G192, G193
- G018, G019, G020, G021, G022, G023, G028
- G031, G032, G037, G040, G043
- G044, G046, G048, G049, G052
- G204, G205, G206, G210, G211, G212, G215, G216
- G058, G059, G060, G061, G066, G067, G069, G071, G072, G073, G074, G075, G079, G080, G083, G085, G086, G087
- G090, G091, G098, G100, G101, G102, G097
- G226, G227, G229, G235
- G105, G106
- G110, G116
- G119, G122, G129, G130, G131, G132, G134, G135
- G236, G238, G243
- G136, G143, G144, G373, G378
- G178, G179
- G199, G203, G276, G277, G278

Local/Partial:

- G042, G108, G133, G180
- G279, G280

Observe:

- G037, G040

메모:

- 순수 Go library boundary를 `depguard` allow/deny로 강하게 둔다.
- benchmark와 coverage Pages publish가 있다.

## 4. 내부 린터 빠른 비교

| repo | 적용 내부 린터 ID |
|---|---|
| `NodeVault` | G057,G058,G060,G061,G062,G064,G065,G067,G068,G069,G070,G071,G072,G073,G074,G075,G077,G078,G079,G080,G083,G085,G086,G087 |
| `podbridge5` | G057,G061,G063,G068,G070,G071,G072,G074,G076,G077,G079,G080,G081,G082,G083,G084,G086,G087,G088 |
| `JUMI` | G059,G061,G069,G071,G072,G074,G080,G083,G087 |
| `bori` | G061,G071,G072,G080,G083,G087 |
| `sori` | G058,G059,G060,G061,G066,G067,G069,G071,G072,G073,G074,G075,G079,G080,G083,G085,G086,G087 |
| `NodeSentinel` | G057,G058,G061,G062,G065,G068,G070,G071,G072,G074,G077,G080,G081,G082,G083,G085,G086,G087 |
| `NodePalette` | G057,G058,G061,G062,G065,G068,G070,G071,G072,G074,G077,G080,G083,G085,G086,G087 |
| `artifact-handoff` | G059,G061,G070,G071,G072,G074,G080,G083,G087 |
| `node-artifact-runtime` | G059,G061,G070,G071,G072,G074,G080,G083,G087 |
| `spawner` | G058,G059,G060,G061,G066,G067,G069,G071,G072,G073,G074,G075,G079,G080,G083,G085,G086,G087 |
| `tori` | G059,G061,G071,G072,G074,G080,G083,G087 |
| `dag-go` | G058,G059,G060,G061,G066,G067,G069,G071,G072,G073,G074,G075,G079,G080,G083,G085,G086,G087 |

## 5. kube-slint/slint-gate 가드레일 적용 비교

| repo | 적용 kube-slint/slint-gate ID | 해석 |
|---|---|---|
| `NodeVault` | G281,G282,G283,G286,G297,G327 | expvar 기반 SnapshotFetcher로 SLI를 측정하고 Go test assertion으로 gate하는 적용 사례 |
| `JUMI` | G281,G282,G283,G288,G289,G293,G297,G326 | JSON fixture/static fetcher로 summary를 만들고 slint-gate CLI로 정책 평가하는 통합 사례 |
| `podbridge5` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `bori` | G281,G285,G288,G289,G296,G297 | kind/VM smoke에서 kube-slint session 또는 slint-gate summary-only 흐름으로 SLI summary와 gate summary artifact를 남기는 관찰형 적용 사례 |
| `sori` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `NodeSentinel` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `NodePalette` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `artifact-handoff` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `node-artifact-runtime` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `spawner` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `tori` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |
| `dag-go` | 없음으로 확인 | 현재 조사 범위에서는 kube-slint/slint-gate 적용 확인 안 됨 |

## 6. 테스트 성격 빠른 비교

| repo | 테스트 성격 ID | 해석 |
|---|---|---|
| `NodeVault` | G136,G138,G140,G141,G143 | unit, SLI/reconcile 회귀, 계약, SLI gate, race |
| `podbridge5` | G136,G141,G142,G143 | unit, VM runtime smoke, VM integration, race |
| `JUMI` | G136,G138,G139,G140,G141,G142 | unit, regression, fixture/golden, cross-repo contract, registry/remote smoke |
| `bori` | G136,G138,G140,G141,G142,G351-G362 | unit, regression, Kubernetes/generated/schema contract, kind/VM, operator behavior |
| `sori` | G136,G141,G142 | unit, CLI/registry smoke, registry integration |
| `NodeSentinel` | G136,G140,G143 | unit, Kubernetes/protobuf contract, race/shuffle |
| `NodePalette` | G136,G140,G143 | unit, Kubernetes contract, race |
| `artifact-handoff` | G136,G138,G140,G142 | unit, resolver regression, protobuf contract, HTTP/gRPC resolver |
| `node-artifact-runtime` | G136,G141,G142 | unit, PID 1 smoke, container execution path |
| `spawner` | G136,G138,G143 | unit, lifecycle regression, race/lifecycle |
| `tori` | G136,G139,G140,G141 | core unit, fixture, protobuf/import boundary, shared-fs/NAS smoke |
| `dag-go` | G136,G143,G144 | unit, race, benchmark/performance |

### 6.1 operator behavior 테스트 적용 비교

현재 대상 repo 중 operator behavior test가 뚜렷하게 확인되는 것은 `bori`다. `NodeVault`도
operator 성격이 강하지만, 이번 문서의 신규 `G351-G362`는 `bori`에서 확인된 controller
behavior/kind behavior 구조를 기준으로 번호화했다. 다른 operator repo에 같은 기준을
적용하려면 fake client, envtest, kind, real cluster 중 어느 층에서 행동을 검증하는지
따로 확인해야 한다.

| repo | operator behavior ID | 해석 |
|---|---|---|
| `bori` | G351,G352,G353,G354,G355,G356,G357,G358,G359,G360,G361,G362 | operator 테스트 전략 문서가 있고, fake-client controller behavior, Ginkgo/Gomega kind behavior, functional reconcile cycle, digest behavior까지 확인됨 |
| 그 외 대상 repo | 없음으로 확인 | 이번 조사 범위에서는 `bori` 수준의 operator behavior test 체계가 별도 축으로 확인되지 않음 |

### 6.2 테스트 더블 / 테스트 라이브러리 적용 비교

이 표는 테스트가 외부 시스템을 어떻게 흉내 내는지 빠르게 보기 위한 표다. `httptest`는
HTTP client/server 계약을 빠르게 검증하는 데 유용하고, `sqlmock`은 실제 DB 없이 SQL
호출 계약을 검증한다. `testify`는 assertion 표현력을 높이는 도구지만, 그 자체가 테스트
범위를 넓히는 것은 아니므로 `G370`은 테스트 스타일 정보로 해석한다.

| repo | 테스트 더블/라이브러리 ID | 해석 |
|---|---|---|
| `NodeVault` | G368 | catalog/registry/build 계열에서 `httptest`로 HTTP/registry 계약을 검증함 |
| `sori` | G368,G370 | OCI/remote fetch 계열에서 `httptest`를 쓰고, 일부 테스트에서 `testify/assert`를 사용함 |
| `NodeSentinel` | G368 | vault/client/worker 계열에서 `httptest`로 외부 HTTP 의존성을 대체함 |
| `NodePalette` | G368 | palette client/server 테스트에서 `httptest` server/recorder를 사용함 |
| `artifact-handoff` | G368 | resolver/metrics HTTP API 테스트에서 `httptest` recorder/request를 폭넓게 사용함 |
| `node-artifact-runtime` | G368 | runtime helper의 remote source 테스트에서 `httptest`를 사용함 |
| `tori` | G369 | DB package 테스트에서 `go-sqlmock`으로 SQL 호출 계약을 검증함 |
| 그 외 대상 repo | 없음으로 확인 | 이번 조사 범위에서는 별도 test double/library 패턴을 G-ID로 줄 만큼 확인하지 않음 |

## 7. 실패 경로 테스트 세부 비교

아래 표는 workflow 이름이 아니라 실제 `*_test.go`의 테스트 함수명과 testdata 이름을
기준으로 확인한 것이다. 수치는 단순 grep 기반 후보 개수라서 정확한 coverage 수치가
아니며, 어떤 실패 유형이 명시적으로 존재하는지 보는 용도다.

| repo | 실패 경로 세부 ID | 관찰된 예 |
|---|---|---|
| `NodeVault` | G306,G307,G308,G309,G310,G311,G312,G313,G314,G315,G316,G317,G318,G319,G320,G321,G325 | malformed JSON/YAML, NotFound, duplicate digest/build id, empty fields, negative timestamps, unsupported recipe variant, unpinned/root-user Dockerfile rejection, registry 401/500, profiler timeout, builder/store/referrer failures |
| `podbridge5` | G306,G307,G309,G311,G314,G315,G316,G317,G319,G321,G322,G323,G325 | invalid option/path/input, file not found, nil store, unknown storage/isolation, context cancellation during retry, runtime/buildah error propagation, compare fail paths, shutdown/cleanup errors |
| `JUMI` | G307,G308,G309,G310,G311,G313,G315,G316,G318,G319,G320,G321,G322,G324,G325 | run/node/attempt not found, duplicate run/node/output, missing IDs/images/bindings, negative retry/timeout, unknown schema/location, HTTP server errors, cancel/fast-fail, artifact registration/finalize/GC failures, bad fixture matrix, reserved label/policy rejection |
| `bori` | G307,G313,G316,G318,G319 | revision/release not found, runner/deploy failures, zero/no dataplane cases, drift reconciliation |
| `sori` | G306,G307,G308,G309,G310,G311,G313,G316,G317,G319,G320,G321,G324 | invalid config/JSON/catalog/chunk, missing files/objects, duplicate catalog cases, empty metadata, negative/invalid benchmark gates, registry/remote errors, integrity/contract/digest/path failures |
| `NodeSentinel` | G306,G307,G311,G313,G316,G317,G318,G319,G320 | invalid ingress/request data, sqlite/store not found/error paths, worker classification/failure paths, vault client/server errors, k8s manifest contract |
| `NodePalette` | G306,G307,G313,G316,G319,G320 | invalid request/CLI/server paths, palette client/server errors, no-data cases, k8s manifest contract |
| `artifact-handoff` | G306,G307,G308,G309,G311,G313,G316,G318,G319,G320,G321,G322,G325 | missing attempt/artifact/producer, unknown consume policy/state, digest mismatch, HTTP/gRPC errors, pending/missing/producer-failed resolver outcomes, GC blocked/expired, HTTP source allowlist rejection |
| `node-artifact-runtime` | G306,G307,G308,G309,G310,G311,G313,G314,G315,G316,G317,G318,G320,G321,G323,G325 | invalid JSON/schema/timeout/size, missing contract/output, duplicate output, unsupported scheme/schema, command failure, timeout kill/process group, digest/size mismatch, loopback/disallowed host/signed URL/path escape/symlink rejection |
| `spawner` | G306,G307,G308,G309,G311,G314,G315,G316,G318,G319,G320,G322,G323,G325 | invalid commands/policies, missing runtime namespace/salt, conflict/not-found recovery, timeout, context cancel, failed/cancelled events, retry after unknown outcome, no leak/no deadlock, invalid state/admission rejection |
| `tori` | G306,G307,G308,G310,G311,G316,G317,G319,G320,G324 | invalid row/binary/files, missing root/file/roles/header, duplicate collision typed errors, negative index, query execution errors, no update/no rows semantics, fixture freeze |
| `dag-go` | G307,G308,G311,G314,G315,G316,G318,G319,G322,G323,G325 | missing node/no runner, duplicate edge/start, invalid transitions/config, context cancel/timeouts, parent failure propagation, failed node/run, zero/no-node edges, retry/error policy, goroutine leak/deadlock tests |

## 8. fuzz 테스트 적용 비교

최종 검수에서 vendor/third-party를 제외하고 `func Fuzz`, `go test -fuzz`,
`-fuzztime`, `testdata/fuzz`를 검색했다. 대상 12개 repo에서는 자체 fuzz guardrail이
확인되지 않았다.

| repo | fuzz 관련 ID | 해석 |
|---|---|---|
| `NodeVault` | 없음으로 확인 | `go.sum`/`vendor`에는 third-party fuzz 관련 코드가 있으나 repo 자체 fuzz target은 아님 |
| `podbridge5` | 없음으로 확인 | `go.sum`에는 `go-fuzz-headers`가 있으나 repo 자체 fuzz target은 아님 |
| `JUMI` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `bori` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `sori` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `NodeSentinel` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `NodePalette` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `artifact-handoff` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `node-artifact-runtime` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `spawner` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `tori` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |
| `dag-go` | 없음으로 확인 | `func Fuzz...`, `-fuzz`, `testdata/fuzz` 미확인 |

## 9. 버전 감사표

이 표는 “어떤 가드레일이 있는가”와 별도로 “어떤 버전으로 실행되는가”를 확인하기 위한
최종 감사표다. 같은 G-ID라도 Action major version이나 tool version이 다르면 운영 위험이
달라질 수 있다. 예를 들어 `G008`은 v6 계열 Action 사용 여부를 뜻하지만, 어떤 repo는
CodeQL workflow만 v6이고 일반 test/lint workflow는 v4/v5일 수 있다. 그래서 이 표에서는
버전 혼재를 별도로 드러낸다.

| repo | Go 기준 | Actions 버전 | 보안/품질 도구 버전 | 특이 사항 |
|---|---|---|---|---|
| `NodeVault` | go 1.25.12 | checkout@v6<br>setup-go@v6<br>upload-artifact@v7<br>codeql-init@v4<br>codeql-analyze@v4<br>golangci-action@v9<br>buf-action@v1 | golangci v2.11.3<br>kube-linter v0.8.3<br>golangci-action version v2.11.3 | CodeQL manual build<br>CodeQL actions matrix |
| `podbridge5` | go 1.25.6<br>required 1.25.6 | checkout@v4, v6<br>setup-go@v6<br>upload-artifact@v4, v7<br>codeql-init@v4<br>codeql-analyze@v4<br>golangci-action@v9<br>ssh-agent@v0.9.0 | golangci-action version latest | CodeQL manual build<br>CodeQL actions matrix<br>govulncheck observe<br>v6와 v4/v5 혼재 |
| `JUMI` | go 1.26.5<br>workflow go-version 1.26.5 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4<br>codeql-init@v4<br>codeql-analyze@v4 | golangci v2.11.3<br>govulncheck v1.1.4<br>kube-linter install v0.8.3 | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |
| `bori` | go 1.26.0 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4, v7<br>codeql-init@v4<br>codeql-analyze@v4<br>golangci-action@v9<br>kube-linter-action@v1 | golangci-action version latest<br>kind v0.24.0<br>Kubernetes 1.30.0 / v1.30.0 | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |
| `sori` | go 1.25.11 | checkout@v6<br>setup-go@v6<br>upload-artifact@v7<br>codeql-init@v4<br>codeql-analyze@v4 | golangci v2.11.3<br>govulncheck v1.1.4 | CodeQL manual build<br>CodeQL actions matrix |
| `NodeSentinel` | go 1.25.11 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4<br>codeql-init@v4<br>codeql-analyze@v4<br>golangci-action@v7<br>buf-action@v1 | golangci v2.11.3<br>golangci-action version `${{ env.GOLANGCI_LINT_VERSION }}` | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |
| `NodePalette` | go 1.25.5 | checkout@v6<br>setup-go@v6<br>upload-artifact@v7<br>codeql-init@v4<br>codeql-analyze@v4<br>golangci-action@v9 | golangci v2.11.3<br>golangci-action version latest | CodeQL manual build<br>CodeQL actions matrix<br>govulncheck observe |
| `artifact-handoff` | go 1.25.0 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4<br>codeql-init@v4<br>codeql-analyze@v4<br>buf-setup@v1 | golangci v2.11.3<br>govulncheck v1.1.4<br>buf v1.54.0<br>buf setup 1.54.0 | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |
| `node-artifact-runtime` | go 1.25.10 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4<br>codeql-init@v4<br>codeql-analyze@v4 | golangci v2.11.3<br>govulncheck v1.1.4 | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |
| `spawner` | go 1.25.10<br>toolchain go1.26.3 | checkout@v6<br>setup-go@v6<br>upload-artifact@v7<br>codeql-init@v4<br>codeql-analyze@v4 | golangci v2.11.3<br>govulncheck v1.1.4 | CodeQL manual build<br>CodeQL actions matrix |
| `tori` | go 1.25.5 | checkout@v6<br>setup-go@v6<br>upload-artifact@v7<br>codeql-init@v4<br>codeql-analyze@v4<br>buf-action@v1 | golangci v2.11.3<br>govulncheck v1.1.4 | CodeQL manual build<br>CodeQL actions matrix |
| `dag-go` | go 1.25.5 | checkout@v4, v6<br>setup-go@v5, v6<br>upload-artifact@v4<br>download-artifact@v4<br>codeql-init@v4<br>codeql-analyze@v4<br>benchmark-action@v1<br>gh-pages@v4 | golangci v2.11.3<br>govulncheck v1.1.4 | CodeQL manual build<br>CodeQL actions matrix<br>v6와 v4/v5 혼재 |

버전 감사 해석:

- CodeQL은 모든 대상 repo에서 `github/codeql-action/init@v4`와 `analyze@v4`를 쓴다.
- CodeQL의 Go 분석은 모든 대상 repo에서 `build-mode: manual`이며, actions 분석을 위한
  `build-mode: none` matrix도 함께 둔다.
- `checkout@v6`/`setup-go@v6`만 쓰는 repo와 v4/v5가 섞인 repo가 공존한다. 문서의
  `G008`, `G009`는 이 차이를 반영한다.
- `golangci-lint` 도구 버전은 대부분 `v2.11.3`으로 수렴하지만, `podbridge5`, `bori`,
  `NodePalette`는 GitHub Action에서 `version: latest` + `install-mode: goinstall`로
  실행한다.
- `NodeSentinel`은 `golangci/golangci-lint-action@v7`을 쓰지만 실제 lint 버전은
  workflow env의 `GOLANGCI_LINT_VERSION: v2.11.3`을 참조한다.
- `artifact-handoff`는 Buf CLI도 `v1.54.0`으로 명시한다.
- `spawner`는 `go 1.25.10`에 추가로 `toolchain go1.26.3`이 있어 Go toolchain 기준을
  둘 다 확인해야 한다.

## 10. 후보 승격 항목 적용 비교

이 장은 이전 문서의 “다음 조사 후보”였던 항목을 실제 G-ID로 승격한 뒤, 현재 대상
repo에 적용되는지를 따로 비교한 표다. 핵심은 네 가지다.

- golden/snapshot/baseline 갱신 정책: `G334-G337`
- smoke 실패 증거 artifact 충분성: `G338-G342`
- CodeQL query suite와 path ignore: `G343-G346`
- Action major version drift: `G347-G350`

주의할 점은 “artifact upload가 있다”와 “smoke 실패 증거가 충분하다”가 같은 뜻이
아니라는 것이다. 예를 들어 coverage report를 upload해도 smoke 실패 당시 pod log나
manifest가 없으면 `G339`, `G340`은 적용됐다고 보지 않는다.

| repo | 신규 적용 ID | 해석 |
|---|---|---|
| `NodeVault` | G343,G347 | CodeQL에서 `vendor/**`를 ignore하고, 주요 checkout/setup-go Action은 v6 계열로 정리됨 |
| `podbridge5` | G338,G341,G342,G343,G348,G349 | VM/runtime smoke 로그를 가져와 artifact로 보존하며, checkout/setup-go 및 artifact Action major drift가 남아 있음 |
| `JUMI` | G334,G335,G336,G338,G340,G342,G343,G348,G349 | `UPDATE_GOLDEN=1` 기반 golden 갱신 흐름과 quality guardrail 언급이 있고, golden fixture freeze와 smoke artifact upload가 있으며, Action major drift가 있음 |
| `bori` | G336,G337,G338,G339,G340,G341,G342,G343,G348,G349 | baseline/snapshot 비교와 갱신 흐름, kind/VM smoke 진단 artifact가 풍부하고, Action major drift가 있음 |
| `sori` | G343,G347 | CodeQL vendor ignore와 v6 계열 checkout/setup-go 사용이 확인됨 |
| `NodeSentinel` | G343,G348,G349,G350 | CodeQL vendor ignore가 있고 checkout/setup-go, artifact, golangci-lint Action major drift가 남아 있음 |
| `NodePalette` | G343,G347 | CodeQL vendor ignore와 v6 계열 checkout/setup-go 사용이 확인됨 |
| `artifact-handoff` | G343,G348,G349 | CodeQL vendor ignore가 있고 checkout/setup-go 및 artifact Action major drift가 남아 있음 |
| `node-artifact-runtime` | G343,G348,G349 | PID1 smoke는 termination JSON을 임시 검증하지만 보존 artifact로 남기지는 않으므로 smoke artifact ID는 적용하지 않고, checkout/setup-go 및 artifact Action major drift만 표시함 |
| `spawner` | G343,G347 | CodeQL vendor ignore와 v6 계열 checkout/setup-go 사용이 확인됨 |
| `tori` | G336,G343,G347 | snapshot/fixture freeze 성격의 테스트가 있고, CodeQL vendor ignore와 v6 계열 checkout/setup-go 사용이 확인됨 |
| `dag-go` | G343,G348,G349 | CodeQL vendor ignore가 있고 checkout/setup-go 및 artifact Action major drift가 남아 있음 |

현재 확인 범위에서 `G344-G346`은 대상 12개 repo에 적용된 것으로 보지 않는다. 모든 repo가
CodeQL config에서 `vendor/**` ignore는 두지만, `queries:`로
`security-extended`나 `security-and-quality` suite를 명시한 증거는 확인되지 않았다.

### 10.1 재검수 운영 세부 항목 적용 비교

이번 재검수에서 추가한 `G363-G367`은 “검사가 무엇을 하는가”보다 “검사가 CI에서 얼마나
운영 가능하게 실행되는가”를 보는 항목이다. 특히 self-hosted runner, 실패 시 artifact
보존, cache/tmp 격리는 나중에 CI 장애를 디버깅할 때 차이를 크게 만든다.

| repo | 신규 운영 세부 ID | 해석 |
|---|---|---|
| `NodeVault` | G363,G364,G365 | 주요 CI job이 self-hosted runner에서 실행되고, `go vet` 직접 gate와 실패 시 artifact 보존이 있음 |
| `podbridge5` | G363,G364,G365 | self-hosted runner 기반 CI/VM runtime workflow, build-tagged `go vet`, `if: always()` artifact 보존이 있음 |
| `JUMI` | G365,G367 / Local G364 | registry smoke artifact는 실패해도 보존하고, Makefile에서 repo-local Go cache/tmp를 사용함. `go vet`은 Makefile에는 있으나 정규 workflow 실행으로는 확인하지 않음 |
| `bori` | G363,G365,G366 | VM integration은 self-hosted runner를 쓰고, kind/VM smoke artifact는 실패해도 보존한다. golangci workflow에서 setup-go cache를 명시적으로 끔 |
| `sori` | G367 / Local G364 | Makefile 기반 test/coverage/security target이 repo-local Go cache/tmp를 사용함. `go vet`은 Makefile에 있으나 정규 workflow gate로는 확인하지 않음 |
| `NodeSentinel` | G364,G365 | build job에서 `go vet`을 직접 실행하고, coverage artifact는 실패 시에도 보존함 |
| `NodePalette` | G363,G364,G365 | self-hosted runner 기반 CI이며, `go vet` 직접 gate와 실패 시 coverage artifact 보존이 있음 |
| `artifact-handoff` | G367 / Local G364 | Makefile은 repo-local Go cache/tmp와 `go vet`을 제공하지만, 정규 workflow에서 vet 실행은 확인하지 않음 |
| `node-artifact-runtime` | G367 / Local G364 | Makefile은 repo-local Go cache/tmp와 `go vet`을 제공하지만, 정규 workflow에서 vet 실행은 확인하지 않음 |
| `spawner` | G367 / Local G364 | Makefile 기반 CI target이 repo-local Go cache/tmp를 사용함. `go vet`은 Makefile에 있으나 정규 workflow gate로는 확인하지 않음 |
| `tori` | Local G364 | `go vet ./...` target은 있으나 정규 workflow gate로는 확인하지 않음 |
| `dag-go` | 적용 보류 | `GOCACHE_DIR`와 `GOTMPDIR`는 정의되어 있으나 `GOENV := GOCACHE="$(GOCACHE)"`로 되어 있어 `G367` 적용은 확정하지 않음 |

### 10.2 11-13장 후보 승격 항목 적용 비교

이 표는 11-13장의 “주의사항/다음 조사 후보”를 실제 번호 체계로 승격한 뒤의 적용 현황이다.
여기서 “없음”은 해당 repo에 가치가 없다는 뜻이 아니라, 이번 로컬 파일 조사 범위에서
확인되지 않았다는 뜻이다.

| repo | 적용 ID | 해석 |
|---|---|---|
| `NodeVault` | G376,G377,G378 | Trivy/ToolScanRecord 계약과 ingestion test가 있고, KUBECONFIG/바이너리 미준비 시 명시 skip되는 통합/SLI 테스트가 있음 |
| `podbridge5` | G378 | overlay/socket capability가 없을 때 명시 skip되는 테스트가 있음 |
| `JUMI` | G375 | `pkg/provenance`의 observed artifact manifest와 lineage parsing test가 있음 |
| `bori` | 없음으로 확인 | 이번 후보군에서는 추가 적용 항목을 확인하지 못함 |
| `sori` | G378 | root 권한, fixture, registry credential, symlink capability 등 환경 조건에 따른 명시 skip이 많음 |
| `NodeSentinel` | G376,G377 | Trivy VulnerabilityReport 계약 문서와 `runL5b`/`parseTrivySummary` 구현 테스트가 있음 |
| `NodePalette` | 없음으로 확인 | 이번 후보군에서는 추가 적용 항목을 확인하지 못함 |
| `artifact-handoff` | 없음으로 확인 | 이번 후보군에서는 추가 적용 항목을 확인하지 못함 |
| `node-artifact-runtime` | G373,G375,G378 | release notes 문서가 있고, artifact manifest/provenance model 및 Linux-specific skip이 있음 |
| `spawner` | 없음으로 확인 | 이번 후보군에서는 추가 적용 항목을 확인하지 못함 |
| `tori` | G378,G379 | NAS/shared fixture/DB capability 조건부 skip과 historical retired baseline skip이 있음 |
| `dag-go` | G373,G378 | `CHANGELOG.md`와 release notes 문서가 있고, short mode에서 stress test를 명시 skip함 |

이번 조사에서 대상 12개 repo에는 `G371`, `G372`, `G374`, `G380`, `G381`, `G382`,
`G383`, `G384`, `G385`, `G386`, `G387`, `G388`, `G389`, `G390` 적용이 확인되지 않았다.
특히 `G382`와 `G385`는 GitHub 서버의 branch protection/ruleset 설정을 봐야 하므로
로컬 파일만으로는 확정할 수 없다. `G383-G390`은 현재 적용 repo가 없더라도, 향후
상향 평준화 기준으로 삼을 가치가 있어 번호를 미리 부여했다.

## 11. 조사상 중요한 주의점

최종 검수에서 각 repo의 `.github/workflows/*`, `.golangci.yml`, `Makefile`,
`buf.yaml`, `.kube-linter.yaml`을 다시 대조했다. 특히 다음 축은 문서와 실제 repo를
재확인해 보정했다.

- CodeQL workflow의 `schedule`과 `strategy.matrix`는 모든 대상 repo에 있으므로
  `G004`, `G192`를 repo별 목록에 반영했다.
- `actions/checkout@v6`/`actions/setup-go@v6`와 v4/v5가 섞인 repo는 `G008`과
  `G009`를 함께 표시했다.
- `needs:`가 실제로 있는 repo만 `G191`로 표시했다. 현재 확인 범위에서는
  `NodeVault`, `dag-go`만 해당한다.
- `NodePalette`의 `govulncheck`는 `continue-on-error: true`이므로 hard fail
  `G036`이 아니라 observe `G037`로 정정했다.
- `sori`, `spawner`, `dag-go`의 `paths-ignore`는 path filter `G006`으로 반영했다.
- `dag-go`의 threshold는 coverage threshold가 아니라 benchmark threshold이므로
  coverage threshold ID인 `G241`, `G242`는 적용 목록에서 제외했다.
- fuzz는 이번 최종 검수에서 별도 항목으로 추가했다. 현재 대상 repo에는 자체 fuzz
  guardrail 적용이 확인되지 않으므로 `G328-G333`은 아직 repo별 적용 ID에 넣지 않았다.
- 이전 문서의 “다음 조사 후보”였던 golden 갱신, smoke artifact 충분성, CodeQL query
  suite, Action version drift는 이번 업데이트에서 `G334-G350`으로 승격했다.
- 추가 전수 재검수에서 self-hosted runner, direct `go vet`, 실패 시 artifact 보존,
  setup-go cache 비활성화, Go cache/tmp 격리, test double/library 사용을 `G363-G370`으로
  승격했다.
- 11-13장 재검토에서 CODEOWNERS, dependency update automation, release note/changelog,
  provenance manifest, Trivy/VulnerabilityReport contract, 조건부 skip, image scan/signing,
  SBOM/SLSA, branch protection required checks를 `G371-G382`로 승격했다.
- 대상 repo에는 아직 명확히 없지만 운영 표준으로 필요하다고 판단한 skip 만료/이슈 추적,
  quarantine registry, branch protection 감사 artifact, release readiness checklist,
  SARIF filter justification, generated-code review policy, security alert SLA,
  CI failure triage runbook도 `G383-G390`으로 승격했다.
- SLI 관점에서는 대표 가드레일 ID와 repo별 수치 SLI를 분리했다. `G281-G327`은
  kube-slint/slint-gate 사용 방식이고, `reconcile_fast_delta`, `jumi_jobs_created_smoke`,
  `bori_workqueue_depth_end` 같은 개별 SLI는 2.23.1 하위 목록에 남겼다. 개별 SLI는 repo
  도메인에 묶이므로 전역 G-ID로 만들지 않는다.
- 내가 제안한 repo별 추가 SLI는 2.23.2에 따로 둔다. 이 항목들은 “현재 적용”이 아니라
  “검토 후보”다. 따라서 repo별 적용 ID 목록에는 넣지 않고, 나중에 사용자가 선택한 항목만
  실제 policy/spec/test로 승격해야 한다.
- churn SLI는 2.23.3과 2.23.4로 분리했다. 2.23.3은 Kubernetes/operator object churn
  후보(`KOC-001`~`KOC-032`), 2.23.4는 데이터 플레인 app churn 후보(`DPC-001`~`DPC-060`)다.
  둘 다 현재 적용 여부가 아니라 “향후 SLI로 검토할 후보”이며, 적용하려면 각 repo의
  metrics/spec/policy/test에 실제로 연결해야 한다.
- Kubernetes Operator 표준 테스트/가드레일 후보는 `G391-G440`으로 승격했다. 기존
  `G351-G362`가 bori에서 실제 확인된 operator behavior test라면, `G391-G440`은 아직
  적용되지 않았더라도 operator repo에 필요할 수 있는 표준 검토 단위다.
- 컨테이너 데이터 플레인/PID1/process supervision 테스트 후보는 `G441-G460`으로 승격했다.
  이는 `DPC-050`~`DPC-060` 같은 process churn SLI 후보와 연결될 수 있지만, SLI가 아니라
  실제 테스트/가드레일 단위다.
- `dag-go`는 `GOCACHE_DIR := /tmp/dag-go-gocache`와 `GOTMPDIR := /tmp/dag-go-gotmp`를
  정의하지만 `GOENV := GOCACHE="$(GOCACHE)" GOTMPDIR="$(GOTMPDIR)"` 형태라, 의도한
  `GOCACHE_DIR`가 실제로 적용되는지 불확실하다. 그래서 `G367`은 적용 보류로 표시했다.

1. `golangci-lint`는 repo마다 내부 린터 구성이 다르다.
   “린트 있음”만으로는 비교가 안 된다. 반드시 G057-G088 내부 린터 ID를 같이 봐야 한다.

2. `depguard`는 단순 린트가 아니라 architecture boundary다.
   `sori`, `dag-go`, `artifact-handoff`, `node-artifact-runtime`, `spawner`, `tori`,
   `JUMI`에서 각각 다른 경계를 강제한다.

3. `govulncheck`는 hard gate와 observe mode가 섞여 있다.
   G036이면 실패 gate, G037이면 관찰 성격이다.

4. 실패 경로 테스트 G137은 workflow/Makefile 수준만으로는 정확히 식별하기 어렵다.
   다음 조사에서는 test function 이름과 assertion을 봐야 한다.

5. Go 저장소 CodeQL은 현재 G018-G028 흐름이 기본이다.
   NodeKit식 raw SARIF filter/manual upload(G029-G030)는 현재 Go repo 기본으로 쓰지 않는다.

6. `kube-slint`는 repo 비교 행이 아니라 G281-G327 가드레일 family다.
   다른 repo에 “kube-slint를 적용한다”는 말은 어느 수준인지 반드시 ID로 나눠야 한다.
   예를 들어 단순히 `slint-gate` CLI만 쓰는 것은 G289이고, 실제 SLI 측정 harness까지
   붙인 것은 G281이다. `slint-gate` 없이 Go test가 직접 SLI 값을 assert하면 G327이고,
   coverage governance까지 쓰면 G294가 추가된다.

7. 실패 경로 테스트는 이제 G306-G325로 세분화했다.
   다만 이는 테스트 함수명과 fixture 이름 중심의 1차 분류다. 정확한 assertion 의미까지
   보려면 각 테스트 본문을 추가로 읽어야 한다.

## 12. 조사 당시 local 변경 주의사항

다음 변경 사항은 이 조사 문서 작성 대상이 아니며 수정하지 않았다.

- `NodeVault`: untracked `test/slint/assets/`
- `artifact-handoff`: modified `go.sum`, untracked `deploy/devspace/`
- `tori`: modified docs/Makefile/README files, untracked product readiness doc
- `dag-go`: modified `Makefile`, `go.sum`

## 13. 다음 조사에서 추가로 번호화할 후보

이번 업데이트에서 기존 후보였던 golden 갱신 정책, smoke artifact 충분성, CodeQL query
suite, Action version drift는 `G334-G350`으로 승격했다. 이후 재검수에서 CI 운영 세부와
테스트 더블은 `G363-G370`, 11-13장 후보는 `G371-G390`, Kubernetes Operator 표준
테스트/가드레일 후보는 `G391-G440`, 컨테이너 데이터 플레인/PID1/process supervision
후보는 `G441-G460`으로 승격했다.

이번 버전에서는 “현재 repo에 명확히 적용되어 있지 않더라도, 표준 지침으로 추적할 가치가
있는 항목”도 번호로 남겼다. 그래서 아래 항목은 더 이상 무번호 후보가 아니라, 적용 여부가
`현재 없음`인 정식 ID로 관리한다.

- GitHub branch protection/ruleset 실제 상태: required checks는 `G382`, 해당 설정을
  감사 artifact로 남기는 것은 `G385`다. 둘 다 로컬 파일만으로는 확정할 수 없고 GitHub
  repository setting 조회가 필요하다.
- CODEOWNERS/dependency automation의 적용 필요성 판단: 파일 존재 여부는 각각 `G371`,
  `G372`로 추적한다. 현재 대상 12개 repo에는 확인되지 않았으므로, 어떤 repo부터 적용할지
  별도 정책 결정이 필요하다.
- SBOM/SLSA/Cosign/Trivy image scan의 CI hard gate 여부: image scan/signing은 `G380`,
  SBOM/SLSA/provenance artifact는 `G381`로 추적한다. 현재 대상 repo에는 명확한 CI 적용
  근거가 없다.
- release note guardrail의 강제 방식: release note 파일 존재는 `G373`, CI/script 강제는
  `G374`, 릴리스 전 사람이 확인할 readiness checklist는 `G386`으로 나눠서 본다.
- conditional skip 관리 정책: skip 존재는 `G378`, historical/quarantine marker는 `G379`,
  issue/owner/expiry date까지 연결된 관리는 `G383`, 별도 quarantine registry는 `G384`로
  나눠서 추적한다.
- CodeQL/SARIF 예외 정책: 수동 SARIF upload/filter 자체는 `G029-G030`, 그 필터가 제품
  source alert를 숨기지 않는다는 근거 문서화는 `G387`로 본다.
- generated code 운영 정책: protobuf/generated drift 검사는 `G151`, `G157`, `G260`에
  이미 있지만, generated code를 리뷰/필터/소유권 관점에서 어떻게 다룰지 정한 정책은
  `G388`로 별도 추적한다.
- 운영 대응 문서: 보안 알림 triage SLA는 `G389`, CI 실패 조사 runbook은 `G390`으로
  추적한다. 둘 다 현재 대상 12개 repo에는 명확한 적용 근거가 없다.

## 14. 번호 평탄화 점검

이번 점검의 결론은 “모든 번호를 억지로 같은 크기로 만들지 않는다”이다. 품질 기준서는
작업 지시에 쓸 수 있어야 하지만, 동시에 사람이 전체 구조를 이해할 수 있어야 한다.
따라서 넓은 family 번호와 좁은 세부 번호를 모두 유지하되, 각 번호가 어떤 수준인지
명확히 읽을 수 있게 한다.

### 14.1 유지할 넓은 family 번호

아래 번호는 일부러 넓게 유지한다. 이 번호들은 repo의 테스트/가드레일 성격을 빠르게
분류하기 위한 상위 축이다.

| ID | 넓게 유지하는 이유 | 같이 봐야 하는 세부 번호 |
|---|---|---|
| G136 골든 패스 단위 테스트 | 정상 경로 테스트 전체를 빠르게 표시하는 family | 구체 test file/function, coverage 정보 |
| G137 실패 경로 테스트 | 실패 유형 전체를 포괄하는 family | G306-G325 |
| G140 계약 테스트 | API/proto/K8s/cross-repo 계약을 한 번에 묶는 family | G145-G162, G320, repo별 contract test |
| G141 스모크 테스트 | boot/CLI/runtime/kind smoke를 묶는 family | G159-G170, G338-G342 |
| G142 통합 테스트 | 여러 구성요소가 함께 도는 큰 분류 | VM/kind/registry/workflow별 세부 ID |
| G143 race/lifecycle 테스트 | 동시성, cancel, cleanup, leak 성격을 묶는 family | G122-G125, G323, G441-G459 |
| G171 kube-slint SLI gate | SLI 기반 gate라는 큰 축 | G281-G327, KOC-*, DPC-* |
| G281 kube-slint SLI 측정 harness | kube-slint를 측정 harness로 쓰는 큰 축 | G282-G287, repo별 SLI 목록 |

이 번호들은 “없애거나 쪼개야 하는 번호”가 아니다. 다만 이 번호만으로는 충분하지 않은
경우가 많다. 예를 들어 `G137`만 보고 “실패 경로가 충분하다”고 판단하면 안 된다.
실제로는 timeout, cancellation, path escape, retry exhaustion, cleanup failure 중
무엇을 막는지 봐야 한다.

### 14.2 세부 번호로 유지할 항목

아래 항목들은 작아 보여도 별도 번호로 유지한다. 실제 운영에서는 이 작은 차이가 적용
강도와 디버깅 가능성을 바꾸기 때문이다.

| ID 범위 | 유지 이유 |
|---|---|
| G018-G030 CodeQL 세부 | workflow 존재, build mode, SARIF upload/filter는 보안 분석 결과가 달라지는 별도 결정이다. |
| G031-G038 govulncheck 세부 | 설치, 실행, JSON report, hard fail/observe가 모두 운영 의미가 다르다. |
| G044-G056 golangci 실행/설정 | config 존재와 CI 실행, checksum install, generated exclusion은 같은 “lint”가 아니다. |
| G057-G088 golangci 내부 린터 | repo별 lint 강도를 비교하려면 어떤 linter가 켜졌는지까지 봐야 한다. |
| G119-G135 테스트 실행/coverage | test 실행, race, shuffle, coverage profile, threshold는 각각 다른 위험을 막는다. |
| G153-G162 Kubernetes manifest/cluster | kube-linter, kubeconform, kind smoke는 실패 지점이 다르므로 합치지 않는다. |
| G334-G350 artifact/version drift | artifact를 남기는 것과 실패 시에도 남기는 것, Action major drift는 운영 차이가 크다. |
| G441-G460 PID1/process supervision | container data plane에서는 signal, zombie, process group, stdout drain이 각각 별도 장애 원인이다. |

### 14.3 중복처럼 보이지만 의도적으로 분리한 항목

아래 항목들은 이름만 보면 비슷하지만 같은 번호로 합치지 않는다.

| 항목 | 왜 분리하는가 |
|---|---|
| G008 latest checkout/setup-go vs G347 checkout/setup-go v6-only | G008은 최신 계열 사용 여부, G347은 핵심 workflow가 v6-only로 정리됐는지 보는 drift 감사 항목이다. |
| G009 older checkout/setup-go vs G348 mixed checkout/setup-go major versions | G009는 구버전 존재, G348은 신구 major 혼재라는 운영 drift를 본다. |
| G154 kube-linter action vs G155 kube-linter CLI vs G262 kube-linter action | Action으로 돌리는지, CLI로 돌리는지, 특정 action major를 쓰는지는 적용/재현 방식이 다르다. |
| G133 coverage threshold vs G241/G242 coverage threshold 세부 | G133은 큰 coverage threshold family이고, G241/G242는 repo-specific threshold 구현을 더 자세히 본다. |
| G171 kube-slint SLI gate vs G289 slint-gate CLI vs G291 slint-gate workflow gate | SLI gate라는 개념, CLI 사용, GitHub workflow gate는 적용 강도가 다르다. |
| G378 conditional skip vs G383 skip expiry/issue tracking | skip이 있다는 것과 skip을 만료/이슈로 관리한다는 것은 운영 성숙도가 다르다. |
| G380 image scan/signing vs G376/G377 Trivy/VulnerabilityReport contract/test | 이미지 자체를 CI에서 scan/sign하는 것과, trivy-operator 결과를 ingest/test하는 것은 다른 계층이다. |

### 14.4 평탄화가 필요한 후보

아래 항목들은 다음 버전에서 더 읽기 쉽게 다듬을 수 있다. 단, 이번 문서에서는 기존 ID
의미를 바꾸지 않고 해석 규칙으로 해결한다.

| 후보 | 현재 상태 | 권장 처리 |
|---|---|---|
| G136-G144 테스트 성격 | 넓은 family 항목 | 유지하되 repo별 상세 비교에서는 G306-G325, G351-G362, G441-G460과 함께 표시한다. |
| G153-G162와 G258-G266 일부 Kubernetes 세부 | 일부 큰 항목과 설치/보존 세부가 섞여 있음 | 다음 문서에서 “검증 종류”와 “도구 설치/운영 방식”을 표 안에서 명시적으로 나눌 수 있다. |
| G281-G327 kube-slint 계열 | 제품 기능, consumer 적용, gate, UX guardrail이 한 family 안에 있음 | 유지하되 `측정`, `정책 평가`, `artifact`, `coverage governance`, `제품 자체 guardrail`로 묶어 읽는다. |
| G391-G440 operator 후보 | 아직 적용 근거가 아니라 표준 후보 | repo에 실제 적용할 때 envtest/CRD/RBAC/status/finalizer/SLI guardrail 순으로 우선순위를 정한다. |
| KOC/DPC 하위 번호 | G-ID가 아니라 SLI 후보 번호 | 실제 repo에 채택되면 repo별 SLI spec/policy 이름으로 구체화하고, 필요할 때만 G-ID와 연결한다. |

### 14.5 앞으로 번호를 추가할 때의 규칙

새 번호를 추가할 때는 아래 질문을 먼저 통과해야 한다.

1. 이 항목은 repo에 “있다/없다”를 객관적으로 판단할 수 있는가?
2. 기존 번호와 실패 원인, 실행 위치, 강제력, artifact 의미가 다른가?
3. 너무 repo-specific한 metric 이름인가? 그렇다면 G-ID가 아니라 KOC/DPC 또는 repo별 SLI 하위 목록으로 둔다.
4. 너무 넓은가? 그렇다면 family ID로 둘지, 세부 ID를 같이 추가할지 결정한다.
5. 사람이 “이 번호를 추가해”라고 말했을 때 무엇을 구현해야 하는지 분명한가?

이 규칙을 따르면 번호가 계속 늘어나도 문서가 단순 목록으로 무너지지 않고, 실제 적용
지침서로 유지될 수 있다.
