package curlpod

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/kubeutil"
	"github.com/HeaInSeo/kube-slint/pkg/slo"
)

// PodLabelSelector는 curl 파드에 사용되는 레이블임.
const PodLabelSelector = "app=curl-metrics"

// Client는 curl-metrics 파드를 실행하고 로그를 가져옴.
// 테스트 지향적임 (kubectl + curlimages/curl 사용).
type Client struct {
	Logger slo.Logger
	Runner kubeutil.CmdRunner

	// 옵션 조정 가능 항목
	Image            string
	LabelSelector    string
	PodNamePrefix    string
	ServiceURLFormat string // e.g. "https://%s.%s.svc:8443/metrics"
}

// New는 안전한 기본값으로 클라이언트를 생성함.
// logger는 nil일 수 있음.
func New(logger slo.Logger, r kubeutil.CmdRunner) *Client {
	if r == nil {
		r = kubeutil.DefaultRunner{}
	}
	return &Client{
		Logger:           slo.NewLogger(logger),
		Runner:           r,
		Image:            "curlimages/curl:latest",
		LabelSelector:    PodLabelSelector,
		PodNamePrefix:    "curl-metrics",
		ServiceURLFormat: "https://%s.%s.svc:8443/metrics",
	}
}

// RunOnce는 /metrics를 스크랩하는 수명이 짧은 curl 파드를 생성함.
// 생성된 파드 이름을 반환함.
// 기다리지 않으므로 WaitDone을 호출한 다음 Logs를 호출해야 함.
func (c *Client) RunOnce(ctx context.Context, ns, token, metricsSvcName, serviceAccountName string) (string, error) {
	c.Logger = slo.NewLogger(c.Logger)
	if c.Runner == nil {
		c.Runner = kubeutil.DefaultRunner{}
	}

	// 이전 curl-metrics 파드 최선(best-effort) 정리
	_ = c.CleanupByLabel(ctx, ns)

	podName := fmt.Sprintf("%s-%d", c.PodNamePrefix, time.Now().UnixNano())
	metricsURL := fmt.Sprintf(c.ServiceURLFormat, metricsSvcName, ns)

	// 테스트 환경의 자체 서명 인증서를 위해 -k 유지, 출력 깔끔하게 유지 (-v 없음)
	curlCmd := fmt.Sprintf(`set -euo pipefail;
curl -ksS --fail-with-body -H "Authorization: Bearer %s" "%s";`, token, metricsURL)

	cmd := exec.Command(
		"kubectl", "run", podName,
		"--restart=Never",
		"--namespace", ns,
		"--image", c.Image,
		"--labels", c.LabelSelector,
		"--overrides",
		fmt.Sprintf(`{
  "apiVersion":"v1",
  "kind":"Pod",
  "metadata":{
    "name":"%s",
    "namespace":"%s",
    "labels":{"app":"curl-metrics"}
  },
  "spec":{
    "serviceAccountName":"%s",
    "restartPolicy":"Never",
    "containers":[{
      "name":"curl",
      "image":"%s",
      "command":["/bin/sh","-c",%q],
      "securityContext":{
        "allowPrivilegeEscalation": false,
        "capabilities": { "drop": ["ALL"] },
        "runAsNonRoot": true,
        "runAsUser": 1000,
        "seccompProfile": { "type": "RuntimeDefault" }
      }
    }]
  }
}`, podName, ns, serviceAccountName, c.Image, curlCmd),
	)

	_, err := c.Runner.Run(ctx, c.Logger, cmd)
	return podName, err
}

// WaitDone은 curl 파드가 종료 단계(Succeeded/Failed)에 도달할 때까지 기다림.
func (c *Client) WaitDone(ctx context.Context, ns, podName string, poll time.Duration) error {
	c.Logger = slo.NewLogger(c.Logger)
	if c.Runner == nil {
		c.Runner = kubeutil.DefaultRunner{}
	}
	if poll <= 0 {
		poll = 2 * time.Second
	}

	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	// 즉시 첫 번째 확인
	if done, err := c.isTerminal(ctx, ns, podName); err != nil {
		return err
	} else if done {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, err := c.isTerminal(ctx, ns, podName)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

// Logs는 주어진 파드의 kubectl 로그를 반환함.
func (c *Client) Logs(ctx context.Context, ns, podName string) (string, error) {
	c.Logger = slo.NewLogger(c.Logger)
	if c.Runner == nil {
		c.Runner = kubeutil.DefaultRunner{}
	}

	cmd := exec.Command("kubectl", "logs", podName, "-n", ns)
	return c.Runner.Run(ctx, c.Logger, cmd)
}

// DeletePodNoWait는 기다리지 않고 최선의 노력(best-effort)으로 파드를 삭제함.
func (c *Client) DeletePodNoWait(ctx context.Context, ns, podName string) error {
	c.Logger = slo.NewLogger(c.Logger)
	if c.Runner == nil {
		c.Runner = kubeutil.DefaultRunner{}
	}

	cmd := exec.Command(
		"kubectl", "delete", "pod", podName,
		"-n", ns,
		"--ignore-not-found=true",
		"--wait=false",
	)
	_, err := c.Runner.Run(ctx, c.Logger, cmd)
	return err
}

// CleanupByLabel은 레이블 셀렉터로 모든 curl-metrics 파드를 삭제함 (최선의 노력, 기다리지 않음).
func (c *Client) CleanupByLabel(ctx context.Context, ns string) error {
	c.Logger = slo.NewLogger(c.Logger)
	if c.Runner == nil {
		c.Runner = kubeutil.DefaultRunner{}
	}

	cmd := exec.Command(
		"kubectl", "delete", "pod",
		"-n", ns,
		"-l", c.LabelSelector,
		"--ignore-not-found=true",
		"--wait=false",
	)
	_, err := c.Runner.Run(ctx, c.Logger, cmd)
	// 최선의 노력(best-effort)이므로 에러를 하드 실패로 간주하지 않고 호출부에서 무시해도 됨.
	return err
}

func (c *Client) isTerminal(ctx context.Context, ns, podName string) (bool, error) {
	cmd := exec.Command(
		"kubectl", "get", "pod", podName,
		"-n", ns,
		"-o", "jsonpath={.status.phase}",
	)
	out, err := c.Runner.Run(ctx, c.Logger, cmd)
	if err != nil {
		return false, err
	}
	phase := strings.TrimSpace(out)
	return phase == "Succeeded" || phase == "Failed", nil
}
