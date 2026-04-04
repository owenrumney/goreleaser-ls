package handler

import (
	"context"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T, uri lsp.DocumentURI, text string) *Handler {
	t.Helper()
	h := New()
	err := h.DidOpen(context.Background(), &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "yaml",
			Text:       text,
		},
	})
	require.NoError(t, err)
	return h
}

func TestInitialize(t *testing.T) {
	h := New()
	result, err := h.Initialize(context.Background(), &lsp.InitializeParams{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "goreleaser-ls", result.ServerInfo.Name)
	assert.NotNil(t, result.Capabilities.HoverProvider)
	assert.NotNil(t, result.Capabilities.CompletionProvider)
}

func TestDidOpenAndClose(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\n")

	h.mu.Lock()
	_, ok := h.parsed[uri]
	h.mu.Unlock()
	assert.True(t, ok)

	err := h.DidClose(context.Background(), &lsp.DidCloseTextDocumentParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)

	h.mu.Lock()
	_, ok = h.parsed[uri]
	h.mu.Unlock()
	assert.False(t, ok)
}

func TestDidChange(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\n")

	err := h.DidChange(context.Background(), &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{URI: uri},
		},
		ContentChanges: []lsp.TextDocumentContentChangeEvent{
			{Text: "version: 2\nproject_name: updated\n"},
		},
	})
	require.NoError(t, err)

	h.mu.Lock()
	text := h.docs[uri]
	h.mu.Unlock()
	assert.Contains(t, text, "updated")
}

func TestHover_KnownKey(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\nbuilds:\n  - main: .\n")

	result, err := h.Hover(context.Background(), &lsp.HoverParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 2},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Contents.Value, "builds")
	assert.Contains(t, result.Contents.Value, "Builds configuration")
}

func TestHover_DeprecatedKey(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\nbrews:\n  - name: test\n")

	result, err := h.Hover(context.Background(), &lsp.HoverParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 2},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Contents.Value, "Deprecated")
}

func TestHover_UnknownPosition(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\n\n\n")

	result, err := h.Hover(context.Background(), &lsp.HoverParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 2, Character: 0},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCompletion_TopLevel(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\n")

	result, err := h.Completion(context.Background(), &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 1, Character: 0},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Items)

	labels := make(map[string]bool)
	for _, item := range result.Items {
		labels[item.Label] = true
	}
	assert.True(t, labels["builds"])
	assert.False(t, labels["version"])
}

func TestDocumentSymbol(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\nbuilds:\n  - main: .\nrelease:\n  draft: true\n")

	result, err := h.DocumentSymbol(context.Background(), &lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)

	names := make(map[string]bool)
	for _, sym := range result {
		names[sym.Name] = true
	}
	assert.True(t, names["version"])
	assert.True(t, names["builds"])
	assert.True(t, names["release"])
}

func TestCodeAction_DeprecatedFix(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\nbrews:\n  - name: test\n")

	h.mu.Lock()
	cfg := h.parsed[uri]
	h.mu.Unlock()

	brewsNode := cfg.Nodes[1]

	actions, err := h.CodeAction(context.Background(), &lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Context: lsp.CodeActionContext{
			Diagnostics: []lsp.Diagnostic{{
				Range:   brewsNode.KeyRange,
				Source:  "goreleaser-ls",
				Message: "`brews` is deprecated: Use homebrew_casks. Use `homebrew_casks` instead.",
			}},
		},
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Contains(t, actions[0].Title, "homebrew_casks")
	assert.NotNil(t, actions[0].Edit)
}

func TestDefinition_IDReference(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	input := `version: 2
builds:
  - id: myapp
    main: .
archives:
  - ids:
      - myapp
`
	h := newTestHandler(t, uri, input)

	// Cursor on "myapp" in the ids list (line 6, within the value).
	result, err := h.Definition(context.Background(), &lsp.DefinitionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 6, Character: 8},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)
	// Should point to the build's "id: myapp" line.
	assert.Equal(t, uri, result[0].URI)
	assert.Equal(t, 2, result[0].Range.Start.Line)
}

func TestDefinition_NoMatch(t *testing.T) {
	uri := lsp.DocumentURI("file:///test.yml")
	h := newTestHandler(t, uri, "version: 2\nproject_name: test\n")

	result, err := h.Definition(context.Background(), &lsp.DefinitionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 0, Character: 2},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInitialize_HasDefinitionProvider(t *testing.T) {
	h := New()
	result, err := h.Initialize(context.Background(), &lsp.InitializeParams{})
	require.NoError(t, err)
	assert.NotNil(t, result.Capabilities.DefinitionProvider)
}
