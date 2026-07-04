package dataplane

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlDocSeparator matches a "---" document separator line (optionally
// trailed by a comment), used to split a multi-document YAML file into
// independently-decodable chunks. Splitting ourselves — rather than making
// repeated Decode calls on one shared yaml.Decoder — is deliberate: once a
// yaml.Decoder's stream hits a parse error, further Decode calls on that
// same Decoder keep returning the same error without ever reaching EOF,
// which turns "one malformed document, keep going" into an infinite loop.
// Decoding each chunk independently (a fresh parse per chunk) avoids that
// entirely, at the acceptable cost of not handling a literal "---" line
// inside a block scalar (an edge case, not real-world manifest shape).
var yamlDocSeparator = regexp.MustCompile(`(?m)^---[ \t]*(#.*)?\r?$`)

var workloadKinds = map[string]bool{
	"Deployment":  true,
	"StatefulSet": true,
	"DaemonSet":   true,
}

// LoadWarning records a single document that failed to load; loading
// continues past it. Only directory-level I/O failure is a fatal error.
type LoadWarning struct {
	File     string
	DocIndex int
	Err      error
}

func (w LoadWarning) String() string {
	return fmt.Sprintf("%s (doc #%d): %v", w.File, w.DocIndex, w.Err)
}

// LoadDir walks dir for *.yaml/*.yml files (sorted for determinism), splits
// each file's multi-document YAML stream, and classifies every document by
// Kind into a Bundle. A per-document decode failure becomes a LoadWarning
// (loading continues); only a directory-level I/O failure (dir missing or
// unreadable) is a fatal error.
func LoadDir(dir string) (*Bundle, []LoadWarning, error) {
	files, err := collectYAMLFiles(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read manifest dir %q: %w", dir, err)
	}

	b := &Bundle{}
	var warnings []LoadWarning
	for _, relPath := range files {
		data, err := os.ReadFile(filepath.Join(dir, relPath))
		if err != nil {
			warnings = append(warnings, LoadWarning{File: relPath, Err: err})
			continue
		}
		warnings = append(warnings, loadDocuments(b, relPath, data)...)
	}
	return b, warnings, nil
}

func collectYAMLFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func loadDocuments(b *Bundle, relPath string, data []byte) []LoadWarning {
	var warnings []LoadWarning
	chunks := yamlDocSeparator.Split(string(data), -1)
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue // empty document (e.g. leading/trailing "---")
		}
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(chunk), &node); err != nil {
			warnings = append(warnings, LoadWarning{File: relPath, DocIndex: i, Err: err})
			continue
		}
		if node.Kind == 0 || len(node.Content) == 0 {
			continue
		}
		if err := classifyDoc(b, &node, relPath); err != nil {
			warnings = append(warnings, LoadWarning{File: relPath, DocIndex: i, Err: err})
		}
	}
	return warnings
}

func classifyDoc(b *Bundle, node *yaml.Node, relPath string) error {
	var sniff struct {
		Kind string `yaml:"kind"`
	}
	if err := node.Decode(&sniff); err != nil {
		return fmt.Errorf("sniff kind: %w", err)
	}

	switch {
	case workloadKinds[sniff.Kind]:
		var w Workload
		if err := node.Decode(&w); err != nil {
			return fmt.Errorf("decode %s: %w", sniff.Kind, err)
		}
		w.SourceFile = relPath
		b.Workloads = append(b.Workloads, w)
	case sniff.Kind == "Service":
		var s Service
		if err := node.Decode(&s); err != nil {
			return fmt.Errorf("decode Service: %w", err)
		}
		s.SourceFile = relPath
		b.Services = append(b.Services, s)
	case sniff.Kind == "ServiceMonitor":
		var sm ServiceMonitor
		if err := node.Decode(&sm); err != nil {
			return fmt.Errorf("decode ServiceMonitor: %w", err)
		}
		sm.SourceFile = relPath
		b.ServiceMonitors = append(b.ServiceMonitors, sm)
	default:
		var meta struct {
			APIVersion string     `yaml:"apiVersion"`
			Metadata   ObjectMeta `yaml:"metadata"`
		}
		_ = node.Decode(&meta) // best-effort; unknown objects are never an error
		b.Unknown = append(b.Unknown, UnknownObject{
			APIVersion: meta.APIVersion,
			Kind:       sniff.Kind,
			Name:       meta.Metadata.Name,
			SourceFile: relPath,
		})
	}
	return nil
}
