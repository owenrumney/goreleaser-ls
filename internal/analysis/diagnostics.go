package analysis

import (
	"fmt"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/goreleaser-ls/internal/model"
	"github.com/owenrumney/goreleaser-ls/internal/schema"
)

// Diagnose runs all diagnostic checks on the config.
func Diagnose(cfg *model.Config) []lsp.Diagnostic {
	if cfg == nil {
		return nil
	}

	var diags []lsp.Diagnostic
	diags = append(diags, checkMissingVersion(cfg)...)
	diags = append(diags, checkUnknownKeys(cfg)...)
	diags = append(diags, checkDeprecatedKeys(cfg)...)
	diags = append(diags, checkTemplateVars(cfg)...)
	diags = append(diags, checkIDReferences(cfg)...)
	return diags
}

func severityPtr(s lsp.DiagnosticSeverity) *lsp.DiagnosticSeverity { return &s }

func checkMissingVersion(cfg *model.Config) []lsp.Diagnostic {
	for _, n := range cfg.Nodes {
		if n.Key == "version" {
			if n.Value != "2" {
				return []lsp.Diagnostic{{
					Range:    n.KeyRange,
					Severity: severityPtr(lsp.SeverityError),
					Source:   "goreleaser-ls",
					Message:  "version must be 2",
				}}
			}
			return nil
		}
	}

	return []lsp.Diagnostic{{
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 1},
		},
		Severity: severityPtr(lsp.SeverityError),
		Source:   "goreleaser-ls",
		Message:  "missing required field: version (must be 2)",
	}}
}

func checkUnknownKeys(cfg *model.Config) []lsp.Diagnostic {
	var diags []lsp.Diagnostic
	checkUnknownKeysRecursive(cfg.Nodes, nil, &diags)
	return diags
}

func checkUnknownKeysRecursive(nodes []*model.Node, parentPath []string, diags *[]lsp.Diagnostic) {
	validFields := schema.ChildKeys(parentPath...)
	if validFields == nil && len(parentPath) > 0 {
		return
	}
	if validFields == nil {
		validFields = schema.TopLevel
	}

	fieldMap := make(map[string]bool, len(validFields))
	for _, f := range validFields {
		fieldMap[f.Key] = true
	}

	for _, n := range nodes {
		if n.IsScalar && len(n.Path) > 0 && n.Path[len(n.Path)-1] != n.Key {
			continue
		}

		if !fieldMap[n.Key] && len(parentPath) == 0 {
			*diags = append(*diags, lsp.Diagnostic{
				Range:    n.KeyRange,
				Severity: severityPtr(lsp.SeverityWarning),
				Source:   "goreleaser-ls",
				Message:  fmt.Sprintf("unknown top-level key: %s", n.Key),
			})
		}

		if len(n.Children) > 0 {
			checkUnknownKeysRecursive(n.Children, n.Path, diags)
		}
	}
}

func checkDeprecatedKeys(cfg *model.Config) []lsp.Diagnostic {
	var diags []lsp.Diagnostic
	checkDeprecatedRecursive(cfg.Nodes, nil, &diags)
	return diags
}

func checkDeprecatedRecursive(nodes []*model.Node, parentPath []string, diags *[]lsp.Diagnostic) {
	for _, n := range nodes {
		path := append(append([]string{}, parentPath...), n.Key)
		f := schema.Lookup(path...)
		if f != nil && f.Deprecated != "" {
			msg := fmt.Sprintf("`%s` is deprecated: %s", n.Key, f.Deprecated)
			if f.Replacement != "" {
				msg += fmt.Sprintf(" Use `%s` instead.", f.Replacement)
			}
			*diags = append(*diags, lsp.Diagnostic{
				Range:    n.KeyRange,
				Severity: severityPtr(lsp.SeverityWarning),
				Source:   "goreleaser-ls",
				Message:  msg,
				Tags:     []lsp.DiagnosticTag{lsp.TagDeprecated},
			})
		}

		if len(n.Children) > 0 {
			checkDeprecatedRecursive(n.Children, path, diags)
		}
	}
}

func checkTemplateVars(cfg *model.Config) []lsp.Diagnostic {
	validVars := make(map[string]bool, len(schema.TemplateVars))
	for _, tv := range schema.TemplateVars {
		validVars[tv.Name] = true
	}
	// Also allow Env.* access.
	validVars["Env"] = true

	var diags []lsp.Diagnostic
	for _, n := range cfg.AllNodes() {
		for _, ref := range n.TemplateRefs {
			// Allow dotted paths like Runtime.Goos — check the full name and the prefix.
			if validVars[ref.Name] {
				continue
			}
			// Check prefix for nested access (e.g. Env.GITHUB_TOKEN).
			parts := splitFirst(ref.Name, '.')
			if validVars[parts] {
				continue
			}
			diags = append(diags, lsp.Diagnostic{
				Range:    ref.Range,
				Severity: severityPtr(lsp.SeverityWarning),
				Source:   "goreleaser-ls",
				Message:  fmt.Sprintf("unknown template variable: .%s", ref.Name),
			})
		}
	}
	return diags
}

func splitFirst(s string, sep byte) string {
	for i := range len(s) {
		if s[i] == sep {
			return s[:i]
		}
	}
	return s
}

// idReferenceSections are top-level keys whose items can reference build IDs via "ids".
var idReferenceSections = []string{
	"archives", "nfpms", "snapcrafts", "dockers_v2", "signs", "binary_signs",
	"docker_signs", "sboms", "universal_binaries", "upx",
}

func checkIDReferences(cfg *model.Config) []lsp.Diagnostic {
	// Collect defined build IDs.
	definedIDs := make(map[string]bool)
	for _, n := range cfg.Nodes {
		if n.Key == "builds" {
			for _, child := range n.Children {
				if child.Key == "id" && child.IsScalar {
					definedIDs[child.Value] = true
				}
			}
		}
	}

	if len(definedIDs) == 0 {
		return nil
	}

	var diags []lsp.Diagnostic
	for _, sectionKey := range idReferenceSections {
		for _, n := range cfg.Nodes {
			if n.Key != sectionKey {
				continue
			}
			for _, child := range n.Children {
				if child.Key != "ids" {
					continue
				}
				for _, idNode := range child.Children {
					if idNode.IsScalar && !definedIDs[idNode.Value] {
						diags = append(diags, lsp.Diagnostic{
							Range:    idNode.KeyRange,
							Severity: severityPtr(lsp.SeverityWarning),
							Source:   "goreleaser-ls",
							Message:  fmt.Sprintf("references unknown build ID: %s", idNode.Value),
						})
					}
				}
			}
		}
	}
	return diags
}
