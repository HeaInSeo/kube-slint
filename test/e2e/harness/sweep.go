package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// OrphanSweepOptions는 고아(orphan) 리소스 정리 동작을 설정함.
type OrphanSweepOptions struct {
	Enabled bool
	Mode    string        // "report-only" (기본값) | "delete"
	Limit   int           // 한 번에 삭제/보고할 최대 고아 리소스 수 (0이면 무제한)
	MaxAge  time.Duration // 이 시간보다 오래된 리소스만 대상으로 함 (0이면 검사 안 함)
}

// DevSweepOptions 제공: 로컬 개발 시 짧은 MaxAge와 적은 Limit
var DevSweepOptions = OrphanSweepOptions{
	Enabled: true,
	Mode:    "report-only",
	Limit:   10,
	MaxAge:  10 * time.Minute,
}

// CISweepOptions 제공: CI 환경에서 적절한 Limit 부여
var CISweepOptions = OrphanSweepOptions{
	Enabled: true,
	Mode:    "report-only",
	Limit:   100,
	MaxAge:  1 * time.Hour,
}

// SweepRequest represents the request parameters for the orphan sweeper.
type SweepRequest struct {
	Namespace           string `json:"namespace"`
	Selector            string `json:"selector"`
	ModeRequested       string `json:"modeRequested"`
	Limit               int    `json:"limit"`
	MaxAgeSeconds       int    `json:"maxAgeSeconds"`
	ExcludeCurrentRunID bool   `json:"excludeCurrentRunId"`
	CurrentRunID        string `json:"currentRunId,omitempty"`
}

// SweepApply represents the applied mode and fallback status.
type SweepApply struct {
	ModeEffective  string `json:"modeEffective"`
	ModeFallback   bool   `json:"modeFallback"`
	FallbackReason string `json:"fallbackReason,omitempty"`
}

// SweepSummary provides aggregated counts of the sweep operation.
type SweepSummary struct {
	Scanned         int            `json:"scanned"`
	Evaluated       int            `json:"evaluated"`
	WouldDelete     int            `json:"wouldDelete"`
	Deleted         int            `json:"deleted"`
	DeleteError     int            `json:"deleteError"`
	Skipped         int            `json:"skipped"`
	SkippedByReason map[string]int `json:"skippedByReason"`
}

// SweepItem records the result for an individual orphan resource.
type SweepItem struct {
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	CreatedAt  string `json:"createdAt,omitempty"`
	AgeSeconds int64  `json:"ageSeconds,omitempty"`
	RunID      string `json:"runId,omitempty"`
	Action     string `json:"action"` // "would-delete", "deleted", "skipped", "delete-error"
	Reason     string `json:"reason,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SweepResult contains the complete output of an orphan sweep operation.
type SweepResult struct {
	SchemaVersion string       `json:"schemaVersion"`
	StartedAt     time.Time    `json:"startedAt"`
	FinishedAt    time.Time    `json:"finishedAt"`
	DurationMs    int64        `json:"durationMs"`
	Request       SweepRequest `json:"request"`
	Apply         SweepApply   `json:"apply"`
	Summary       SweepSummary `json:"summary"`
	Items         []SweepItem  `json:"items"`
	Warnings      []string     `json:"warnings,omitempty"`
}

// WriteSweepResultJSON writes the SweepResult to the provided io.Writer as pretty JSON.
func WriteSweepResultJSON(w io.Writer, r SweepResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// SweepOrphans는 이전 kube-slint run-id의 리소스를 탐지하고 선택적으로 삭제함.
// 기존 API 호환성을 위해 SweepOrphansWithResult 결과를 버리고 성공 여부만 반환함.
func (s *Session) SweepOrphans(ctx context.Context, opts OrphanSweepOptions) error {
	_, err := s.SweepOrphansWithResult(ctx, opts)
	return err
}

// SweepOrphansWithResult detects and optionally deletes resources from previous kube-slint run-ids.
// It returns a SweepResult struct containing the detailed outcome of the sweep operation.
func (s *Session) SweepOrphansWithResult(ctx context.Context, opts OrphanSweepOptions) (SweepResult, error) {
	startedAt := time.Now()
	res := SweepResult{
		SchemaVersion: "v1.0",
		StartedAt:     startedAt,
		Summary: SweepSummary{
			SkippedByReason: make(map[string]int),
		},
		Items:    []SweepItem{},
		Warnings: []string{},
	}

	if s == nil || s.impl == nil || !opts.Enabled {
		res.FinishedAt = time.Now()
		res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
		return res, nil
	}

	ns := s.impl.Config.Namespace
	runID := s.impl.RunID

	if ns == "" || runID == "" {
		res.Summary.SkippedByReason["missing_guard"]++
		res.Warnings = append(res.Warnings, "skip - missing namespace or run-id for safety guard")
		fmt.Printf("kube-slint [orphan-sweep]: skip - missing namespace or run-id for safety guard\n")
		res.FinishedAt = time.Now()
		res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
		return res, nil
	}

	// Normalize mode
	modeReq := strings.TrimSpace(opts.Mode)
	modeEff := modeReq
	fallback := false
	fallbackReason := ""

	if modeEff != "delete" && modeEff != "report-only" {
		modeEff = "report-only"
		fallback = true
		fallbackReason = "invalid_mode"
		warnMsg := fmt.Sprintf("invalid mode %q provided, falling back to report-only", modeReq)
		res.Warnings = append(res.Warnings, warnMsg)
		fmt.Printf("kube-slint [orphan-sweep]: warning - %s\n", warnMsg)
	}

	res.Request = SweepRequest{
		Namespace:           ns,
		ModeRequested:       modeReq,
		Limit:               opts.Limit,
		MaxAgeSeconds:       int(opts.MaxAge.Seconds()),
		ExcludeCurrentRunID: true,
		CurrentRunID:        runID,
	}

	res.Apply = SweepApply{
		ModeEffective:  modeEff,
		ModeFallback:   fallback,
		FallbackReason: fallbackReason,
	}

	labelSelector := fmt.Sprintf("app.kubernetes.io/managed-by=kube-slint,slint-run-id!=%s", runID)
	res.Request.Selector = labelSelector

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", ns, "-l", labelSelector, "-o", "jsonpath={range .items[*]}{.metadata.name},{.metadata.labels.slint-run-id},{.metadata.creationTimestamp}{\"\\n\"}{end}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		res.FinishedAt = time.Now()
		res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
		return res, fmt.Errorf("failed to list orphans: %v (output: %s)", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("kube-slint [orphan-sweep]: mode=%s ns=%s run-id=%s :: no orphan resources detected\n", modeEff, ns, runID)
		res.FinishedAt = time.Now()
		res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
		return res, nil
	}

	var targetNames []string
	var hitLimit int

	for _, line := range lines {
		parts := strings.SplitN(line, ",", 3)
		if len(parts) != 3 {
			continue
		}
		res.Summary.Scanned++
		name, rId, tsStr := parts[0], parts[1], parts[2]

		item := SweepItem{
			Kind:      "Pod",
			Namespace: ns,
			Name:      name,
			RunID:     rId,
			CreatedAt: tsStr,
		}

		ts, err := time.Parse(time.RFC3339, tsStr)
		if err == nil {
			item.AgeSeconds = int64(startedAt.Sub(ts).Seconds())
			if opts.MaxAge > 0 && startedAt.Sub(ts) < opts.MaxAge {
				item.Action = "skipped"
				item.Reason = "max_age_not_reached"
				res.Summary.Skipped++
				res.Summary.SkippedByReason[item.Reason]++
				res.Items = append(res.Items, item)
				continue
			}
		} else {
			fmt.Printf("kube-slint [orphan-sweep]: warning - failed to parse creation timestamp for pod %s: %v\n", name, err)
		}

		res.Summary.Evaluated++

		if opts.Limit > 0 && len(targetNames) >= opts.Limit {
			item.Action = "skipped"
			item.Reason = "limit_exceeded"
			res.Summary.Skipped++
			res.Summary.SkippedByReason[item.Reason]++
			res.Items = append(res.Items, item)
			hitLimit++
			continue
		}

		targetNames = append(targetNames, name)

		if modeEff == "delete" {
			item.Action = "would-delete"
			item.Reason = "matched"
		} else {
			item.Action = "would-delete"
			item.Reason = "matched"
			res.Summary.WouldDelete++
		}
		res.Items = append(res.Items, item)
	}

	fmt.Printf("kube-slint [orphan-sweep]: mode=%s ns=%s run-id=%s limit=%d maxAge=%v\n", modeEff, ns, runID, opts.Limit, opts.MaxAge)
	fmt.Printf("kube-slint [orphan-sweep]: detected %d matching orphan(s) ", res.Summary.Evaluated)
	if hitLimit > 0 {
		fmt.Printf("(processing %d, skipping %d due to limit)\n", len(targetNames), hitLimit)
	} else {
		fmt.Printf("(processing all)\n")
	}

	if modeEff == "delete" {
		if len(targetNames) == 0 {
			fmt.Printf("kube-slint [orphan-sweep]: no targets to delete\n")
		} else {
			fmt.Printf("kube-slint [orphan-sweep]: proceeding with deletion for %d orphan(s)...\n", len(targetNames))

			args := append([]string{"delete", "pods", "-n", ns, "--ignore-not-found=true"}, targetNames...)
			delCmd := exec.CommandContext(ctx, "kubectl", args...)
			delOut, delErr := delCmd.CombinedOutput()

			if delErr != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("delete command failed: %v", delErr))
				for i := range res.Items {
					if res.Items[i].Action == "would-delete" {
						res.Items[i].Action = "delete-error"
						res.Items[i].Error = fmt.Sprintf("kubectl fail: %v", delErr)
						res.Summary.DeleteError++
					}
				}
				res.FinishedAt = time.Now()
				res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
				return res, fmt.Errorf("failed to delete orphans: %v (output: %s)", delErr, string(delOut))
			} else {
				for i := range res.Items {
					if res.Items[i].Action == "would-delete" {
						res.Items[i].Action = "deleted"
						res.Summary.Deleted++
					}
				}
				fmt.Printf("kube-slint [orphan-sweep]: deletion complete\n")
			}
		}
	} else {
		if len(targetNames) > 0 {
			fmt.Printf("kube-slint [orphan-sweep]: report-only mode, skipped deletion of %v\n", targetNames)
			fmt.Printf("kube-slint [orphan-sweep]: to delete, set option mode='delete'\n")
		}
	}

	res.FinishedAt = time.Now()
	res.DurationMs = res.FinishedAt.Sub(startedAt).Milliseconds()
	return res, nil
}
