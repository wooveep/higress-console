package portal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var aiSensitiveDefaultDictionaryCandidates = []string{
	"resource/public/ai-sensitive/sensitive_word_dict.txt",
	"backend/resource/public/ai-sensitive/sensitive_word_dict.txt",
}

func DefaultAISensitiveDictionaryText() string {
	seen := map[string]struct{}{}
	for _, candidate := range aiSensitiveDefaultDictionaryCandidates {
		for _, path := range aiSensitiveDictionaryFileCandidates(candidate) {
			cleaned := filepath.Clean(path)
			if _, ok := seen[cleaned]; ok {
				continue
			}
			seen[cleaned] = struct{}{}
			bytes, err := os.ReadFile(cleaned)
			if err != nil {
				continue
			}
			content := strings.TrimSpace(string(bytes))
			if content != "" {
				return content
			}
		}
	}
	return ""
}

func aiSensitiveDictionaryFileCandidates(path string) []string {
	candidates := []string{
		path,
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
		filepath.Join("..", "..", "..", path),
		filepath.Join("..", "..", "..", "..", path),
		filepath.Join("..", "..", "..", "..", "..", path),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates,
			filepath.Join(base, path),
			filepath.Join(base, "..", path),
			filepath.Join(base, "..", "..", path),
			filepath.Join(base, "..", "..", "..", path),
			filepath.Join(base, "..", "..", "..", "..", path),
			filepath.Join(base, "..", "..", "..", "..", "..", path),
		)
	}
	return candidates
}
