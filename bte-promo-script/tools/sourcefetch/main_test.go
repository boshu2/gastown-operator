package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocxTextLocal(t *testing.T) {
	path := os.Getenv("TEST_DOCX_PATH")
	if path == "" {
		t.Skip("set TEST_DOCX_PATH to run")
	}
	text, err := docxText(path)
	require.NoError(t, err)
	require.NotEmpty(t, text)
}

func TestFetchAndExtract(t *testing.T) {
	docxPath := os.Getenv("TEST_DOCX_PATH")
	if docxPath == "" {
		t.Skip("set TEST_DOCX_PATH to run")
	}
	raw, err := os.ReadFile(docxPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "ep01.txt")
	require.NoError(t, fetchAndExtract(srv.Client(), srv.URL, out))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
