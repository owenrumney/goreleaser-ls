package parser

import (
	"testing"

	"github.com/owenrumney/goreleaser-ls/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_BasicConfig(t *testing.T) {
	input := `version: 2
project_name: myapp
dist: ./dist
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Nodes, 3)

	assert.Equal(t, "version", cfg.Nodes[0].Key)
	assert.Equal(t, "2", cfg.Nodes[0].Value)
	assert.True(t, cfg.Nodes[0].IsScalar)

	assert.Equal(t, "project_name", cfg.Nodes[1].Key)
	assert.Equal(t, "myapp", cfg.Nodes[1].Value)

	assert.Equal(t, "dist", cfg.Nodes[2].Key)
	assert.Equal(t, "./dist", cfg.Nodes[2].Value)
}

func TestParse_NestedConfig(t *testing.T) {
	input := `version: 2
release:
  github:
    owner: owenrumney
    name: myrepo
  draft: true
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Nodes, 2)

	release := cfg.Nodes[1]
	assert.Equal(t, "release", release.Key)
	require.Len(t, release.Children, 2)

	github := release.Children[0]
	assert.Equal(t, "github", github.Key)
	require.Len(t, github.Children, 2)
	assert.Equal(t, "owner", github.Children[0].Key)
	assert.Equal(t, "owenrumney", github.Children[0].Value)
}

func TestParse_SequenceOfMappings(t *testing.T) {
	input := `version: 2
builds:
  - id: default
    main: ./cmd/app
    binary: app
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Nodes, 2)

	builds := cfg.Nodes[1]
	assert.Equal(t, "builds", builds.Key)
	require.Len(t, builds.Children, 3)
	assert.Equal(t, "id", builds.Children[0].Key)
	assert.Equal(t, "default", builds.Children[0].Value)
}

func TestParse_SequenceOfScalars(t *testing.T) {
	input := `version: 2
env:
  - GO111MODULE=on
  - CGO_ENABLED=0
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	env := cfg.Nodes[1]
	assert.Equal(t, "env", env.Key)
	require.Len(t, env.Children, 2)
	assert.Equal(t, "GO111MODULE=on", env.Children[0].Value)
}

func TestParse_EmptyInput(t *testing.T) {
	cfg := Parse("file:///test.yml", "")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Nodes)
}

func TestParse_InvalidYAML(t *testing.T) {
	cfg := Parse("file:///test.yml", "{{invalid yaml")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Nodes)
}

func TestParse_KeyRanges(t *testing.T) {
	input := `version: 2
project_name: myapp
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	// "version" starts at line 0, col 0.
	versionNode := cfg.Nodes[0]
	assert.Equal(t, 0, versionNode.KeyRange.Start.Line)
	assert.Equal(t, 0, versionNode.KeyRange.Start.Character)
	assert.Equal(t, 7, versionNode.KeyRange.End.Character) // len("version")

	// "project_name" starts at line 1, col 0.
	pnNode := cfg.Nodes[1]
	assert.Equal(t, 1, pnNode.KeyRange.Start.Line)
	assert.Equal(t, 0, pnNode.KeyRange.Start.Character)
}

func TestParse_Paths(t *testing.T) {
	input := `version: 2
release:
  github:
    owner: foo
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	release := cfg.Nodes[1]
	assert.Equal(t, []string{"release"}, release.Path)

	github := release.Children[0]
	assert.Equal(t, []string{"release", "github"}, github.Path)

	owner := github.Children[0]
	assert.Equal(t, []string{"release", "github", "owner"}, owner.Path)
}

func TestFindNodeAtPosition(t *testing.T) {
	input := `version: 2
project_name: myapp
release:
  draft: true
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	t.Run("on version key", func(t *testing.T) {
		node := cfg.FindNodeAtPosition(pos(0, 3))
		require.NotNil(t, node)
		assert.Equal(t, "version", node.Key)
	})

	t.Run("on nested key", func(t *testing.T) {
		node := cfg.FindNodeAtPosition(pos(3, 4))
		require.NotNil(t, node)
		assert.Equal(t, "draft", node.Key)
	})
}

func TestParse_TemplateRefs(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "-X main.version={{ .Version }} -X main.commit={{ .ShortCommit }}"
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	// Find the ldflags node.
	builds := cfg.Nodes[1]
	var ldflags *model.Node
	for _, child := range builds.Children {
		if child.Key == "ldflags" {
			ldflags = child
			break
		}
	}
	require.NotNil(t, ldflags)
	require.Len(t, ldflags.TemplateRefs, 2)
	assert.Equal(t, "Version", ldflags.TemplateRefs[0].Name)
	assert.Equal(t, "ShortCommit", ldflags.TemplateRefs[1].Name)
}

func TestParse_TemplateRefs_DottedPath(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "{{ .Env.GITHUB_TOKEN }}"
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	builds := cfg.Nodes[1]
	var ldflags *model.Node
	for _, child := range builds.Children {
		if child.Key == "ldflags" {
			ldflags = child
			break
		}
	}
	require.NotNil(t, ldflags)
	require.Len(t, ldflags.TemplateRefs, 1)
	assert.Equal(t, "Env.GITHUB_TOKEN", ldflags.TemplateRefs[0].Name)
}

func TestParse_NoTemplateRefs(t *testing.T) {
	input := `version: 2
project_name: myapp
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	for _, n := range cfg.Nodes {
		assert.Empty(t, n.TemplateRefs)
	}
}

func TestFindNodeByPath(t *testing.T) {
	input := `version: 2
release:
  github:
    owner: foo
`
	cfg := Parse("file:///test.yml", input)
	require.NotNil(t, cfg)

	node := cfg.FindNodeByPath("release", "github", "owner")
	require.NotNil(t, node)
	assert.Equal(t, "owner", node.Key)
	assert.Equal(t, "foo", node.Value)

	node = cfg.FindNodeByPath("nonexistent")
	assert.Nil(t, node)
}
