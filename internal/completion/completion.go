package completion

import (
	"strings"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/goreleaser-ls/internal/model"
	"github.com/owenrumney/goreleaser-ls/internal/schema"
)

func kindPtr(k lsp.CompletionItemKind) *lsp.CompletionItemKind { return &k }

// Complete returns completion items for the given position.
func Complete(cfg *model.Config, text string, pos lsp.Position) []lsp.CompletionItem {
	line := lineAt(text, pos.Line)

	// Template variable completion inside {{ .
	if idx := templateTrigger(line, pos.Character); idx >= 0 {
		return templateCompletions()
	}

	// Determine the YAML path context at cursor.
	path := pathAtPosition(cfg, text, pos)
	fields := schema.ChildKeys(path...)

	if fields == nil && len(path) == 0 {
		fields = schema.TopLevelFields()
	}

	// Filter out keys that already exist at this level.
	existing := existingKeysAtPath(cfg, path)

	var items []lsp.CompletionItem
	for _, f := range fields {
		if existing[f.Key] {
			continue
		}

		detail := f.Doc
		if f.Deprecated != "" {
			detail = "**Deprecated:** " + f.Deprecated + "\n\n" + detail
		}

		items = append(items, lsp.CompletionItem{
			Label:  f.Key,
			Kind:   kindPtr(completionKindForField(f)),
			Detail: detail,
		})
	}

	return items
}

func templateCompletions() []lsp.CompletionItem {
	var items []lsp.CompletionItem
	for _, tv := range schema.TemplateVars {
		items = append(items, lsp.CompletionItem{
			Label:  tv.Name,
			Kind:   kindPtr(lsp.CompletionItemKindVariable),
			Detail: tv.Doc,
		})
	}
	return items
}

func completionKindForField(f *schema.Field) lsp.CompletionItemKind {
	switch f.Type {
	case schema.TypeObject, schema.TypeMap:
		return lsp.CompletionItemKindModule
	case schema.TypeList:
		return lsp.CompletionItemKindEnum
	case schema.TypeEnum:
		return lsp.CompletionItemKindEnumMember
	case schema.TypeBool:
		return lsp.CompletionItemKindValue
	default:
		return lsp.CompletionItemKindField
	}
}

// pathAtPosition determines the YAML key path at the given cursor position.
func pathAtPosition(cfg *model.Config, text string, pos lsp.Position) []string {
	if cfg == nil {
		return nil
	}

	node := cfg.FindNodeAtPosition(pos)
	if node == nil {
		// Check indentation to infer parent.
		line := lineAt(text, pos.Line)
		indent := countIndent(line)
		if indent == 0 {
			return nil
		}
		// Walk backwards to find the parent key.
		return parentPathFromIndent(text, pos.Line, indent)
	}
	return node.Path
}

// parentPathFromIndent walks backwards through lines to find the parent key path.
func parentPathFromIndent(text string, fromLine int, targetIndent int) []string {
	lines := strings.Split(text, "\n")
	var path []string

	currentIndent := targetIndent
	for i := fromLine - 1; i >= 0; i-- {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineIndent := countIndent(line)
		if lineIndent < currentIndent {
			key := extractKey(line)
			if key != "" {
				path = append([]string{key}, path...)
				currentIndent = lineIndent
			}
			if lineIndent == 0 {
				break
			}
		}
	}

	return path
}

func existingKeysAtPath(cfg *model.Config, path []string) map[string]bool {
	existing := make(map[string]bool)
	if cfg == nil {
		return existing
	}

	var nodes []*model.Node
	if len(path) == 0 {
		nodes = cfg.Nodes
	} else {
		parent := cfg.FindNodeByPath(path...)
		if parent != nil {
			nodes = parent.Children
		}
	}

	for _, n := range nodes {
		existing[n.Key] = true
	}
	return existing
}

func lineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return lines[line]
}

func countIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

func extractKey(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed, _ = strings.CutPrefix(trimmed, "- ")
	if strings.HasPrefix(trimmed, "#") {
		return ""
	}
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		return trimmed[:idx]
	}
	return ""
}

// templateTrigger checks if the cursor is in a template expression like {{ .
// Returns the index of the dot, or -1.
func templateTrigger(line string, col int) int {
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]
	// Look for {{ . pattern
	for i := len(prefix) - 1; i >= 2; i-- {
		if prefix[i] == '.' {
			before := strings.TrimSpace(prefix[:i])
			if strings.HasSuffix(before, "{{") {
				return i
			}
		}
		// Stop at closing braces.
		if prefix[i] == '}' {
			break
		}
	}
	return -1
}
