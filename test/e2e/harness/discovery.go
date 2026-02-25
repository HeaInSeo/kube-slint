package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DiscoveredConfig represents the minimal fields supported in the bridge sprint YAML.
// DiscoveredConfig는 브리지 스프린트 YAML에서 지원하는 최소 필드를 나타냅니다.
type DiscoveredConfig struct {
	Format     string `yaml:"format" json:"format"`
	Strictness struct {
		Mode string `yaml:"mode" json:"mode"`
	} `yaml:"strictness" json:"strictness"`
	Gating struct {
		GateOnLevel string `yaml:"gateOnLevel" json:"gateOnLevel"`
	} `yaml:"gating" json:"gating"`
	Cleanup struct {
		Enabled bool   `yaml:"enabled" json:"enabled"`
		Mode    string `yaml:"mode" json:"mode"`
	} `yaml:"cleanup" json:"cleanup"`
	Write struct {
		ArtifactsDir string `yaml:"artifactsDir" json:"artifactsDir"`
	} `yaml:"write" json:"write"`
	Reliability struct {
		RecordTiming bool `yaml:"recordTiming" json:"recordTiming"`
	} `yaml:"reliability" json:"reliability"`
}

// ConfigSource represents where the configuration was loaded from.
// ConfigSource는 구성이 로드된 위치를 나타냅니다.
type ConfigSource struct {
	Type     string // "injected" | "env" | "discovered"
	Path     string // File path if Type is "env" or "discovered"
	Disabled bool   // True if discovery was disabled via SLINT_DISABLE_DISCOVERY=1
}

// DiscoverConfig searches for and loads the kube-slint configuration.
// It follows these priority rules:
//  1. Environment variable: SLINT_CONFIG_PATH
//  2. Automatic discovery: .slint.yaml or slint.config.yaml (climbing up directories)
//
// DiscoverConfig는 kube-slint 구성을 검색하고 로드합니다.
// 다음과 같은 우선순위 규칙을 따릅니다:
//  1. 환경 변수: SLINT_CONFIG_PATH
//  2. 자동 탐색: .slint.yaml 또는 slint.config.yaml (디렉터리 위로 올라가며 탐색)
func DiscoverConfig(startDir string) (*DiscoveredConfig, ConfigSource, error) {
	if isDiscoveryDisabled() {
		return nil, ConfigSource{Type: "injected", Disabled: true}, nil
	}

	// 1. Check environment variable path
	// 1. 환경 변수 경로 확인
	envPath := os.Getenv("SLINT_CONFIG_PATH")
	if envPath != "" {
		cfg, err := loadConfigFile(envPath)
		if err != nil {
			return nil, ConfigSource{}, fmt.Errorf("failed to load config from SLINT_CONFIG_PATH (%s): %w", envPath, err)
		}
		return cfg, ConfigSource{Type: "env", Path: envPath}, nil
	}

	// 2. Automatic discovery
	// 2. 자동 탐색
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return nil, ConfigSource{}, fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	targetFiles := []string{".slint.yaml", "slint.config.yaml"}
	currentDir := startDir

	// Climb up directories, but stop at root
	// 디렉터리 탐색 종료 조건: 루트 디렉터리에 도달하면 종료
	for {
		for _, target := range targetFiles {
			path := filepath.Join(currentDir, target)
			if fileExists(path) {
				cfg, err := loadConfigFile(path)
				if err != nil {
					return nil, ConfigSource{}, fmt.Errorf("failed to load discovered config from %s: %w", path, err)
				}
				return cfg, ConfigSource{Type: "discovered", Path: path}, nil
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root
			// 루트 도달
			break
		}
		currentDir = parentDir
	}

	// Not found, fallback to injected/default
	// 찾지 못함, 주입/기본값으로 폴백
	return nil, ConfigSource{Type: "injected"}, nil
}

func isDiscoveryDisabled() bool {
	val := os.Getenv("SLINT_DISABLE_DISCOVERY")
	return val == "1" || strings.ToLower(val) == "true"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func loadConfigFile(path string) (*DiscoveredConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg DiscoveredConfig
	
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
	} else {
		// default to yaml
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
