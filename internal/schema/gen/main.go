// Command gen downloads the official GoReleaser JSON schemas and generates
// Go source files containing the field definitions used by the language server.
//
//go:generate go run . -out ..
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

const (
	ossURL = "https://raw.githubusercontent.com/goreleaser/goreleaser/refs/heads/main/www/static/schema.json"
	proURL = "https://raw.githubusercontent.com/goreleaser/goreleaser/refs/heads/main/www/static/schema-pro.json"
)

// jsonSchema represents a JSON Schema object. Some fields use json.RawMessage
// because they can be either a boolean or a schema object in the wild.
type jsonSchema struct {
	Ref                  string                 `json:"$ref"`
	Defs                 map[string]*jsonSchema `json:"$defs"`
	Type                 jsonType               `json:"type"`
	Properties           schemaMap              `json:"properties"`
	Items                *jsonSchema            `json:"items"`
	AdditionalProperties json.RawMessage        `json:"additionalProperties"`
	Enum                 []any                  `json:"enum"`
	OneOf                []*jsonSchema          `json:"oneOf"`
	Default              any                    `json:"default"`
}

func (s *jsonSchema) hasAdditionalPropertiesSchema() bool {
	if len(s.AdditionalProperties) == 0 {
		return false
	}
	var b bool
	return json.Unmarshal(s.AdditionalProperties, &b) != nil
}

// schemaMap is a map[string]*jsonSchema that tolerates non-object values
// (e.g. `"Internal": true` in SlackAttachment).
type schemaMap map[string]*jsonSchema

func (m *schemaMap) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	result := make(schemaMap, len(raw))
	for k, v := range raw {
		var s jsonSchema
		if err := json.Unmarshal(v, &s); err != nil {
			// Skip non-schema values (e.g. booleans).
			continue
		}
		result[k] = &s
	}
	*m = result
	return nil
}

// jsonType handles the JSON schema "type" field which can be a string or array.
type jsonType []string

func (t *jsonType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*t = []string{s}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*t = arr
		return nil
	}
	return nil
}

func (t jsonType) First() string {
	if len(t) == 0 {
		return ""
	}
	return t[0]
}

// field is the intermediate representation before code generation.
type field struct {
	Key        string
	Doc        string
	Type       string // TypeString, TypeInt, etc.
	EnumValues []string
	Deprecated string
	Children   []*field
}

// docOverrides provides documentation for fields where it can't be inferred.
var docOverrides = map[string]string{
	"version":      "Schema version (must be `2`).",
	"project_name": "Project name used in templates and defaults.",
	"dist":         "Output directory for artifacts. Default: `./dist`.",
	"pro":          "Enable GoReleaser Pro features.",
	"report_sizes": "Report artifact sizes in the log.",
}

// deprecations maps dotted paths to deprecation info.
var deprecations = map[string]struct{ msg, replacement string }{
	"builds.gobinary":        {"Use `tool`.", "tool"},
	"archives.builds":        {"Use `ids`.", "ids"},
	"archives.format":        {"Use `formats` (list).", "formats"},
	"nfpms.builds":           {"Use `ids`.", "ids"},
	"snapshot.name_template": {"Use `version_template`.", "version_template"},
	"brews":                  {"Use `homebrew_casks`.", "homebrew_casks"},
	"brews.tap":              {"Use `repository`.", "repository"},
	"dockers":                {"Use `dockers_v2`.", "dockers_v2"},
	"docker_manifests":       {"Use `dockers_v2` with manifests.", "dockers_v2"},
	"homebrew_casks.tap":     {"Use `repository`.", "repository"},
}

func main() {
	outDir := "."
	for i, arg := range os.Args[1:] {
		if arg == "-out" && i+1 < len(os.Args)-1 {
			outDir = os.Args[i+2]
		}
	}

	log.SetFlags(0)

	for _, variant := range []struct {
		name string
		url  string
		file string
	}{
		{"oss", ossURL, "schema_oss_gen.go"},
		{"pro", proURL, "schema_pro_gen.go"},
	} {
		log.Printf("fetching %s schema...", variant.name)
		schema, err := fetchSchema(variant.url)
		if err != nil {
			log.Fatalf("fetch %s: %v", variant.name, err)
		}

		fields := convertProject(schema)
		path := filepath.Join(outDir, variant.file)
		if err := writeGoFile(path, variant.name, fields); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		log.Printf("wrote %s (%d top-level fields)", path, len(fields))
	}
}

func fetchSchema(url string) (*jsonSchema, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var s jsonSchema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func convertProject(root *jsonSchema) []*field {
	projectDef, ok := root.Defs["Project"]
	if !ok {
		log.Fatal("Project definition not found in schema")
	}

	fields := make([]*field, 0, len(projectDef.Properties))
	for key, prop := range projectDef.Properties {
		f := convertProperty(root, key, prop, []string{key}, nil)
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })
	return fields
}

func convertProperty(root *jsonSchema, key string, prop *jsonSchema, path []string, visited map[string]bool) *field {
	if visited == nil {
		visited = make(map[string]bool)
	}

	resolved := resolve(root, prop, visited)
	f := &field{Key: key}

	// Collect string enum values first so we can decide whether to treat as enum.
	var enumVals []string
	for _, v := range resolved.Enum {
		if s, ok := v.(string); ok && s != "" {
			enumVals = append(enumVals, s)
		}
	}

	switch {
	case len(enumVals) > 0:
		f.Type = "TypeEnum"
		f.EnumValues = enumVals
	case resolved.Type.First() == "array":
		f.Type = "TypeList"
		if resolved.Items != nil {
			itemResolved := resolve(root, resolved.Items, copyVisited(visited))
			if len(itemResolved.Properties) > 0 {
				f.Children = convertProperties(root, itemResolved.Properties, path, copyVisited(visited))
			}
		}
	case resolved.Type.First() == "object":
		switch {
		case resolved.hasAdditionalPropertiesSchema():
			f.Type = "TypeMap"
		case len(resolved.Properties) > 0:
			f.Type = "TypeObject"
			f.Children = convertProperties(root, resolved.Properties, path, copyVisited(visited))
		default:
			f.Type = "TypeObject"
		}
	case resolved.Type.First() == "integer":
		f.Type = "TypeInt"
	case resolved.Type.First() == "boolean":
		f.Type = "TypeBool"
	default:
		if t := typeFromOneOf(resolved); t != "" {
			f.Type = t
		} else {
			f.Type = "TypeString"
		}
	}

	// Apply doc override or generate doc.
	dottedPath := strings.Join(path, ".")
	if doc, ok := docOverrides[dottedPath]; ok {
		f.Doc = doc
	} else {
		f.Doc = generateDoc(key, f)
	}

	// Apply deprecation.
	if dep, ok := deprecations[dottedPath]; ok {
		f.Deprecated = dep.msg
		f.Doc += "\000REPLACEMENT:" + dep.replacement
	}

	return f
}

func convertProperties(root *jsonSchema, props map[string]*jsonSchema, parentPath []string, visited map[string]bool) []*field {
	fields := make([]*field, 0, len(props))
	for key, prop := range props {
		childPath := append(slices.Clone(parentPath), key)
		f := convertProperty(root, key, prop, childPath, copyVisited(visited))
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })
	return fields
}

func resolve(root *jsonSchema, s *jsonSchema, visited map[string]bool) *jsonSchema {
	if s == nil {
		return &jsonSchema{}
	}

	// Handle oneOf: prefer array or object options over scalars.
	// If all options are scalar, return the original so typeFromOneOf can pick.
	if len(s.OneOf) > 0 {
		for _, opt := range s.OneOf {
			resolved := resolve(root, opt, copyVisited(visited))
			if resolved.Type.First() == "array" || len(resolved.Properties) > 0 {
				return resolved
			}
		}
		return s
	}

	if s.Ref == "" {
		return s
	}

	name := s.Ref[strings.LastIndex(s.Ref, "/")+1:]
	if visited[name] {
		return &jsonSchema{Type: jsonType{"object"}}
	}
	visited[name] = true

	def, ok := root.Defs[name]
	if !ok {
		return s
	}
	return resolve(root, def, visited)
}

func typeFromOneOf(s *jsonSchema) string {
	if len(s.OneOf) == 0 {
		return ""
	}
	types := make(map[string]bool)
	for _, opt := range s.OneOf {
		types[opt.Type.First()] = true
	}
	if types["boolean"] {
		return "TypeBool"
	}
	if types["integer"] {
		return "TypeInt"
	}
	return ""
}

func copyVisited(v map[string]bool) map[string]bool {
	c := make(map[string]bool, len(v))
	maps.Copy(c, v)
	return c
}

func generateDoc(key string, f *field) string {
	words := strings.ReplaceAll(key, "_", " ")
	words = capitalizeFirst(words)

	switch f.Type {
	case "TypeList":
		if len(f.Children) > 0 {
			return words + " configuration."
		}
		return words + "."
	case "TypeObject":
		return words + " configuration."
	case "TypeEnum":
		if len(f.EnumValues) > 0 {
			return words + ": " + strings.Join(f.EnumValues, ", ") + "."
		}
		return words + "."
	default:
		return words + "."
	}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Template for generating Go source files. Uses recursive templates for
// nested fields, with a maximum of 4 levels of indentation.
var goFileTmpl = template.Must(template.New("gofile").Parse(`// Code generated by go generate; DO NOT EDIT.

package schema

var {{ .VarName }} = []*Field{
{{ range .Fields }}{{ template "field" . }}{{ end }}}
{{ define "field" }}	{
		Key: {{ printf "%q" .Key }},
		Doc: {{ printf "%q" .CleanDoc }},
		Type: {{ .Type }},
{{ if .EnumValues }}		EnumValues: []string{ {{ .EnumValuesLit }} },
{{ end }}{{ if .Deprecated }}		Deprecated: {{ printf "%q" .Deprecated }},
{{ end }}{{ if .Replacement }}		Replacement: {{ printf "%q" .Replacement }},
{{ end }}{{ if .Children }}		Children: []*Field{
{{ range .Children }}{{ template "childfield" . }}{{ end }}		},
{{ end }}	},
{{ end }}{{ define "childfield" }}			{
				Key: {{ printf "%q" .Key }},
				Doc: {{ printf "%q" .CleanDoc }},
				Type: {{ .Type }},
{{ if .EnumValues }}				EnumValues: []string{ {{ .EnumValuesLit }} },
{{ end }}{{ if .Deprecated }}				Deprecated: {{ printf "%q" .Deprecated }},
{{ end }}{{ if .Replacement }}				Replacement: {{ printf "%q" .Replacement }},
{{ end }}{{ if .Children }}				Children: []*Field{
{{ range .Children }}{{ template "deepfield" . }}{{ end }}				},
{{ end }}			},
{{ end }}{{ define "deepfield" }}					{
						Key: {{ printf "%q" .Key }},
						Doc: {{ printf "%q" .CleanDoc }},
						Type: {{ .Type }},
{{ if .EnumValues }}						EnumValues: []string{ {{ .EnumValuesLit }} },
{{ end }}{{ if .Deprecated }}						Deprecated: {{ printf "%q" .Deprecated }},
{{ end }}{{ if .Replacement }}						Replacement: {{ printf "%q" .Replacement }},
{{ end }}{{ if .Children }}						Children: []*Field{
{{ range .Children }}							{{ template "leaffield" . }}{{ end }}						},
{{ end }}					},
{{ end }}{{ define "leaffield" }}{Key: {{ printf "%q" .Key }}, Doc: {{ printf "%q" .CleanDoc }}, Type: {{ .Type }}{{ if .EnumValues }}, EnumValues: []string{ {{ .EnumValuesLit }} }{{ end }}},
{{ end }}`))

type tmplField struct {
	Key        string
	Doc        string
	Type       string
	EnumValues []string
	Deprecated string
	Children   []*tmplField
}

func (f *tmplField) CleanDoc() string {
	if idx := strings.Index(f.Doc, "\000REPLACEMENT:"); idx >= 0 {
		return f.Doc[:idx]
	}
	return f.Doc
}

func (f *tmplField) Replacement() string {
	if idx := strings.Index(f.Doc, "\000REPLACEMENT:"); idx >= 0 {
		return f.Doc[idx+len("\000REPLACEMENT:"):]
	}
	return ""
}

func (f *tmplField) EnumValuesLit() string {
	parts := make([]string, 0, len(f.EnumValues))
	for _, v := range f.EnumValues {
		parts = append(parts, fmt.Sprintf("%q", v))
	}
	return strings.Join(parts, ", ")
}

func toTmplField(f *field) *tmplField {
	tf := &tmplField{
		Key:        f.Key,
		Doc:        f.Doc,
		Type:       f.Type,
		EnumValues: f.EnumValues,
		Deprecated: f.Deprecated,
	}
	for _, c := range f.Children {
		tf.Children = append(tf.Children, toTmplField(c))
	}
	return tf
}

func writeGoFile(path, variant string, fields []*field) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	varName := "ossFields"
	if variant == "pro" {
		varName = "proFields"
	}

	var tmplFields []*tmplField
	for _, fld := range fields {
		tmplFields = append(tmplFields, toTmplField(fld))
	}

	return goFileTmpl.Execute(f, struct {
		VarName string
		Fields  []*tmplField
	}{
		VarName: varName,
		Fields:  tmplFields,
	})
}
