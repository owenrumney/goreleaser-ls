package parser

import (
	"regexp"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/goreleaser-ls/internal/model"
	"gopkg.in/yaml.v3"
)

var templateRefRe = regexp.MustCompile(`\{\{\s*\.(\w+(?:\.\w+)*)\s*\}\}`)

// Parse parses YAML text into a Config with positional information.
func Parse(uri lsp.DocumentURI, text string) *model.Config {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return &model.Config{URI: uri}
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return &model.Config{URI: uri}
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return &model.Config{URI: uri}
	}

	nodes := parseMappingNode(root, nil)
	return &model.Config{
		URI:   uri,
		Nodes: nodes,
	}
}

func parseMappingNode(node *yaml.Node, parentPath []string) []*model.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	var nodes []*model.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		path := append(append([]string{}, parentPath...), keyNode.Value)

		n := &model.Node{
			Key:  keyNode.Value,
			Path: path,
			KeyRange: lsp.Range{
				Start: lsp.Position{Line: keyNode.Line - 1, Character: keyNode.Column - 1},
				End:   lsp.Position{Line: keyNode.Line - 1, Character: keyNode.Column - 1 + len(keyNode.Value)},
			},
			Range: nodeRange(keyNode, valNode),
		}

		switch valNode.Kind {
		case yaml.ScalarNode:
			n.Value = valNode.Value
			n.IsScalar = true
			n.ValueRange = lsp.Range{
				Start: lsp.Position{Line: valNode.Line - 1, Character: valNode.Column - 1},
				End:   lsp.Position{Line: valNode.Line - 1, Character: valNode.Column - 1 + len(valNode.Value)},
			}
			n.TemplateRefs = extractTemplateRefs(valNode)
		case yaml.MappingNode:
			n.Children = parseMappingNode(valNode, path)
			n.IsMapItem = true
		case yaml.SequenceNode:
			n.Children = parseSequenceNode(valNode, path)
		}

		nodes = append(nodes, n)
	}
	return nodes
}

func parseSequenceNode(node *yaml.Node, parentPath []string) []*model.Node {
	if node.Kind != yaml.SequenceNode {
		return nil
	}

	var nodes []*model.Node
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.MappingNode:
			children := parseMappingNode(item, parentPath)
			nodes = append(nodes, children...)
		case yaml.ScalarNode:
			n := &model.Node{
				Key:      item.Value,
				Value:    item.Value,
				IsScalar: true,
				Path:     parentPath,
				KeyRange: lsp.Range{
					Start: lsp.Position{Line: item.Line - 1, Character: item.Column - 1},
					End:   lsp.Position{Line: item.Line - 1, Character: item.Column - 1 + len(item.Value)},
				},
				Range: lsp.Range{
					Start: lsp.Position{Line: item.Line - 1, Character: item.Column - 1},
					End:   lsp.Position{Line: item.Line - 1, Character: item.Column - 1 + len(item.Value)},
				},
			}
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func nodeRange(keyNode, valNode *yaml.Node) lsp.Range {
	startLine := keyNode.Line - 1
	startChar := keyNode.Column - 1

	endLine := valNode.Line - 1
	endChar := valNode.Column - 1

	if valNode.Kind == yaml.ScalarNode {
		endChar += len(valNode.Value)
	}

	// For multi-line values, try to capture the full range.
	if valNode.Kind == yaml.MappingNode || valNode.Kind == yaml.SequenceNode {
		if len(valNode.Content) > 0 {
			last := lastNode(valNode)
			endLine = last.Line - 1
			endChar = last.Column - 1 + len(last.Value)
		}
	}

	return lsp.Range{
		Start: lsp.Position{Line: startLine, Character: startChar},
		End:   lsp.Position{Line: endLine, Character: endChar},
	}
}

func lastNode(node *yaml.Node) *yaml.Node {
	if len(node.Content) == 0 {
		return node
	}
	return lastNode(node.Content[len(node.Content)-1])
}

func extractTemplateRefs(valNode *yaml.Node) []*model.TemplateRef {
	matches := templateRefRe.FindAllStringSubmatchIndex(valNode.Value, -1)
	if len(matches) == 0 {
		return nil
	}

	line := valNode.Line - 1
	colBase := valNode.Column - 1

	// yaml.Node.Column points at the quote character for quoted scalars,
	// but Value contains the unquoted content. Adjust for the opening quote.
	if valNode.Style == yaml.DoubleQuotedStyle || valNode.Style == yaml.SingleQuotedStyle {
		colBase++
	}

	var refs []*model.TemplateRef
	for _, m := range matches {
		// m[2], m[3] are the submatch (the variable name after the dot).
		name := valNode.Value[m[2]:m[3]]
		refs = append(refs, &model.TemplateRef{
			Name: name,
			Range: lsp.Range{
				Start: lsp.Position{Line: line, Character: colBase + m[2]},
				End:   lsp.Position{Line: line, Character: colBase + m[3]},
			},
		})
	}
	return refs
}
