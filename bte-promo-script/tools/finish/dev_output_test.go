package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDevOutputScriptOnly(t *testing.T) {
	tmp := t.TempDir()
	host := filepath.Join(tmp, "bte-script")
	if err := os.MkdirAll(host, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROMO_DEV_OUTPUT_DIR", host)

	src := filepath.Join(tmp, "RBM_Betrayal_v1.md")
	if err := os.WriteFile(src, []byte("# script"), 0o644); err != nil {
		t.Fatal(err)
	}

	copyDevOutput("job-1", []labeledFile{
		{Key: "map", Path: filepath.Join(tmp, "map.md")},
		{Key: "script", Path: src},
	})

	dst := filepath.Join(host, "RBM_Betrayal_v1.md")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("script not copied: %v", err)
	}
	if string(data) != "# script" {
		t.Fatalf("unexpected content: %q", data)
	}
	if _, err := os.Stat(filepath.Join(host, "job-1")); err == nil {
		t.Fatal("should not create job subdir")
	}
}

func TestCopyDevOutputDisabled(t *testing.T) {
	t.Setenv("PROMO_DEV_OUTPUT_DIR", "")
	copyDevOutput("job-1", nil)
}
