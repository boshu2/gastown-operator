package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyDevOutput saves the W4 script file to a host-mounted directory for local testing.
// LOCAL TEST ONLY — remove before push (see DefaultDevOutputHostPath in promo-api defaults.go).
// Never fails finish/webhook — logs a warning on permission or copy errors.
func copyDevOutput(_ string, labeled []labeledFile) {
	base := strings.TrimSpace(os.Getenv("PROMO_DEV_OUTPUT_DIR"))
	if base == "" {
		return
	}

	var scriptPath string
	for _, lf := range labeled {
		if lf.Key == "script" {
			scriptPath = lf.Path
			break
		}
	}
	if scriptPath == "" {
		fmt.Fprintf(os.Stderr, "promo-finish dev-output: skip (no script artifact in handoff)\n")
		return
	}

	src, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "promo-finish dev-output: WARN resolve script path: %v\n", err)
		return
	}
	dst := filepath.Join(base, filepath.Base(src))

	if err := copyFile(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "promo-finish dev-output: WARN could not save script to %s: %v\n", dst, err)
		fmt.Fprintf(os.Stderr, "promo-finish dev-output: fix: chmod 777 %s on the Docker Desktop host\n", base)
		return
	}
	fmt.Fprintf(os.Stderr, "promo-finish dev-output: W4 script saved => %s\n", dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o777); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
