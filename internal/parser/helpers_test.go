package parser

import "github.com/owenrumney/go-lsp/lsp"

func pos(line, char int) lsp.Position {
	return lsp.Position{Line: line, Character: char}
}
