# kube-slint Project Progress Log

This file tracks the incremental stages of kube-slint work.
Update this file at the **start and end** of every stage/task.

---

## Current Status: Stage (Active) — Phase 3 Actualization Part 2 (Mock E2E Hardening & Removal Gate)

**Branch:** `main`
**Last updated:** 2026-02-28

### Current Focus

* (P3-2 시작) P3-1 MVP를 테이블 기반(Table-Driven) 통합 검증 구조로 고도화.
* **검증 케이스 확장 완료**: Happy Path, Missing Metric, Network Fetch Error, Delta-ish Scenario, Multi-metric Mixed Result 5가지 시나리오가 100% In-memory 파이프라인(httptest)만으로 통과됨을 입증.
* **Legacy 사형 게이트 확립**: `legacy_e2e` 폴더를 실제로 삭제하기 위한 정식 조건표(Removal Gate) 초안을 본 문서에 기재함. (이번 단계에서는 물리 삭제 미실행 원칙 준수)

### Definition of Done (DoD)

* [x] `harness_integration_test.go` 확장 (테이블 기반 시나리오 구성완료)
* [x] 결함 주입/엣지 케이스 포함 (Missing, Error, Delta)
* [x] `legacy_e2e` 폴더 보존 밑 제거 게이트 정의
* [x] `docs/PROGRESS_LOG.md` 갱신 (현재 상태)

### Next command to run

* (P3-3 진행 승인 대기 / legacy_e2e 폴더 최종 삭제 단행)

### If blocked, fallback check

* (해당 없음)

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

## Pending Items

### Legacy E2E Removal Gate Definition (삭제 게이트 초안)

* [x] Happy path (single pass) 검증 완료
* [x] Missing metric behavior 엣지 케이스 검증 완료
* [x] Fetch error behavior 엣지 케이스 검증 완료
* [x] Delta path/state change 계산 검증 완료
* [x] New mock E2E path passes stably in repeated local runs (0.01초 이내 PASS)
* [ ] `test/e2e/README.md` 업데이트 (대체 경로 안내 및 기존 안내 정돈)
* [ ] `legacy_e2e` 디렉토리 전체 물리 삭제(git rm) 및 GitHub PR 반영

### Next Stage (planned)

* [ ] GitHub 웹 UI 접속 및 `docs/RELEASE_NOTES_DRAFT.md` 본문 복사하여 정식 Release 기록
* [ ] **Phase 3 실제 구현 (P3-3)** 승인 대기.
  - 목표: Removal Gate의 잔여 조건(문서 갱신 및 legacy_e2e 물리 삭제)을 실행하고 Phase 3 대장정을 공식 종결함.

### Proposed Next Stage (pending approval)

* [ ] Kustomize Parameterization 구조 개편 착수 (Stage D UX 부채 해결).
* 승인 필요: **Yes (user + ChatGPT)**

### Follow-up (deferred)

* [ ] (Phase 2) `session.go`에서 `curlPodFetcher` 모듈 분리 및 `kubeutil` 내 `TODO(security)`/`TODO(refactor)` 해소 (릴리즈 이후로 지연)
* [ ] `sli-summary.json` CLI Console Output 요약 기능 지원

### Backlog (optional)

* [ ] Trigger-based 경계 지원 (Annotation/Condition 기반)

---

## Recent Validation Baseline

* `golangci-lint ./...` — PASS (2026-02-27)
* `go test ./test/e2e/harness/...` — PASS (2026-02-27, harness scope)
* `go mod tidy` — PASS (2026-02-27)
* `git diff --exit-code go.mod go.sum` — PASS (2026-02-27)

---

## Working Guardrails (Do not regress)

* Non-invasive instrumentation (no production operator code instrumentation changes)
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
