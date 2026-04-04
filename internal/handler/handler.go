package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"
	"github.com/owenrumney/goreleaser-ls/internal/analysis"
	"github.com/owenrumney/goreleaser-ls/internal/completion"
	"github.com/owenrumney/goreleaser-ls/internal/model"
	"github.com/owenrumney/goreleaser-ls/internal/parser"
	"github.com/owenrumney/goreleaser-ls/internal/schema"
)

// Handler implements the LSP handler interfaces for goreleaser config files.
type Handler struct {
	client *server.Client
	mu     sync.Mutex
	docs   map[lsp.DocumentURI]string
	parsed map[lsp.DocumentURI]*model.Config
}

// New creates a new Handler.
func New() *Handler {
	return &Handler{
		docs:   make(map[lsp.DocumentURI]string),
		parsed: make(map[lsp.DocumentURI]*model.Config),
	}
}

func boolPtr(b bool) *bool { return &b }

// Initialize handles the initialize request.
func (h *Handler) Initialize(_ context.Context, _ *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: boolPtr(true),
				Change:    lsp.SyncFull,
				Save:      &lsp.SaveOptions{IncludeText: boolPtr(false)},
			},
			HoverProvider:          boolPtr(true),
			DocumentSymbolProvider: boolPtr(true),
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{".", "{"},
			},
			DefinitionProvider: boolPtr(true),
			CodeActionProvider: &lsp.CodeActionOptions{
				CodeActionKinds: []lsp.CodeActionKind{lsp.CodeActionQuickFix},
			},
		},
		ServerInfo: &lsp.ServerInfo{
			Name:    "goreleaser-ls",
			Version: "0.1.0",
		},
	}, nil
}

// Shutdown handles the shutdown request.
func (h *Handler) Shutdown(_ context.Context) error {
	return nil
}

// SetClient stores the client for sending notifications.
func (h *Handler) SetClient(client *server.Client) {
	h.client = client
}

// DidOpen handles textDocument/didOpen.
func (h *Handler) DidOpen(ctx context.Context, params *lsp.DidOpenTextDocumentParams) error {
	h.mu.Lock()
	uri := params.TextDocument.URI
	text := params.TextDocument.Text
	h.docs[uri] = text
	h.parsed[uri] = parser.Parse(uri, text)
	cfg := h.parsed[uri]
	h.mu.Unlock()

	updateSchemaVariant(cfg)
	h.publishDiagnostics(ctx, uri, cfg)
	return nil
}

// DidChange handles textDocument/didChange.
func (h *Handler) DidChange(ctx context.Context, params *lsp.DidChangeTextDocumentParams) error {
	h.mu.Lock()
	uri := params.TextDocument.URI
	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[len(params.ContentChanges)-1].Text
		h.docs[uri] = text
		h.parsed[uri] = parser.Parse(uri, text)
	}
	cfg := h.parsed[uri]
	h.mu.Unlock()

	updateSchemaVariant(cfg)
	h.publishDiagnostics(ctx, uri, cfg)
	return nil
}

// DidClose handles textDocument/didClose.
func (h *Handler) DidClose(_ context.Context, params *lsp.DidCloseTextDocumentParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.docs, params.TextDocument.URI)
	delete(h.parsed, params.TextDocument.URI)
	return nil
}

// DidSave handles textDocument/didSave — runs diagnostics.
func (h *Handler) DidSave(ctx context.Context, params *lsp.DidSaveTextDocumentParams) error {
	h.mu.Lock()
	uri := params.TextDocument.URI
	cfg := h.parsed[uri]
	h.mu.Unlock()

	h.publishDiagnostics(ctx, uri, cfg)
	return nil
}

func updateSchemaVariant(cfg *model.Config) {
	if cfg == nil {
		return
	}
	for _, n := range cfg.Nodes {
		if n.Key == "pro" && n.Value == "true" {
			schema.UsePro()
			return
		}
	}
	schema.UseOSS()
}

func (h *Handler) publishDiagnostics(ctx context.Context, uri lsp.DocumentURI, cfg *model.Config) {
	if cfg == nil || h.client == nil {
		return
	}

	diags := analysis.Diagnose(cfg)
	if diags == nil {
		diags = []lsp.Diagnostic{}
	}
	_ = h.client.PublishDiagnostics(ctx, &lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

// Hover handles textDocument/hover.
func (h *Handler) Hover(_ context.Context, params *lsp.HoverParams) (*lsp.Hover, error) {
	h.mu.Lock()
	cfg := h.parsed[params.TextDocument.URI]
	h.mu.Unlock()

	if cfg == nil {
		return nil, nil
	}

	node := cfg.FindNodeAtPosition(params.Position)
	if node == nil {
		return nil, nil
	}

	// Look up schema docs for this key path.
	f := schema.Lookup(node.Path...)
	if f == nil && len(node.Path) > 0 {
		// Try the parent path + key for sequence items.
		f = schema.Lookup(node.Path[:len(node.Path)-1]...)
	}

	if f != nil {
		var sb strings.Builder
		fmt.Fprintf(&sb, "**`%s`**\n\n%s", node.Key, f.Doc)

		if f.Type == schema.TypeEnum && len(f.EnumValues) > 0 {
			sb.WriteString("\n\n**Values:** ")
			sb.WriteString(strings.Join(f.EnumValues, ", "))
		}

		if f.Deprecated != "" {
			sb.WriteString("\n\n**Deprecated:** ")
			sb.WriteString(f.Deprecated)
			if f.Replacement != "" {
				fmt.Fprintf(&sb, " Use `%s` instead.", f.Replacement)
			}
		}

		return &lsp.Hover{
			Contents: lsp.MarkupContent{
				Kind:  "markdown",
				Value: sb.String(),
			},
		}, nil
	}

	return nil, nil
}

// Completion handles textDocument/completion.
func (h *Handler) Completion(_ context.Context, params *lsp.CompletionParams) (*lsp.CompletionList, error) {
	h.mu.Lock()
	cfg := h.parsed[params.TextDocument.URI]
	text := h.docs[params.TextDocument.URI]
	h.mu.Unlock()

	items := completion.Complete(cfg, text, params.Position)
	if items == nil {
		items = []lsp.CompletionItem{}
	}
	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// Definition handles textDocument/definition.
func (h *Handler) Definition(_ context.Context, params *lsp.DefinitionParams) ([]lsp.Location, error) {
	h.mu.Lock()
	uri := params.TextDocument.URI
	cfg := h.parsed[uri]
	h.mu.Unlock()

	if cfg == nil {
		return nil, nil
	}

	node := cfg.FindNodeAtPosition(params.Position)
	if node == nil {
		return nil, nil
	}

	// If cursor is on a scalar value under an "ids" key, try to jump to the build definition.
	if node.IsScalar && len(node.Path) > 0 {
		parentKey := node.Path[len(node.Path)-1]
		if parentKey == "ids" {
			return h.findBuildByID(uri, cfg, node.Value), nil
		}
	}

	// If cursor is on an ID key, find all references to it.
	if node.Key == "id" && node.IsScalar && len(node.Path) > 0 {
		return []lsp.Location{{URI: uri, Range: node.KeyRange}}, nil
	}

	return nil, nil
}

func (h *Handler) findBuildByID(uri lsp.DocumentURI, cfg *model.Config, id string) []lsp.Location {
	for _, n := range cfg.Nodes {
		if n.Key == "builds" {
			for _, child := range n.Children {
				if child.Key == "id" && child.IsScalar && child.Value == id {
					return []lsp.Location{{URI: uri, Range: child.KeyRange}}
				}
			}
		}
	}
	return nil
}

// DocumentSymbol handles textDocument/documentSymbol.
func (h *Handler) DocumentSymbol(_ context.Context, params *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	h.mu.Lock()
	cfg := h.parsed[params.TextDocument.URI]
	h.mu.Unlock()

	if cfg == nil {
		return nil, nil
	}

	var symbols []lsp.DocumentSymbol
	for _, n := range cfg.Nodes {
		sym := nodeToSymbol(n)
		symbols = append(symbols, sym)
	}
	return symbols, nil
}

func nodeToSymbol(n *model.Node) lsp.DocumentSymbol {
	kind := lsp.SymbolKindProperty
	if len(n.Children) > 0 {
		kind = lsp.SymbolKindObject
	}

	name := n.Key
	if name == "" {
		name = n.Value
	}
	if name == "" {
		name = "-"
	}

	sym := lsp.DocumentSymbol{
		Name:           name,
		Detail:         n.Value,
		Kind:           kind,
		Range:          n.Range,
		SelectionRange: n.KeyRange,
	}

	for _, child := range n.Children {
		sym.Children = append(sym.Children, nodeToSymbol(child))
	}

	return sym
}

// CodeAction handles textDocument/codeAction.
func (h *Handler) CodeAction(_ context.Context, params *lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	h.mu.Lock()
	cfg := h.parsed[params.TextDocument.URI]
	h.mu.Unlock()

	if cfg == nil {
		return nil, nil
	}

	var actions []lsp.CodeAction
	kind := lsp.CodeActionQuickFix

	for _, diag := range params.Context.Diagnostics {
		if diag.Source != "goreleaser-ls" {
			continue
		}

		// Fix deprecated field names.
		if strings.Contains(diag.Message, "is deprecated") {
			action := deprecationFix(params.TextDocument.URI, cfg, diag)
			if action != nil {
				action.Kind = &kind
				actions = append(actions, *action)
			}
		}
	}

	return actions, nil
}

func deprecationFix(uri lsp.DocumentURI, cfg *model.Config, diag lsp.Diagnostic) *lsp.CodeAction {
	// Find the node at the diagnostic range.
	node := cfg.FindNodeAtPosition(diag.Range.Start)
	if node == nil {
		return nil
	}

	f := schema.Lookup(node.Path...)
	if f == nil || f.Replacement == "" {
		return nil
	}

	return &lsp.CodeAction{
		Title:       fmt.Sprintf("Replace `%s` with `%s`", f.Key, f.Replacement),
		Diagnostics: []lsp.Diagnostic{diag},
		IsPreferred: boolPtr(true),
		Edit: &lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				uri: {{
					Range:   node.KeyRange,
					NewText: f.Replacement,
				}},
			},
		},
	}
}
