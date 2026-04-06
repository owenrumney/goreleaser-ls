package schema

//go:generate go run ./gen -out .

import "sync"

// FieldType describes what kind of value a field expects.
type FieldType int

const (
	TypeString FieldType = iota
	TypeInt
	TypeBool
	TypeList
	TypeMap
	TypeObject
	TypeEnum
)

// Field describes a goreleaser configuration field.
type Field struct {
	Key         string
	Doc         string
	Type        FieldType
	Deprecated  string
	Replacement string
	Required    bool
	Children    []*Field
	EnumValues  []string
}

var (
	mu       sync.RWMutex
	topLevel = ossFields
)

// TopLevel returns the schema fields for the active variant (OSS or Pro).
func TopLevelFields() []*Field {
	mu.RLock()
	defer mu.RUnlock()
	return topLevel
}

// UsePro switches the active schema to the Pro variant.
func UsePro() {
	mu.Lock()
	defer mu.Unlock()
	topLevel = proFields
}

// UseOSS switches the active schema to the OSS variant.
func UseOSS() {
	mu.Lock()
	defer mu.Unlock()
	topLevel = ossFields
}

// Lookup returns the field definition for the given YAML key path.
func Lookup(path ...string) *Field {
	mu.RLock()
	fields := topLevel
	mu.RUnlock()

	var matched *Field
	for i, key := range path {
		matched = nil
		for _, f := range fields {
			if f.Key == key {
				matched = f
				if i < len(path)-1 {
					fields = f.Children
				}
				break
			}
		}
		if matched == nil {
			return nil
		}
	}
	return matched
}

// ChildKeys returns the valid child keys for the given path.
func ChildKeys(path ...string) []*Field {
	if len(path) == 0 {
		return TopLevelFields()
	}
	f := Lookup(path...)
	if f != nil {
		return f.Children
	}
	return nil
}

// TemplateVars lists goreleaser template variables available in string fields.
var TemplateVars = []TemplateVar{
	// Project/Version
	{Name: "ProjectName", Doc: "Project name from config or git remote."},
	{Name: "Version", Doc: "Current version (tag without `v` prefix)."},
	{Name: "RawVersion", Doc: "Raw version string."},
	{Name: "Major", Doc: "Major version number."},
	{Name: "Minor", Doc: "Minor version number."},
	{Name: "Patch", Doc: "Patch version number."},
	{Name: "Prerelease", Doc: "Prerelease suffix."},
	// Git
	{Name: "Tag", Doc: "Current git tag."},
	{Name: "PreviousTag", Doc: "Previous git tag."},
	{Name: "Branch", Doc: "Current git branch."},
	{Name: "ShortCommit", Doc: "Short commit hash."},
	{Name: "FullCommit", Doc: "Full commit hash."},
	{Name: "CommitDate", Doc: "Commit date (RFC 3339)."},
	{Name: "CommitTimestamp", Doc: "Commit Unix timestamp."},
	{Name: "GitURL", Doc: "Git remote URL."},
	{Name: "GitTreeState", Doc: "Git tree state (clean or dirty)."},
	{Name: "IsGitClean", Doc: "True if git tree is clean."},
	{Name: "IsGitDirty", Doc: "True if git tree is dirty."},
	{Name: "Summary", Doc: "Git describe summary."},
	{Name: "TagSubject", Doc: "Annotated tag subject line."},
	{Name: "TagBody", Doc: "Annotated tag body."},
	{Name: "TagContents", Doc: "Full annotated tag message."},
	// Build context
	{Name: "IsSnapshot", Doc: "True if this is a snapshot build."},
	{Name: "IsNightly", Doc: "True if this is a nightly build."},
	{Name: "IsDraft", Doc: "True if release is a draft."},
	{Name: "IsSingleTarget", Doc: "True if building for a single target."},
	{Name: "Date", Doc: "Current date (RFC 3339)."},
	{Name: "Now", Doc: "Current time."},
	{Name: "Timestamp", Doc: "Current Unix timestamp."},
	{Name: "Env", Doc: "Map of environment variables. Access with `{{ .Env.VAR_NAME }}`."},
	{Name: "ReleaseURL", Doc: "URL of the created release."},
	{Name: "ReleaseNotes", Doc: "Generated release notes."},
	{Name: "ModulePath", Doc: "Go module path."},
	// Per-artifact
	{Name: "Os", Doc: "Target operating system."},
	{Name: "Arch", Doc: "Target architecture."},
	{Name: "Arm", Doc: "ARM version."},
	{Name: "Mips", Doc: "MIPS variant."},
	{Name: "Amd64", Doc: "AMD64 microarchitecture level."},
	{Name: "Target", Doc: "Build target string (os_arch)."},
	{Name: "Binary", Doc: "Binary name."},
	{Name: "ArtifactName", Doc: "Artifact filename."},
	{Name: "ArtifactPath", Doc: "Artifact file path."},
	{Name: "ArtifactExt", Doc: "Artifact file extension."},
	// Runtime
	{Name: "Runtime.Goos", Doc: "Runtime OS (the machine running goreleaser)."},
	{Name: "Runtime.Goarch", Doc: "Runtime architecture."},
}

// TemplateVar describes a goreleaser template variable.
type TemplateVar struct {
	Name string
	Doc  string
}
