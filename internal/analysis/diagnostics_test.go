package analysis

import (
	"testing"

	"github.com/owenrumney/goreleaser-ls/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnose_MissingVersion(t *testing.T) {
	input := `project_name: myapp
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Message, "missing required field: version")
}

func TestDiagnose_WrongVersion(t *testing.T) {
	input := `version: 1
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Message, "version must be 2")
}

func TestDiagnose_ValidVersion(t *testing.T) {
	input := `version: 2
project_name: myapp
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	for _, d := range diags {
		assert.NotContains(t, d.Message, "version")
	}
}

func TestDiagnose_UnknownTopLevelKey(t *testing.T) {
	input := `version: 2
bogus_key: something
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	found := false
	for _, d := range diags {
		if d.Message == "unknown top-level key: bogus_key" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestDiagnose_DeprecatedField(t *testing.T) {
	input := `version: 2
brews:
  - name: myapp
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	found := false
	for _, d := range diags {
		if assert.ObjectsAreEqual(d.Range, cfg.Nodes[1].KeyRange) {
			found = true
			assert.Contains(t, d.Message, "deprecated")
		}
	}
	assert.True(t, found)
}

func TestDiagnose_DeprecatedNestedField(t *testing.T) {
	input := `version: 2
archives:
  - format: tar.gz
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	found := false
	for _, d := range diags {
		if d.Message != "" && len(d.Message) > 0 {
			if contains(d.Message, "format") && contains(d.Message, "deprecated") {
				found = true
			}
		}
	}
	assert.True(t, found, "expected deprecation warning for archives.format")
}

func TestDiagnose_NilConfig(t *testing.T) {
	diags := Diagnose(nil)
	assert.Empty(t, diags)
}

func TestDiagnose_ValidConfig(t *testing.T) {
	input := `version: 2
project_name: myapp
builds:
  - main: .
archives:
  - formats:
      - tar.gz
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	for _, d := range diags {
		assert.NotContains(t, d.Message, "unknown")
		assert.NotContains(t, d.Message, "deprecated")
	}
}

func TestDiagnose_UnknownTemplateVar(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "-X main.version={{ .Version }} -X main.foo={{ .BogusVar }}"
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	found := false
	for _, d := range diags {
		if contains(d.Message, "unknown template variable") && contains(d.Message, "BogusVar") {
			found = true
		}
	}
	assert.True(t, found, "expected warning for unknown template var .BogusVar")
}

func TestDiagnose_ValidTemplateVar(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "-X main.version={{ .Version }}"
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	for _, d := range diags {
		assert.NotContains(t, d.Message, "unknown template variable")
	}
}

func TestDiagnose_TemplateVarEnvAccess(t *testing.T) {
	input := `version: 2
builds:
  - ldflags: "-X main.token={{ .Env.GITHUB_TOKEN }}"
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	for _, d := range diags {
		assert.NotContains(t, d.Message, "unknown template variable")
	}
}

func TestDiagnose_UnknownBuildID(t *testing.T) {
	input := `version: 2
builds:
  - id: myapp
    main: .
archives:
  - ids:
      - myapp
      - nonexistent
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	found := false
	for _, d := range diags {
		if contains(d.Message, "references unknown build ID") && contains(d.Message, "nonexistent") {
			found = true
		}
	}
	assert.True(t, found, "expected warning for unknown build ID 'nonexistent'")
}

func TestDiagnose_ValidBuildID(t *testing.T) {
	input := `version: 2
builds:
  - id: myapp
    main: .
archives:
  - ids:
      - myapp
`
	cfg := parser.Parse("file:///test.yml", input)
	diags := Diagnose(cfg)

	for _, d := range diags {
		assert.NotContains(t, d.Message, "references unknown build ID")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
