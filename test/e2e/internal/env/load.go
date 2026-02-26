package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadOptions 는 환경 변수에서 로드된 E2E 테스트 설정을 포함하는 Options를 반환함.
func LoadOptions() Options {
	return Options{
		Enabled: boolEnv("SLOLAB_ENABLED", false),

		ArtifactsDir: stringEnv("ARTIFACTS_DIR", "/tmp"),
		RunID:        stringEnv("CI_RUN_ID", ""),

		SkipCleanup:            boolEnv("E2E_SKIP_CLEANUP", false),
		SkipCertManagerInstall: boolEnv("CERT_MANAGER_INSTALL_SKIP", false),

		TokenRequestTimeout: durationEnv("TOKEN_REQUEST_TIMEOUT", 2*time.Minute),
	}
}

// --- 헬퍼 함수 (규칙 통일: "1"/"true"/"yes"/"on" 모두 허용) ---

// stringEnv는 환경 변수를 문자열로 반환함.
func stringEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

// boolEnv는 환경 변수를 bool로 파싱함.
func boolEnv(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return def
	}
}

// durationEnv는 환경 변수를 time.Duration으로 파싱함. 숫자만 입력되면 초 단위로 간주함.
func durationEnv(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	return def
}
