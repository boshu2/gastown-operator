package main

import (
	"fmt"
	"strings"
)

type labeledFile struct {
	Key  string
	Path string
}

// parseFileArgs turns CLI args "script=output/script.md metadata=output/meta.json"
// into labeled files. Keys are defined by the static workflow (step → output name).
func parseFileArgs(args []string) ([]labeledFile, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("at least one key=path argument is required")
	}

	out := make([]labeledFile, 0, len(args))
	seen := make(map[string]struct{}, len(args))
	for _, arg := range args {
		key, path, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=path, got %q (keys come from scripts/PROMO_FINISH.md)", arg)
		}
		key = strings.TrimSpace(key)
		path = strings.TrimSpace(path)
		if key == "" || path == "" {
			return nil, fmt.Errorf("invalid key=path: %q", arg)
		}
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		seen[key] = struct{}{}
		out = append(out, labeledFile{Key: key, Path: path})
	}
	return out, nil
}
