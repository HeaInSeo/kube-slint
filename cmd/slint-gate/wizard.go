package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
)

const wizardUsage = `Usage: slint-gate wizard [flags]

Interactive onboarding: walks the same init -> recommend-policy -> baseline
approve -> ci github-actions loop that 'quickstart' describes, but prompts
for input and runs each step for you instead of just printing the next
command.

Requires an interactive terminal (stdin must be a TTY) -- refuses to run
under CI or piped/non-interactive stdin, where 'quickstart' or the
individual flag-driven commands (init, recommend-policy, baseline approve,
ci github-actions) should be used instead.

Flags:
`

func runWizard(args []string) error {
	fs := flag.NewFlagSet("wizard", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, wizardUsage)
		fs.PrintDefaults()
	}
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML")
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the measurement summary JSON")
	baselinePath := fs.String("baseline", "", "Path to a baseline summary JSON (prompted for when needed if not given)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if !isInteractiveStdin() {
		return fmt.Errorf("wizard requires an interactive terminal (stdin is not a TTY); use 'slint-gate quickstart' for a non-interactive status check, or run init/recommend-policy/baseline approve/ci github-actions directly")
	}

	return runWizardLoop(os.Stdin, *policyPath, *summaryPath, *baselinePath)
}

// isInteractiveStdin reports whether stdin is a real terminal (as opposed to
// a pipe, redirected file, or /dev/null, as CI and scripted invocations use).
// os.ModeCharDevice alone isn't sufficient here — /dev/null is itself a
// character device, so a naive os.Stdin.Stat() check would treat piped
// output-to-null the same as a real TTY. term.IsTerminal does the actual
// ioctl-based check. This is the concrete fix for the risk this command was
// originally deferred over: a stdin-prompting wizard invoked
// non-interactively would otherwise hang forever waiting for input that
// will never arrive.
func isInteractiveStdin() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// runWizardLoop drives the interactive loop, re-inspecting onboarding state
// after every step so it always acts on the current (not stale) state. It
// calls the same run* functions the individual subcommands use directly,
// with programmatically-built args, rather than reimplementing their logic.
// Declining a confirmation prompt stops the loop without error — the user
// can always resume later by re-running 'slint-gate wizard'.
func runWizardLoop(in io.Reader, policyPath, summaryPath, baselinePath string) error {
	sc := bufio.NewScanner(in)
	fmt.Println("kube-slint wizard — interactive onboarding")
	fmt.Println()

	for {
		st := inspectOnboardingState(policyPath, summaryPath, baselinePath)

		fmt.Println("Status:")
		fmt.Printf("  Policy:      %s\n", statusLine(st.HasPolicy, policyPath))
		fmt.Printf("  Measurement: %s\n", measurementStatusLine(st.HasSummary, st.SummaryLoadErr, st.Summary, summaryPath))
		if st.HasPolicy && st.HasSummary {
			fmt.Printf("  Gate:        %s\n", st.GateResult)
		} else {
			fmt.Println("  Gate:        not evaluated yet")
		}
		fmt.Println()

		done, next, err := runWizardStep(sc, nextOnboardingStep(st), st, policyPath, summaryPath, baselinePath)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		baselinePath = next
	}
}

// runWizardStep executes a single onboarding step. It returns done=true when
// the loop should stop (either the step is terminal, like wiring CI, or the
// user declined to proceed), and the (possibly updated) baselinePath to
// carry into the next iteration.
func runWizardStep(sc *bufio.Scanner, step onboardingStep, st onboardingState, policyPath, summaryPath, baselinePath string) (done bool, nextBaselinePath string, err error) {
	switch step {
	case stepRunE2E:
		fmt.Println("Next: run your E2E test with kube-slint attached, then re-run 'slint-gate wizard'.")
		return true, baselinePath, nil
	case stepInspectInvalidSummary, stepInspectFailedGate:
		return wizardStepInspect(sc, summaryPath, baselinePath)
	case stepInit:
		return wizardStepInit(sc, policyPath, baselinePath)
	case stepRecommendPolicy:
		return wizardStepRecommendPolicy(sc, policyPath, summaryPath, baselinePath)
	case stepApproveBaseline:
		return wizardStepApproveBaseline(sc, st, policyPath, summaryPath, baselinePath)
	default: // stepWireCI
		return wizardStepWireCI(sc, policyPath, summaryPath, baselinePath)
	}
}

func wizardStepInspect(sc *bufio.Scanner, summaryPath, baselinePath string) (bool, string, error) {
	fmt.Println("Next: slint-gate inspect --summary " + summaryPath)
	if !confirm(sc, "Run it now? [Y/n]: ", true) {
		return true, baselinePath, nil
	}
	if err := runInspect([]string{"--summary", summaryPath}); err != nil {
		fmt.Println("(inspect reported an issue above — fix it and re-run 'slint-gate wizard')")
	}
	return true, baselinePath, nil
}

func wizardStepInit(sc *bufio.Scanner, policyPath, baselinePath string) (bool, string, error) {
	fmt.Println("Next: slint-gate init --profile kubebuilder-operator")
	if !confirm(sc, "Run it now? [Y/n]: ", true) {
		return true, baselinePath, nil
	}
	ns, _ := promptLine(sc, "  Namespace for metrics service auto-discovery (optional, Enter to skip): ")
	svc, _ := promptLine(sc, "  MetricsServiceName override (optional, Enter to auto-discover/skip): ")
	initArgs := []string{"--output", policyPath, "--profile", "kubebuilder-operator"}
	if ns != "" {
		initArgs = append(initArgs, "--namespace", ns)
	}
	if svc != "" {
		initArgs = append(initArgs, "--service", svc)
	}
	if err := runInit(initArgs); err != nil {
		return true, baselinePath, fmt.Errorf("init: %w", err)
	}
	return false, baselinePath, nil
}

func wizardStepRecommendPolicy(sc *bufio.Scanner, policyPath, summaryPath, baselinePath string) (bool, string, error) {
	fmt.Println("Next: slint-gate recommend-policy --summary " + summaryPath + " --output " + policyPath)
	if !confirm(sc, "Run it now? [Y/n]: ", true) {
		return true, baselinePath, nil
	}
	if err := runRecommendPolicy([]string{"--summary", summaryPath, "--output", policyPath}); err != nil {
		return true, baselinePath, fmt.Errorf("recommend-policy: %w", err)
	}
	return false, baselinePath, nil
}

func wizardStepApproveBaseline(sc *bufio.Scanner, st onboardingState, policyPath, summaryPath, baselinePath string) (bool, string, error) {
	suggested := baselinePath
	if suggested == "" {
		suggested = filepath.Join("docs", "baselines", "sli-summary.json")
	}
	fmt.Println("Next: slint-gate baseline approve --summary " + summaryPath + " --policy " + policyPath)
	if !confirm(sc, "Run it now? [Y/n]: ", true) {
		return true, baselinePath, nil
	}
	out, ok := promptLine(sc, fmt.Sprintf("  Baseline output path [%s]: ", suggested))
	if !ok {
		return true, baselinePath, nil
	}
	if out == "" {
		out = suggested
	}
	approveArgs := []string{"--summary", summaryPath, "--policy", policyPath, "--output", out}
	if st.GateResult == gate.GateWarn {
		if confirm(sc, "  Gate result is WARN, not PASS. Approve anyway with --allow-warn? [y/N]: ", false) {
			approveArgs = append(approveArgs, "--allow-warn")
		}
	}
	if err := runBaselineApprove(approveArgs); err != nil {
		return true, baselinePath, fmt.Errorf("baseline approve: %w", err)
	}
	return false, out, nil // carry the newly-approved path into the next iteration
}

func wizardStepWireCI(sc *bufio.Scanner, policyPath, summaryPath, baselinePath string) (bool, string, error) {
	fmt.Println("Next: slint-gate ci github-actions --summary " + summaryPath + " --policy " + policyPath + " --baseline " + baselinePath)
	if !confirm(sc, "Print the CI step snippet now? [Y/n]: ", true) {
		return true, baselinePath, nil
	}
	ciArgs := []string{"--summary", summaryPath, "--policy", policyPath}
	if baselinePath != "" {
		ciArgs = append(ciArgs, "--baseline", baselinePath)
	}
	if err := runCIGithubActions(ciArgs); err != nil {
		return true, baselinePath, fmt.Errorf("ci github-actions: %w", err)
	}
	fmt.Println("\nOnboarding loop complete.")
	return true, baselinePath, nil
}

// confirm prompts and returns the user's yes/no answer, falling back to
// defaultYes on an empty answer or on EOF (e.g. piped input running out).
func confirm(sc *bufio.Scanner, prompt string, defaultYes bool) bool {
	fmt.Print(prompt)
	if !sc.Scan() {
		return defaultYes
	}
	ans := strings.ToLower(strings.TrimSpace(sc.Text()))
	if ans == "" {
		return defaultYes
	}
	return ans == "y" || ans == "yes"
}

// promptLine prompts for a free-text line, returning ok=false only on EOF
// (no more input available) — an empty answer is ok=true, value "".
func promptLine(sc *bufio.Scanner, prompt string) (string, bool) {
	fmt.Print(prompt)
	if !sc.Scan() {
		return "", false
	}
	return strings.TrimSpace(sc.Text()), true
}
