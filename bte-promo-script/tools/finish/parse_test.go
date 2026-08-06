package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileArgs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := parseFileArgs([]string{
			"map=working files/map.md",
			"script=output/script.md",
		})
		require.NoError(t, err)
		assert.Equal(t, []labeledFile{
			{Key: "map", Path: "working files/map.md"},
			{Key: "script", Path: "output/script.md"},
		}, got)
	})

	t.Run("missing equals", func(t *testing.T) {
		_, err := parseFileArgs([]string{"script"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key=path")
	})

	t.Run("duplicate key", func(t *testing.T) {
		_, err := parseFileArgs([]string{"script=a.md", "script=b.md"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key")
	})

	t.Run("empty args", func(t *testing.T) {
		_, err := parseFileArgs(nil)
		require.Error(t, err)
	})
}
