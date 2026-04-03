package completion

import (
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/goreleaser-ls/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplete_TopLevelKeys(t *testing.T) {
	input := `version: 2
`
	cfg := parser.Parse("file:///test.yml", input)
	items := Complete(cfg, input, lsp.Position{Line: 1, Character: 0})

	require.NotEmpty(t, items)

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	assert.True(t, labels["builds"])
	assert.True(t, labels["archives"])
	assert.True(t, labels["release"])
	assert.False(t, labels["version"]) // already exists
}

func TestComplete_NestedKeys(t *testing.T) {
	input := `version: 2
release:
  `
	cfg := parser.Parse("file:///test.yml", input)
	items := Complete(cfg, input, lsp.Position{Line: 2, Character: 2})

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	assert.True(t, labels["github"])
	assert.True(t, labels["draft"])
}

func TestComplete_TemplateVars(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "{{ .`
	cfg := parser.Parse("file:///test.yml", input)
	items := Complete(cfg, input, lsp.Position{Line: 2, Character: 20})

	require.NotEmpty(t, items)

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}

	assert.True(t, labels["Version"])
	assert.True(t, labels["Tag"])
	assert.True(t, labels["ProjectName"])
}

func TestComplete_ExcludesExistingKeys(t *testing.T) {
	input := `version: 2
project_name: test
`
	cfg := parser.Parse("file:///test.yml", input)
	items := Complete(cfg, input, lsp.Position{Line: 2, Character: 0})

	for _, item := range items {
		assert.NotEqual(t, "version", item.Label)
		assert.NotEqual(t, "project_name", item.Label)
	}
}

func TestComplete_EmptyFile(t *testing.T) {
	input := ""
	cfg := parser.Parse("file:///test.yml", input)
	items := Complete(cfg, input, lsp.Position{Line: 0, Character: 0})

	require.NotEmpty(t, items)

	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	assert.True(t, labels["version"])
}
