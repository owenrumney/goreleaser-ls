package model

import "github.com/owenrumney/go-lsp/lsp"

// Config is the top-level AST for a parsed .goreleaser.yml file.
type Config struct {
	URI   lsp.DocumentURI
	Nodes []*Node
}

// Node represents a YAML key-value pair with positional information.
type Node struct {
	Key          string
	Value        string
	KeyRange     lsp.Range
	ValueRange   lsp.Range
	Range        lsp.Range
	Path         []string
	Children     []*Node
	TemplateRefs []*TemplateRef
	IsScalar     bool
	IsMapItem    bool
}

// TemplateRef is a Go template reference found in a string value.
type TemplateRef struct {
	Name  string
	Range lsp.Range
}

// FindNodeAtPosition returns the deepest node containing the given position.
func (c *Config) FindNodeAtPosition(pos lsp.Position) *Node {
	if c == nil {
		return nil
	}
	return findNodeAt(c.Nodes, pos)
}

func findNodeAt(nodes []*Node, pos lsp.Position) *Node {
	for _, n := range nodes {
		if !inRange(pos, n.Range) {
			continue
		}
		if child := findNodeAt(n.Children, pos); child != nil {
			return child
		}
		return n
	}
	return nil
}

// FindNodeByPath returns the node at the given YAML key path.
func (c *Config) FindNodeByPath(path ...string) *Node {
	if c == nil || len(path) == 0 {
		return nil
	}
	nodes := c.Nodes
	var matched *Node
	for _, key := range path {
		matched = nil
		for _, n := range nodes {
			if n.Key == key {
				matched = n
				nodes = n.Children
				break
			}
		}
		if matched == nil {
			return nil
		}
	}
	return matched
}

// AllNodes returns a flat list of all nodes in the config.
func (c *Config) AllNodes() []*Node {
	if c == nil {
		return nil
	}
	var result []*Node
	collectNodes(c.Nodes, &result)
	return result
}

func collectNodes(nodes []*Node, result *[]*Node) {
	for _, n := range nodes {
		*result = append(*result, n)
		collectNodes(n.Children, result)
	}
}

func inRange(pos lsp.Position, r lsp.Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}
