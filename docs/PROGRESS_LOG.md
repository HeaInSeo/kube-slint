# kube-slint Project Progress Log

This file tracks the incremental stages of kube-slint work.
Update this file at the **start and end** of every stage/task.

---

## Current Status: Stage (Post-RC) — Phase 6-d Go CLI Migration Complete

**Branch:** `main`
**Last updated:** 2026-04-30 (Go CLI migration + README full rewrite)

---

### Phase 6-d: Go CLI Migration (COMPLETE)

**완료 항목:**

* [x] `hack/slint_gate.py` (Python + pyyaml) → `cmd/slint-gate` Go 바이너리로 완전 대체
* [x] `internal/gate` 패키지 구현 (threshold check, regression detection, reliability check, gate result 계산)
* [x] `internal/gate` 단위 테스트 16종, 89.2% coverage
* [x] `Makefile` `slint-gate` 타깃 추가 (`bin/slint-gate` 빌드)
* [x] `.github/workflows/slint-gate.yml`: Python 의존 제거, Go + jq 전환
* [x] `--github-step-summary` 플래그로 Actions step summary 렌더링 Go 내부 흡수
* [x] `README.md` 전면 재작성 (영문, Quick Start / 플로우 다이어그램 / 상세 사용법 포함)
* [x] `README(Kor).md` 전면 재작성 (한국어, 동일 구조 미러링)
* [x] `golangci-lint v2.1.0` — 0 issues
* [x] `DECISIONS.md` D-012 결정 추가
* [x] GitHub push 완료 (커밋 3개: feat, ci, docs)

**근거:** Python 런타임/pyyaml 의존 제거 + Go 단일 스택 통합. 내부 게이트 로직을 테스트 가능한 `internal/gate` 패키지로 캡슐화하여 회귀 방어.

---

## Current Status: Stage (RC Approved) — Phase 6-c Regression Gate Model

**Branch:** `main`
**Last updated:** 2026-03-20 (RC baseline approval)

### Current Focus

* kube-slint 정체성은 이미 **operator 개발 단계에서 operational SLI를 lint-style로 적용하는 shift-left quality guardrail**로 정렬되어 있음.
* 현재 기준선에서 확정된 사실:
  1. `docs/DECISIONS.md`와 `docs/project-status.yaml`가 현재 자동화/상태의 상위 기준선이다.
  2. `slint-gate`와 `roadmap-status`는 현재 CI 가시성 기준선이다.
  3. `hello-operator`는 canonical consumer validation fixture로 연결되어 있다.
  4. `hello-operator`에서는 `kube-slint` 상향 후 `snapshotFetcher` 수동 워크어라운드 제거, `../kube-slint` 직접 의존 제거, `artifacts/` 경로 정렬, `E2E_SLI=1` 재검증이 완료되었다.
  5. `.slint/policy.yaml` + `docs/baselines/hello-operator-sli-summary.json` + `hello-operator/artifacts/sli-summary.json` 조합으로 `slint-gate` smoke가 `PASS`를 반환했다.
* 이번 RC 기준선의 확정 구성 요소는 `.slint/policy.yaml`, `docs/baselines/hello-operator-sli-summary.json`, 그리고 `hello-operator` canonical consumer fixture 경로(`Tiltfile`, `hack/run-slint-gate.sh`, `test/e2e/TestHelloSLIE2E`)다.
* `hello-operator/.slint/policy.yaml`는 local first-run/dev policy이고, RC baseline contract와 경쟁하지 않는다. RC 판단은 계속 `kube-slint` 저장소의 policy + baseline pair를 기준으로 유지한다.
* 현재 남은 실행/fixture 부채는 `hello-sample-create` 기본 fixture 의미와 local `pyyaml` 의존 자체다.
* post-RC improvement는 artifact-backed baseline flow, summary schema 확장 후보, baseline update path의 추가 workflow 보조, 오래된 prose history 정리다.
* 추적 리스크는 `hello-operator` E2E 경로의 `PreFetch/Start()` semantics 의존이다.

### RC Approval

1. **regression gate model 최소 완료 기준**
   - RC 승인 기준은 현재 문서/정책/스모크 증거로 충족되었다:
   - `slint-gate` workflow가 현재 summary/policy 입력으로 실행 가능할 것
   - 현재 summary/policy 기준에서 `FAIL/WARN/NO_GRADE` 해석 경로가 문서와 workflow에서 일치할 것
   - repository-stored baseline source는 `docs/baselines/hello-operator-sli-summary.json`로 고정할 것
   - RC 기준 policy 파일은 `.slint/policy.yaml`로 고정할 것
   - baseline update 경로는 일반 PR과 분리된 승인 기반 변경으로만 허용할 것
   - 현재 smoke 결과는 `PASS`이며, RC 결정은 이 baseline contract를 기준으로 승인되었다.

2. **canonical consumer fixture 재현 체크 항목**
   - RC 기준 재현 체크 항목은 아래로 고정한다:
   - `hello-operator`가 `kube-slint` `4d3867ccc6ba` 기준선으로 고정되어 있을 것
   - `go test ./test/e2e -run TestHelloSLIE2E -tags e2e -count=1` 이 통과할 것
   - `E2E_SLI=1 go test ./test/e2e -run TestHelloSLIE2E -tags e2e -count=1 -timeout 45s` 가 통과할 것
   - `Tiltfile`, `hack/run-slint-gate.sh`, `.slint/policy.yaml` 조합이 현재 fixture 운영 경로로 유지될 것

3. **deferred debt / post-RC improvement 분류**
   - deferred debt:
   - `hello-sample-create`: fixture용 고정 케이스이며 현재 canonical consumer 증거 경로를 깨지 않으므로 non-blocking.
   - local `pyyaml`: bridge 스크립트 실행 의존성일 뿐 제품/consumer baseline 실증을 무효화하지 않으므로 non-blocking.
   - post-RC improvement:
   - `reconcile_error_ratio` 같은 summary schema 외부 metric
   - baseline update path 추가 자동화와 artifact-backed baseline flow
   - 오래된 prose history와 현재 기준선 표현의 추가 정리

4. **추적 리스크**
   - `hello-operator`의 실제 `E2E_SLI=1` 경로가 통과했고, 수동 `snapshotFetcher` 제거도 완료되었다.
   - 따라서 `PreFetch/Start()` semantics 의존은 현재 **수용 가능한 운영 리스크**로 유지한다.
   - 단, 이 경로는 `session.Start()`가 workload 변경 전에 호출된다는 계약에 의존하므로 post-RC 추적 리스크로 계속 남긴다.

### Regression Baseline Lifecycle

- **생성**: canonical consumer fixture(`hello-operator`)의 승인된 `E2E_SLI=1` summary를 repository-stored baseline file `docs/baselines/hello-operator-sli-summary.json`로 반영한다.
- **사용**: RC 기준 regression comparison은 `.slint/policy.yaml`과 `docs/baselines/hello-operator-sli-summary.json` 조합을 기본 입력으로 사용한다.
- **부재 시 판정**: first-run 또는 baseline 미지정은 `WARN` 기본값으로 취급하되, RC 결정 시점에는 baseline file이 존재해야 한다.
- **손상 시 판정**: baseline file이 unreadable/corrupt면 regression 축은 `NO_GRADE`로 간주하며, RC 결정에는 불충분한 상태로 본다.
- **갱신 승인 경로**: baseline update는 일반 PR 변경과 분리된 승인 기반 변경으로만 허용한다. 변경 이유와 비교 근거를 함께 남겨야 한다.
- **승인 보조 helper**: `hack/prepare-baseline-update.sh /path/to/sli-summary.json` 는 baseline candidate 복사본과 normalized diff를 준비하지만, repository baseline을 자동 교체하지는 않는다.
- **운영 진입점**: `make baseline-update-prepare BASELINE_SUMMARY=/path/to/sli-summary.json` 로 helper 실행 경로를 표준화한다.
- **artifact-backed 시작점**: artifact summary는 repository baseline을 대체하지 않고, baseline update review용 candidate input source로만 사용한다.
- **RC metric set**: 현재 RC 기준 regression policy는 summary에 실제 존재하는 `reconcile_total_delta`, `workqueue_depth_end`만 사용한다.
- **post-RC 확장 후보**: `reconcile_error_ratio` 같은 summary schema 외부 metric은 이번 RC 기준에서는 제외하고, schema/measurement 확장 과제로 분리한다.

### Definition of Done (DoD)

* [x] Stage 상태를 `Phase 6-b Shift-left Guardrail Alignment`로 전환
* [x] 정체성/계약/모드/회귀게이트/소비자 기준 저장소에 대한 Decision Log 신설
* [x] Phase 6-b ~ Phase 7-a + Release Gate(guardrail RC) 로드맵 초안 반영
* [x] GitHub Actions 계획 메모(`slint-gate`, `roadmap-status`, `baseline-update`) 문서화
* [x] README 후속 수정 포인트를 notes로 기록 (코드 변경 없음)
* [x] `slint-gate` 입력/출력 계약 및 regression gate 판정 초안 문서화 (`docs/notes/slint-gate-spec-2026-03-07.md`)
* [x] policy 파일 경로/최소 스키마 + `slint-gate-summary.json` 최소 출력 계약 초안 문서화 (`docs/notes/slint-gate-io-contract-2026-03-07.md`)

### Next command to run

* `gh workflow list` (현재 CI 워크플로우 인벤토리 확인)
* `gh workflow view roadmap-status` (status visibility workflow 기준 확인)
* `cat docs/notes/slint-gate-io-contract-2026-03-07.md` (Phase 6-c 구현 입력/출력 계약 고정본 검토)

### If blocked, fallback check

* `docs/DECISIONS.md`와 `docs/notes/phase-6b-guardrail-alignment-2026-03-07.md`를 기준 계약으로 우선 유지하고, 구현 단계는 Phase 6-c 이후로 분리

---

## Completed Items

### Stage 7 — Implementation & Stabilization

* 기초 하네스 구현 및 안정화 완료
* GitHub Actions lint/test 통과 상태 확보 완료

### Stage T-2 — Harness Test Reinforcement 2nd

* Cleanup mode matrix 테스트 보강 완료
* CheckGating 테스트 보강 완료
* preset/default specs smoke 테스트 보강 완료

### Stage Audit & UX/Docs Reinforcement (Post-T-2)

* (Audit) 계측 실패 격리, E2E Hook 기반 등의 핵심 철학 정렬 확인 완료
* (Docs) `sli-summary.json` 결과 해석 가이드 보강 완료
* (Docs) Custom SLI 튜토리얼(`SessionConfig.Specs`) 안내 완료
* (Docs) Artifact 존재 가정 금지(IO 실패 격리) 경고 문서화 완료
* (Docs) 초보자 가독성을 위한 상태 계층(Status Layers) 표 도입 및 JSON 예시 추가 조치 완료
* (Docs) 마감용 리터치를 통한 7.3/7.4 상태 표현 계층 및 JSON 해석 문장의 용어 정밀화 완료

### Stage E2E Final Verification

* (Verification) `test/e2e` 매니저 컨트롤러 구버전 테스트 코드 발견 및 무시(repository가 library로 전환된 철학에 맞지 않음). `test/e2e/harness`의 시뮬레이터 및 Go JSON 정합성 테스트로 Fallback 수행.
* (Verification) Gating/Strictness 실패 시 `harness.Attach` 에러 전파 흡수 여부 확인(테스트 실패시키지 않고 GinkgoWriter에 로그 남김 -> "테스트!=측정실패" 철학 준수).
* (Docs Patch) 섹션 6.3에 `Attach` 훅의 로그-only 에러 삼킴 규칙을 소규모 명시 패치하여 Artifact 부재 경고 타당성 최종 확인.

### Stage Phase A/B (T-3 SanitizeFilename 보강)

* (Phase A) 문서 v1.2 가이드 7.4항 "Partial" 조건 설명 시 평가 스킵이 아닌 보조 지표 누락 가능성을 명확히 분리 서술.
* (Phase A) PROGRESS_LOG 릴리즈 항목 중복 제거 및 구버전 (Current) 꼬리표 정리 완료.
* (Phase B) `test/e2e/harness/sanitize_test.go` 파일 구축. 빈 문자열(`""` -> `"unknown"`), 공백정리(`"  "` -> `"unknown"`), 경로구분자, 특수문자 치환 등 파일시스템 보호를 위한 10종 엣지케이스 Table-driven 테스트로 방어력 증명 완료 (기존 함수 수정 없이 통과).

### Stage Cleanup Audit & Diagnostics

* 저장소 구조/테스트 신뢰성에 대한 진단(Audit) 실시 및 `docs/notes/cleanup-audit-report-2026-02-27.md` 제출.
* 발견 사항 요약: 루트 디렉토리의 임시 로그(`.log`, `e2e.test`) 방치, `test/e2e` 폴더 내의 Dummy Controller 배포 코드가 더 이상 유효하지 않은 Legacy 상태(Broken E2E), `pkg/kubeutil`의 YAML Sprintf 하드코딩 부채(`TODO(security)`), 그리고 `test/e2e/harness/session.go` 내의 Fetcher Adapter 결합 관찰.

### Stage Cleanup Execution (Phases 1 & 3)

* (Phase 1) 루트 및 각종 디렉토리에 산재되어 있던 방치 파일(`TODO.md`, `code_review.md`, `test_full_v*.log`, `cover.out`, `e2e.test` 등) 삭제 및 Git Tracked 로그 파일(`lint.log` 등)을 `git rm` 명령으로 저장소 인덱스에서 정리함. 
* (Phase 3) Library화로 인해 동작하지 않는 `test/e2e` 하위 레거시 테스트(`e2e_test.go`, `e2e_suite_test.go`)들에 `//go:build legacy_e2e` 빌드 태그를 부여하여 표준 `go test ./...` 및 CI 범위에서 격리(Quarantine) 처리함. 
* (Phase 3) `test/e2e/README.md`를 생성하여 해당 E2E 테스트가 제외된 이력을 명시하고, 파일 경로를 정확히 `test/e2e/...` 하위로 정정함. `Makefile` `test` 커맨드는 `grep -v /e2e` 방식 대신 기본 동작으로 정상화.
### Stage Consistency Patch

* (Correction) 이전에 지워지지 않고 Git에 임시로 Tracked되어 남아 있던 `lint.log`, `test_full.log` 등 4개 파일을 `git rm`하여 증거 기반으로 제거함.
* (Correction) `test/e2e/README.md` 내에 기재된 `e2e_test.go`의 경로 누락(`test/e2e_test.go` -> `test/e2e/e2e_test.go`)을 실제 파일 시스템 구조와 맞게 정합성 수립.
* (Correction) `PROGRESS_LOG.md` 내의 "100%", "영구 제거", "Ready for Release"와 같은 과장 표현 및 릴리즈 독단 판정 문구를 모두 객관적("격리", "정리", "상태 갱신")인 표현으로 배제함.

### Stage Cleanup Execution Phase 1-lite & 3-prep (Policy First)

* 명백한 잔해로 판별된 최상위 `cover.out` 등의 물리적 흔적 삭제 불가 여부 확인 및 Gitignore(`bin/`) 통제 추가 조치 (Read-only 기조 유지).
* 애매한 항목 삭제 대신 처리 정책 결정을 위해 `docs/notes/cleanup-policy-decision-input-2026-02-28.md` 문서 도출 (`presets/`, `scripts/check-slo-metrics.sh` 정책 비교 및 삭제/이관 추천안). 과감한 삭제 전 사용자 결정 요쳥.
* 소비자 단위로써의 테스트를 재건하기 위한 아키텍처 초안(`docs/notes/e2e-modernization-prep-2026-02-28.md`) 수립, Mock Server 기반의 Harness Integration Test 전략 선제안 (대규모 삭제 전초 작업).

### Stage A — Policy Checkpoint gates (Stop-and-Report)

* 증거 확보 전 삭제 금지 기조에 따라, `cleanup-policy-decision-input-2026-02-28.md` 내에 기재된 조건부 삭제 조항(`Delete (Conditional)`)을 단순히 '문서 예제 존재 확인'에서 **'Phase 4-a 소비자 검증 자산 성공 확보'**라는 구체적이고 물리적인 Execution Gate로 치환함.
* `pkg/` 변경 금지 및 `test/consumer-onboarding/` 산출물 배치 준수 가이드라인 등을 공식화하여 문서 간 정합성을 일치시킴.
* Stage B 시작 시점에 정책 체크박스 문구를 정밀화("Approve conditional delete policy (JSON examples + Phase 4-a success evidence)?")하는 Preflight 반영 완료.

### Stage B — Phase 4-a: Consumer Onboarding Probe (Go import)

* `test/consumer-onboarding/kubebuilder-default-sli/` 하위에 최소화된 빈 깡통 Reconciler 기반 샘플 구축.
* `envtest`를 사용해 테스트 클러스터 메모리에 매니저를 띄우고 `kube-slint` Harness `NewSession` -> `Start()` -> `End()` 사이클 호출 확인 (PASS 증거 획득).
* **관찰 결과 (4분류 분석)**:
  1. **문서 UX 문제**: `harness` API 사용 시 필수 설정(`Namespace`, `MetricsServiceName` 등)이 무엇인지 컴파일러 레벨에서 직관적이지 않음 (추후 가이드라인 보강 필요 증거).
  2. **API/인터페이스 문제**: 소비자 입장에서 `spec.PromMetric()` 보다 `spec.UnsafePromKey()`를 써야 하는 등 Spec 선언 과정의 구조체가 모호함.
  3. **테스트 자산 배치/구조 문제**: `setup-envtest` 바이너리(`test-operator/bin/k8s`)가 상위 폴더에 의존하여 Consumer 측 복사(cp)가 필요했음 (단독 실행 배포 시 약점). 
  4. **로깅/디버깅 문제**: `Session.Start()` 실행 시 Endpoint 스크랩 실패 등은 `kube-slint [discovery]:` 등 유의미한 표준 출력 정보가 다수 발생하어 쉘 스크립트(`check-slo-metrics.sh`) 없이도 로깅 수준이 충분함을 교차 검증함.

### Stage C — 정책 삭제 조건 재평가 (Evidence-based Judgment)

* Stage B의 결과를 파악하여 `cleanup-policy-decision-input-2026-02-28.md`의 조건부 삭제 조항 달성 여부를 판정함 (물리 삭제 절대 금지 원칙 준수).
* **`presets/` 판정**: Stage B 통합 테스트에서 패키지 없이 순수 JSON-string 형태로 정상 작동함을 증명. 조건 충족(Condition Met).
* **`scripts/check-slo-metrics.sh` 판정**: Stage B 구동 시 자동화된 파이프라인(Session Engine)이 뿜어내는 수많은 scrape 에러/로그가 디버깅에 충분하다고 판단됨. Phase 4-a / 4-b 필수 조건은 OR 조건(하나면 충분)으로 해석됨. 조건 충족(Condition Met via Phase 4-a).
* **Stage D와의 연결**: Stage B는 "라이브러리를 임포트하는 Go 소비자"의 입장을 대변함. 쿠버네티스 환경에 인프라(Kustomize Base/Overlays 등)를 심는 "운영/배포 소비자"의 입장은 별개의 검증이 필요함. 따라서 `check-slo-metrics.sh`의 삭제 근거는 확보되었으나, Kustomize Consumer UX를 다루는 Stage D(Phase 4-b)는 인프라 프로비저닝 구조 정합성 확인을 위해 독립적으로 수행되어야 함.

### Stage D — Phase 4-b: Kustomize Consumer UX Probe (Remote Resource)

* Kustomize 환경에서 `kube-slint` 인프라를 소비하는 외부 오퍼레이터의 UX 검증을 위해 `test/consumer-onboarding/kustomize-remote-consumer` 자산을 구축.
* 테스트 경로: `github.com/HeaInSeo/kube-slint//config/default?ref=0f48f...` 및 `//config/samples/prometheus?ref=0f48f...`
* **관찰 결과 (4분류 분석)**:
  1. **문서 UX 문제**: `README.md`는 Remote 핀 고정의 중요성을 잘 명시하나, `config/default`가 빈 껍데기임을 은연중에 인정하며 "로컬 복사 후 변형"을 권유함. 이는 원격 Kustomize 수입을 사실상 사용 불능하게 만드는 모순된 지시사항임.
  2. **Kustomize 경로/참조(ref pinning) 문제**: 문법적인 Kustomize Remote Fetch(`//`와 `?ref=`)는 정상 동작함. 툴링/경로상의 블로커는 없었음.
  3. **배치/구조 문제**: Standalone 파편이 남아있어, 실 사용(`config/samples/prometheus`) 시 리소스의 `matchLabels`가 라이브러리를 쓰는 타겟 Operator가 아니라 `kube-slint` 이름으로 하드코딩되어 있음. 유동적인 `nameReference`나 변수화 없이 Remote 가져오기는 불가능함(오류 없는 사일런트 실패 유발).
  4. **오류 메시지/디버깅 UX 문제**: Kustomize 빌드-어플라이는 에러 없이 통과해버리기 때문에, 사용자는 왜 자기 Metrics가 수집되지 않는지 Kubernetes 내부를 한참 뜯어봐야 하는 심각한 로깅/침묵의 UX를 가짐.

### Stage E — Approved Cleanup Execution & Final Synthesis

* 사용자 승인(User Approval)에 따라 확보된 정책 판단을 바탕으로, `presets/` 전체 디렉토리와 `scripts/check-slo-metrics.sh`를 소스 코드 트랙에서 영구 삭제(git rm) 함.
* `docs/notes/cleanup-policy-decision-input-2026-02-28.md`를 갱신하여 Condition Met 상태를 Execution Completed 상태로 변경함.
* **UX 부채 분리 (Stage D 파생)**: Kustomize 배포용 리소스(config/samples 등)가 `main` 브랜치에 그대로 남아있어 Remote Kustomize 접근 시 하드코딩 오류를 범하는 현상은 여전히 남아있음. 이는 삭제와는 별개의 문제이므로 Kustomize UX 부채로 라벨링하여 배포 구조 정립 과제(Backlog)로 격리함.

### Release & Tagging Preparation

* **태그 전략 (Tag Strategy)**: 제안 버전 `v1.0.0-rc.1`
  * **근거**: 라이브러리 E2E Harness 코어 로직이 안정화되었고, 불필요한 레거시(Standalone 찌꺼기)가 모두 청소됨. Stage B(Go import) 검증은 통과했으나, Kustomize UX 개선 및 Phase 3(Mock E2E) 구현 등 Consumer 온보딩을 위한 비기능적 백로그가 남았으므로 정식 `v1.0.0` 이전에 Release Candidate 1 을 발행하는 것이 적절함.
  * **명령어 (실행 대기용)**:
    1. `git tag -a v1.0.0-rc.1 -m "Release v1.0.0-rc.1: Cleanup and Harness Stabilization"`
    2. `git push origin v1.0.0-rc.1`
* **릴리즈 노트 초안**: `docs/RELEASE_NOTES_DRAFT.md` 참조.

### Release & Tagging Execution

* 정리된 태그 전략에 따라 `v1.0.0-rc.1` annotated tag 생성 및 `origin` 푸시 완료.
* (진단용 레거시/정리 상태 종결 및 정식 마일스톤 도달)

### Phase 3 Actualization Part 1 (Legacy E2E Replacement MVP)

* **테스트 구조 정합성**: `harness.Session`을 감싸는 단순하고 확실한 mock 테스트 경로 확보. `legacy_e2e`의 무거운 바이너리 파이프라인/배포 로직을 대체할 뼈대가 됨.
* **API 사용성 검증**: `SessionConfig.Fetcher` 확장이 외부 패키지에서도 완벽하게 열려 있음을 증명함.
* **안정성 (httptest)**: K8s 의존성이 전혀 없는 100% In-memory 파이프라인이므로 flakiness zero(0.01초 소요).

### Phase 3 Actualization Part 2 (Mock E2E Hardening & Legacy Removal Gate)

* **테스트 커버리지 고도화 완료**: P3-1 MVP를 기반으로 `test/e2e/harness_integration_test.go`를 Table-Driven 형식으로 재구축.
* **케이스별 실제 관찰 결과 보증**: 
  - **Missing Metric**: 응답에 Metric 정보가 없으면 Session 엔진이 `Skip` 판정 및 "missing input metrics" 사유를 뿜어냄을 인증.
  - **Fetch Error**: HTTP 500 에러 주입 시 Session이 뻗지 않고 Panic 없이 `Block/Skip` 상태 반환 및 신뢰도 지표 `Failed/Partial` 구조를 발송하는 것을 검증.
  - **Delta Path**: 카운터가 증가하는 시나리오(`ComputeDelta`)에서 `Start` (10.0), `End` (25.0) 를 모방하여 정상적으로 Delta 산출치(15.0)가 판정됨을 입증함.
* **안정성 및 CI 편입도**: `test/e2e/README.md`에 설명된 기존 E2E의 Flakiness 고질병(Pod 재시작, 클러스터 타임아웃 등)이 해당 테스트에선 HTTP Mock 통신으로 처리되므로 완벽히 없음을 확인.

---

### Stage Phase 3 Actualization Part 3 (Final Removal Execution)

* **테스트 전략 문서화**: `test/e2e/README.md`를 갱신하여 현재 레포지토리의 공식 통합 테스트가 Mock 기반 In-memory 테스트임을 명시.
* **레거시 자산 영구 삭제 완료**: `e2e_test.go`, `e2e_suite_test.go`, `manifests/`, `e2eutil/` 등 기존 `//go:build legacy_e2e`로 봉인되어 있던 파일과 디렉토리를 `git rm` 으로 소각.
* **결합 끊기**: `Makefile`에서 불필요하게 Kind 클러스터를 띄우고 지우던 고비용 `test-e2e` 스크립트를 깔끔하게 1줄 테스트(`go test ... -run TestHarnessIntegration_TableDriven`)로 대체. K8s 의존성이 테스트 스위트에서 영원히 제거됨.
* **삭제 게이트 완수 증명**: Happy path, Missing metric, Fetch error, Delta path 안정성 보증 및 `test/e2e/README.md` 대체 경로 안내 갱신 완료.

### Stage Phase 6-a (P0 DX Unblock)

* **실클러스터 통합 옵션(Knobs) 지원**: `SessionConfig`에 `CurlImage`와 `TLSInsecureSkipVerify` (자체 서명 인증서 무시) 옵션을 추가하고 `fetcher_curlpod`에 전달하여, 방화벽 내부 프라이빗 환경이나 외부 프로메테우스 연동 시 발생하는 Block 요소를 해소.
* **기본 동작 유지(No Regression)**: 기본 `curl` 이미지 태그 유지 및 TLS 검증(Verify) On 상태를 기본 동작으로 고수.
* **통합 가이드 반영**: `README.md` 및 `README(Kor).md`에 설정(`sess := harness.NewSession(...)`) 예시 및 RBAC(`create pods`) 관련 최소 주의사항 기재.

---

## Pending Items

### Stage Roadmap (draft)

1. [x] **Phase 6-b: goal/contract alignment**
   - identity/contract/모드/회귀게이트/소비자 기준 저장소(hello-operator) 문서 정렬 완료
2. [ ] **Phase 6-c: regression gate model**
   - baseline 대비 절대 임계치 + 회귀 비교를 policy gate로 1급화
3. [x] **Phase 6-d: GitHub Actions visibility**
   - `slint-gate`: policy violation 중심 gate
   - `roadmap-status`: 현재 stage/계약 충족도 요약
   - `baseline-update`: 승인 기반 baseline 갱신 경로는 후속 작업으로 남음
4. [x] **Phase 7-a: hello-operator consumer validation**
   - `hello-operator`를 canonical DX 검증 저장소로 고정
   - ko+tilt inner-loop 검증 기준선은 확정, 세부 하드닝은 후속 과제
5. [ ] **Release Gate: guardrail RC**
   - "테스트 프레임워크"가 아니라 "shift-left guardrail" 메시지/계약이 CI+문서+소비자 검증에서 일치할 때 RC 진행

### Follow-up (deferred)

우선순위에 따른 장기/단기 기술 부채:

1. [ ] RBAC 템플릿 및 실클러스터 배포 가이드라인 전면 표준화 (이번 문서 패링 외 본격적인 manifest 제공)
2. [ ] Kustomize Parameterization 구조 개편 착수 (Stage D UX 부채 해결을 위한 근본적 분리, Helm 등 장기 옵션 고려).
3. [ ] `kubeutil` 내 YAML Sprintf 하드코딩 부채(`TODO(security)`/`TODO(refactor)`) 해소
4. [ ] `sli-summary.json` CLI Console Output 요약 기능 지원

### Backlog (optional)

* [ ] Trigger-based 경계 지원 (Annotation/Condition 기반)
* [ ] policy 결과물과 measurement 결과물 분리 출력 템플릿(요약 리포트) 초안
* [ ] README / README(Kor) 메시지 재배치 (lint-style guardrail 전면화)

---

## Recent Validation Baseline

* `golangci-lint ./...` — PASS (2026-03-02)
* `go test ./test/e2e/harness/...` — PASS (2026-03-02)
* `make test-e2e` — PASS (2026-03-02)
* `kubectl kustomize test/consumer-onboarding/external-onboarding-validation/kustomize` — PASS (2026-03-02)

---

## Working Guardrails (Do not regress)

* GitHub Actions CI (Build/Test/Lint) must always pass (절대 실패 금지, 발생 즉시 해결)
* Non-invasive instrumentation (no production operator code instrumentation)
* E2E Hook-based measurement
* Measurement failure != test failure (best-effort / warn / skip)
* Raw metrics (`/metrics`) vs summarized output (`sli-summary.json`) separation
* Keep scope small; defer instead of expanding

---

## Deferred / Risks (rolling)

* E2E Hook 내부 에러가 외부 라이브러리(Ginkgo 등)에 전파될 때 환경마다 exit code나 Fail() 처리가 상이할 수 있는 리스크

---

## Notes for Next Agent / Next Chat (short)

* Start by reading `docs/PROGRESS_LOG.md`.
* Confirm Current Status + DoD before editing.
* Record out-of-scope findings in Deferred first.
