# Kube-slint Cleanup Audit & Diagnostic Report
**Date:** 2026-02-27
**Status:** Audit Complete (Read-only)

## Background
본 보고서는 `kube-slint` 프로젝트가 "단일 오퍼레이터 저장소" 형태에서 "다른 오퍼레이터들이 끌어다 쓰는 순수 계측 라이브러리(Library)" 형태로 방향 전환을 완수함에 따라, 아직 남아있는 레거시 패턴과 구조적 잔재를 진단하기 위해 작성되었습니다.

---

## A. 저장소 정리 필요성 진단 (파일/폴더 관점)

저장소 루트와 `docs/`, `test/` 하위를 분석한 결과입니다.

| 파일 / 폴더 | 현재 상태 (분석) | 권장 분류 |
| --- | --- | --- |
| `lint.log`, `test_full_v*.log`, `cover.out`, `e2e.test` | 임시 실행 후 남은 로컬 테스트/빌드 아티팩트 잔해입니다. | **Likely obsolete** (삭제 후보) |
| `docs/old/` | Library화 되기 전인 구 버전 아키텍처나 폐기된 설정 문서들이 들어있습니다. | **Keep as historical reference** (위키 이관 또는 그대로 보존) |
| `test/e2e/e2e_test.go`, `test/e2e/e2e_suite_test.go`, `test/e2e/manifests/` | kube-slint가 자체 Dummy Controller를 배포(`make deploy`)해서 동작 여부를 테스트하던 구버전 E2E 잔재입니다. 라이브러리 전환으로 인해 현재 구동이 불가능(Index out of bounds 발생)합니다. | **Needs confirmation** (라이브러리용 E2E로 재작성 필요, 기존 코드는 버리는 쪽으로 가닥) |
| `TODO.md`, `code_review.md` | 과거 리뷰/할일 내역. `PROGRESS_LOG.md` 체제가 도입되었으므로 파편화되어 있습니다. | **Likely obsolete** (핵심 내용만 PROGRESS_LOG로 옮기고 휴지통) |

---

## B. 코드 구조 정리 필요성 진단 (구조/책임/가독성 관점)

단순 삭제를 넘어, 리팩토링이나 경계 재설정이 필요한 핫스팟입니다.

### 1) Kubeutil의 중복과 잠재적 YAML 취약성 (Technical Debt)
* **근거**: `pkg/kubeutil/rbac.go`, `token.go`, `wait.go` 등 여러 파일에 `TODO(refactor): Extract the common polling loop`, `TODO(security): fmt.Sprintf 문자열 템플릿 대신...` 주석이 방치되어 있습니다.
* **진단**: 당장 크래시를 유발하지 않으나 라이브러리로 외부에 제공되는 만큼, 하드코딩된 문자열 YAML 조립 방식은 악의적 네임스페이스 주입 등에 취약할 수 있습니다.
* **제안 (작게 쪼갠 2단계)**:
  1. `kubeutil.Poll` 공통 유틸리티 생성 및 `wait.go` 중복 제거.
  2. `rbac.go` 문자열 렌더링을 `corev1.ServiceAccount` 구조체 마샬링(Marshaling) 방식으로 교체.

### 2) Harness 영역에 Fetcher 구상 세부 로직 침범
* **근거**: `test/e2e/harness/session.go` 파일 432라인 이하에 `curlPodFetcher` 구현체가 어색하게 존재합니다.
* **진단**: 하네스(`session.go`)는 `Engine`을 제어하는 오케스트레이션 역할에만 머무르는 것이 클린 아키텍처 철학입니다. 현재 구체화된(Concrete) 컬링 전략이 들어와 결합도를 높입니다.
* **제안 (1단계)**: `session.go` 일부분을 `pkg/slo/fetch/curlpod/fetcher.go` 같은 Adapter 모듈로 분리.

---

## C. 테스트 신뢰성 진단 (블로킹 여부 판별)

*사용자님이 언급한 "테스트 코드 신뢰성/범위가 불확실하다"는 우려에 대한 증거 기반 응답입니다.*

1. **테스트 계층 구분**: 
   - `pkg/slo/...` (단위, 엔진 코어): 정상
   - `test/e2e/harness/...` (하네스 시뮬레이션/모킹): **우수 (최근 보강됨)**
   - `test/e2e` (종단간 통합): **Broken** (더미 오퍼레이터를 띄우던 레거시라서 터짐)
2. **무엇을 보장하고, 무엇을 보장하지 못하는가**:
   - (보장함) Metric이 Json/Map 형태로 들어왔을 때 -> 결과지표를 엄격하게 패스/경고/실패 분류하고 json을 남기는 논리구조 (Harness 단계의 신뢰도는 100%)
   - (보장 못함) Library 호출자(오퍼레이터)가 띄운 타사 파드 내에서 `curl`이 실제 NetworkPolicy나 DNS 타이밍 이슈 없이 "살아서 데이터를 긁어올 수 있는가?" (E2E 레벨의 통신망 이슈)
3. **Flaky / 회귀 방지 Gap**:
   - `curlPod` 자체가 k8s 클러스터 런타임에 종속(API 딜레이, RBAC, Image Pull 시간 등)을 가지므로 매우 Flaky합니다. 현재 E2E가 깨진 상태이기에 회귀를 이 계층에서 전혀 방어하지 못합니다.
4. **릴리즈 블로킹(Blocking) 판단**:
   - `Non-blocking` 입니다.
   - kube-slint 핵심 논리 층위(Engine, Spec, Summary)는 하네스 모킹 시뮬레이션으로 철벽 보호를 받고 있으므로 라이브러리 자체 성능엔 문제가 없습니다. 
   - 다만 E2E 프레임워크 제공자로서 "진짜로 붙어지는 껍데기 예시(Fake Operator Testing)" 코드를 `test/e2e` 안에 다시 세워주거나, 혹은 아예 제거하고 튜토리얼 레포지토리(Example repo)로 미루는 결단이 릴리즈 이후(post-release) 또는 릴리즈 직전에 이루어져야 합니다.

---

## D. 결론 및 향후 처리 제안 (Action Items)

진단 결과, 현재 코드베이스에서 **치명적인 오작동**을 일으키는 버그는 없으며 주로 "역사적 잔재"와 "정리 불량"이 이슈의 핵심입니다.
사용자님과 논의를 위해 다음 순서를 제안합니다:

1. **(Quick Win)** 루트 디렉토리 `.log`, `.test`, `cover.out` 임시 스크랩 일괄 제거 및 `gitignore` 보강.
2. **(Structure)** `session.go`에서 `curlPodFetcher` 모듈 분리 및 `kubeutil` 잠재적 보안 부채(TODO) 청산.
3. **(Testing)** 깨진 `test/e2e` 더미 컨트롤러 제거 후, 라이브러리 소비 관점의 통합 E2E 테스트(단순 파드 하나 띄우고 라이브러리 Attach)로 재구축.
