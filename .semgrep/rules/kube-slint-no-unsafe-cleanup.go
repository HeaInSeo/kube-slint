package fixtures

import (
	"context"
	"os/exec"
)

var execCommandContext = exec.CommandContext

func bad(name, ns string) *exec.Cmd {
	// ruleid: kube-slint-no-unsafe-cleanup
	return exec.Command("kubectl", "delete", "pods", name, "-n", ns, "--ignore-not-found=true")
}

func badWithContext(ctx context.Context, name, ns string) *exec.Cmd {
	// ruleid: kube-slint-no-unsafe-cleanup
	return execCommandContext(ctx, "kubectl", "delete", "pods", name, "-n", ns, "--ignore-not-found=true")
}

func good(ns, selector string) *exec.Cmd {
	// ok: kube-slint-no-unsafe-cleanup
	return exec.Command("kubectl", "delete", "pods", "-n", ns, "-l", selector, "--ignore-not-found=true")
}

func goodWithContext(ctx context.Context, ns, selector string) *exec.Cmd {
	// ok: kube-slint-no-unsafe-cleanup
	return execCommandContext(ctx, "kubectl", "delete", "pods", "-n", ns, "-l", selector, "--ignore-not-found=true")
}
