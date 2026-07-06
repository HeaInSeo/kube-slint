package fixtures

import "os"

func bad(path string, force bool) error {
	if _, statErr := os.Stat(path); statErr == nil && !force {
		return nil
	}
	// ruleid: kube-slint-no-stat-before-write
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return err
	}
	return nil
}

func badSummaryWrite(path string, force bool) error {
	if _, statErr := os.Stat(path); statErr == nil && !force {
		return nil
	}
	// ruleid: kube-slint-no-stat-before-write
	return summary.WriteFile(path, s)
}

func good(path string) error {
	// ok: kube-slint-no-stat-before-write
	return os.WriteFile(path, nil, 0o644)
}
