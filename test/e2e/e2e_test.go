package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/HeaInSeo/kube-slint/pkg/devutil"
	"github.com/HeaInSeo/kube-slint/pkg/kubeutil"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/curlpod"

	"github.com/HeaInSeo/kube-slint/test/e2e/harness"
	e2eenv "github.com/HeaInSeo/kube-slint/test/e2e/internal/env"
	"github.com/HeaInSeo/kube-slint/test/e2e/manifests"
)

// Step 6 후보: 이거 따로 빼야 함.
const namespace = "kube-slint-system"
const serviceAccountName = "kube-slint-controller-manager"
const metricsServiceName = "kube-slint-controller-manager-metrics-service"

var _ = Describe("Manager", Ordered, func() {
	var (
		// Step 6 후보: 추후 런타임에 쓰이는 정보들, 초기 설정에 관련된 정보들, 계측에 필요한 설정 등은 정리했지만 문서로 만들어 둘 것.
		cfg     e2eenv.Options
		rootDir string

		cm *curlpod.Client

		// 테스트마다 공유됨
		metricsToken string
		metricsPod   *curlpod.CurlPod
		//token   string
	)

	BeforeAll(func() {
		cfg = e2eenv.LoadOptions()
		By(fmt.Sprintf("ArtifactsDir=%q RunID=%q Enabled=%v", cfg.ArtifactsDir, cfg.RunID, cfg.Enabled))

		var err error
		rootDir, err = devutil.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		cm = curlpod.New(logger, runner)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Step 6 후보: e2eutil로 뺄 것.
		run := func(cmd *exec.Cmd, msg string) string {
			cmd.Dir = rootDir
			out, err := runner.Run(ctx, logger, cmd)
			Expect(err).NotTo(HaveOccurred(), msg)
			return out
		}

		By("기준(baseline) 보안 강제 적용으로 매니저 네임스페이스 생성")
		//		nsManifest := fmt.Sprintf(`apiVersion: v1
		// kind: Namespace
		// metadata:
		//   name: %s
		// `, namespace)
		// Step 6 후보: apply.go에서 ApplyTemplate 적용할지 고민 중
		nsManifest, err := devutil.RenderTemplateFileString(
			rootDir,
			"test/e2e/manifests/namespace.tmpl.yaml.gotmpl",
			manifests.NamespaceData{Namespace: namespace},
		)
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		cmd.Dir = rootDir
		cmd.Stdin = strings.NewReader(nsManifest)
		run(cmd, "Failed to apply namespace with security policy")

		//By("labeling the namespace to enforce the security policy")
		//cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace, "pod-security.kubernetes.io/enforce=baseline")
		//cmd.Dir = rootDir
		//run(cmd, "Failed to label namespace with security policy")

		By("CRD 설치")
		run(exec.Command("make", "install"), "CRD 설치 실패")

		By("controller-manager 배포")
		run(exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage)), "controller-manager 배포 실패")

		// Step 6 후보: 추후 ApplyClusterRoleBinding을 감싸서 구현할 수도 있는데 고민 중.
		By("controller-manager SA에 대한 메트릭 리더(reader) RBAC 보장 (멱등성)")
		Expect(kubeutil.ApplyClusterRoleBinding(
			ctx, logger, runner,
			"kube-slint-e2e-metrics-reader",
			"kube-slint-metrics-reader",
			namespace,
			serviceAccountName,
		)).To(Succeed())
	})

	AfterAll(func() {
		if cfg.SkipCleanup {
			By("E2E_SKIP_CLEANUP 활성화됨: 정리 건너뜀")
			return
		}
		// Step 6 후보: 10*time.Minute 따로 뺄 것.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		By("최선(best-effort): curl-metrics 파드 정리")
		_ = cm.CleanupByLabel(ctx, namespace)
		// Step 6 후보: 기본 Makefile에 대한 의존성이 생기지만 무시해도 될 듯함.
		By("controller-manager 배포 해제 (최선의 노력)")
		cmd := exec.Command("make", "undeploy")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
		// Step 6 후보: 기본 Makefile에 대한 의존성이 생기지만 무시해도 될 듯함.
		By("CRD 제거 (최선의 노력)")
		cmd = exec.Command("make", "uninstall")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
		// Step 6 후보: curlmetrics.go 사용하자.
		By("매니저 네임스페이스 제거 (최선의 노력)")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found=true")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
	})

	// Step 6 후보: opts *WaitOptions로 할지 고민 중. 5*time.Minute 따로 뺄 것.
	BeforeEach(func() {
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer waitCancel()

		opts := kubeutil.WaitOptions{}

		By("controller-manager 준비 대기")
		Expect(kubeutil.WaitControllerManagerReady(waitCtx, logger, runner, namespace, opts)).To(Succeed())

		By("메트릭 서비스 엔드포인트 준비 대기")
		Expect(kubeutil.WaitServiceHasEndpoints(waitCtx, logger, runner, namespace, metricsServiceName, opts)).To(Succeed())

		// ---- 공유 토큰 + curlpod (harness 및 It 모두에서 사용) ----
		tokCtx, cancel := context.WithTimeout(context.Background(), cfg.TokenRequestTimeout)
		defer cancel()

		By("서비스 어카운트 토큰 요청 (공유됨)")
		t, err := kubeutil.ServiceAccountToken(tokCtx, logger, runner, namespace, serviceAccountName)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeEmpty())
		metricsToken = t

		metricsPod = &curlpod.CurlPod{
			Client:             cm,
			Namespace:          namespace,
			MetricsServiceName: metricsServiceName,
			ServiceAccountName: serviceAccountName,
			Token:              metricsToken,
			// Image / ServiceURLFormat 재정의 필요하면 여기서 지정
		}

		// 참고:
		// - 아래 Fetcher 타입은 제안한 방식대로 pkg/slo/fetch/curlpod/fetcher.go로 빼는 게 정석임.
		// - 아직 없으면, 일단 harness 내부 default fetcher를 유지하거나,
		//   test-side adapter로 fetcher를 임시 구현해도 됨.
		// metricsFetcher = &curlpod.Fetcher{
		// 	Pod:               metricsPod,
		// 	AggregateNameOnly: true,
		// 	// 시간 초과 재정의 필요하면 여기서
		// }
		// 일단 유지함
	})

	// V4 하네스 사용 (표준화됨)
	_, err := harness.Attach(func() harness.SessionConfig {
		// 참고: 토큰 발급/스크랩 로직은 BeforeEach에서 공유됨.
		// tokCtx, cancel := context.WithTimeout(context.Background(), cfg.TokenRequestTimeout)
		// defer cancel()

		// By("하네스용 서비스 어카운트 토큰 요청")
		// t, err := kubeutil.ServiceAccountToken(tokCtx, logger, runner, namespace, serviceAccountName)
		// Expect(err).NotTo(HaveOccurred())
		// Expect(t).NotTo(BeEmpty())

		return harness.SessionConfig{
			// Step 6 후보: Enabled 지울지 고민하자. 일단 주석 처리함.
			//Enabled: 			cfg.Enabled,
			Namespace:          namespace,
			MetricsServiceName: metricsServiceName,
			TestCase:           "", // 하네스에 의해 자동 채워짐
			Suite:              "e2e",

			RunID: cfg.RunID,
			//ServiceAccountName: serviceAccountName,
			//Token:              t,
			ArtifactsDir: cfg.ArtifactsDir,
			// Step 6 후보: 일단 이렇게 주석 처리함. 확인 필요.
			//Fetcher: metricsFetcher,

			// Step 6 후보(태그): 런 상관관계(correlation) 분석을 위해 실행 메타 태그를 추가함.
			// 예: git commit SHA, kind cluster name, controller image tag, k8s version, CI run id 등
			// Tags: map[string]string{
			// 	"commit":  "",
			// 	"cluster": "",
			// 	"image":   "",
			// },
		}
	})
	Expect(err).NotTo(HaveOccurred())

	It("메트릭 엔드포인트가 메트릭을 제공하는지 확인해야 함", func() {
		By("curl 파드를 통해 /metrics 스크랩")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		text, err := metricsPod.Run(ctx, 5*time.Minute, 2*time.Minute)
		Expect(err).NotTo(HaveOccurred())

		if !strings.Contains(text, "controller_runtime_reconcile_total") {
			head := text
			if len(head) > 800 {
				head = head[:800]
			}
			logger.Logf("metrics text head:\n%s", head)
		}

		Expect(text).To(ContainSubstring("controller_runtime_reconcile_total"))
		By("완료")
	})
})
