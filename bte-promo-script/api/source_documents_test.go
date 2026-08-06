package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSourceDocuments(t *testing.T) {
	t.Run("empty ok", func(t *testing.T) {
		got, err := normalizeSourceDocuments(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("valid https", func(t *testing.T) {
		urls := []string{
			"https://example.com/a.docx",
			"https://example.com/b.docx",
		}
		got, err := normalizeSourceDocuments(urls)
		require.NoError(t, err)
		assert.Equal(t, urls, got)
	})

	t.Run("reject file scheme", func(t *testing.T) {
		_, err := normalizeSourceDocuments([]string{"file:///tmp/x.docx"})
		require.Error(t, err)
	})

	t.Run("reject duplicate", func(t *testing.T) {
		_, err := normalizeSourceDocuments([]string{
			"https://example.com/a.docx",
			"https://example.com/a.docx",
		})
		require.Error(t, err)
	})
}

func TestSourceDocumentsAnnotation(t *testing.T) {
	raw, err := sourceDocumentsAnnotation([]string{"https://a.test/1.docx"})
	require.NoError(t, err)
	assert.Contains(t, raw, "https://a.test/1.docx")
}
