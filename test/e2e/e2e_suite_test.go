//go:build legacy_e2e
// +build legacy_e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/HeaInSeo/kube-slint/pkg/devutil"
	"github.com/HeaInSeo/kube-slint/pkg/kubeutil"
	"github.com/HeaInSeo/kube-slint/pkg/slo"
	"github.com/HeaInSeo/kube-slint/test/e2e/e2eutil"
)

var (
	// 선택적 환경 변수:
	// - CERT_MANAGER_INSTALL_SKIP=true: 테스트 설정 중 CertManager 설치를 생략함.
	skipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"

	// isCertManagerAlreadyInstalled는 CertManager CRD가 클러스터에서 발견되면 true로 설정됨.
	isCertManagerAlreadyInstalled = false

	// projectImage는 테스트할 코드 소스 변경 사항으로 빌드되고 로드될 이미지의 이름임.
	projectImage = "example.com/kube-slint:v0.0.1"

	// logger는 스위트 로거임. 항상 안전함 (nil -> no-op).
	logger = slo.NewLogger(e2eutil.GinkgoLog)

	// runner는 kubeutil/devutil 헬퍼(컨텍스트 인식)에서 사용됨.
	runner kubeutil.CmdRunner = kubeutil.DefaultRunner{}
	// useExistingCluster는 기존 클러스터를 사용할지 아니면 Kind를 프로비저닝할지 결정함.
	useExistingCluster = os.Getenv("USE_EXISTING_CLUSTER") == "true"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	logger.Logf("Starting kube-slint integration test suite")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	// 설정 단계를 위한 합리적인 기본 타임아웃임.
	// 개별 kubectl 명령어들도 자체적인 타임아웃을 가지고 있음 (예: kubectl wait --timeout).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if useExistingCluster {
		logger.Logf("USE_EXISTING_CLUSTER=true: skipping Kind cluster image build and load")
	} else {
		By("building the manager(Operator) image")
		root, err := devutil.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
		cmd.Dir = root

		_, err = runner.Run(ctx, logger, cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

		By("loading the manager(Operator) image on Kind")
		Expect(devutil.LoadImageToKindClusterWithName(ctx, logger, runner, projectImage)).
			To(Succeed(), "Failed to load the manager(Operator) image into Kind")
	}

	// 스킵되지 않았고 아직 설치되지 않은 경우 스위트 진행 전 CertManager를 설정함.
	if skipCertManagerInstall {
		logger.Logf("CERT_MANAGER_INSTALL_SKIP=true: skipping cert-manager setup")
		return
	}

	By("checking if cert-manager is installed already")
	isCertManagerAlreadyInstalled = kubeutil.IsCertManagerCRDsInstalled(ctx, logger, runner)
	if isCertManagerAlreadyInstalled {
		logger.Logf("WARNING: cert-manager is already installed; skipping installation")
		return
	}

	By("installing cert-manager")
	Expect(kubeutil.InstallCertManager(ctx, logger, runner)).
		To(Succeed(), "Failed to install cert-manager")
})

var _ = AfterSuite(func() {
	if skipCertManagerInstall || isCertManagerAlreadyInstalled {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	By("uninstalling cert-manager (best-effort)")
	if err := kubeutil.UninstallCertManager(ctx, logger, runner); err != nil {
		warnf("failed to uninstall cert-manager: %v", err)
	}
})

func warnf(format string, args ...any) {
	logger.Logf("WARNING: "+format, args...)
}
