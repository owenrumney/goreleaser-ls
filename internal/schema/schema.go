package schema

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

// Lookup returns the field definition for the given YAML key path.
func Lookup(path ...string) *Field {
	fields := TopLevel
	for i, key := range path {
		for _, f := range fields {
			if f.Key == key {
				if i == len(path)-1 {
					return f
				}
				fields = f.Children
				break
			}
		}
	}
	return nil
}

// ChildKeys returns the valid child keys for the given path.
func ChildKeys(path ...string) []*Field {
	if len(path) == 0 {
		return TopLevel
	}
	f := Lookup(path...)
	if f != nil {
		return f.Children
	}
	return nil
}

// TopLevel defines all goreleaser v2 top-level keys.
var TopLevel = []*Field{
	{Key: "version", Doc: "Schema version (must be `2`).", Type: TypeInt, Required: true},
	{Key: "project_name", Doc: "Project name used in templates and defaults.", Type: TypeString},
	{Key: "dist", Doc: "Output directory for artifacts. Default: `./dist`.", Type: TypeString},
	{Key: "env", Doc: "Global environment variables available to all steps.", Type: TypeList},
	{Key: "env_files", Doc: "Files containing environment variables (e.g. token files).", Type: TypeObject, Children: envFilesFields},
	{Key: "before", Doc: "Pre-release hooks that run before the pipeline.", Type: TypeObject, Children: beforeFields},
	{Key: "git", Doc: "Git configuration for tag and commit detection.", Type: TypeObject, Children: gitFields},
	{Key: "gomod", Doc: "Go module proxy settings.", Type: TypeObject, Children: gomodFields},
	{Key: "builds", Doc: "Build configurations for Go binaries.", Type: TypeList, Children: buildFields},
	{Key: "universal_binaries", Doc: "macOS universal binary creation (lipo).", Type: TypeList, Children: universalBinaryFields},
	{Key: "upx", Doc: "UPX binary compression settings.", Type: TypeList, Children: upxFields},
	{Key: "archives", Doc: "Archive packaging (tar.gz, zip, etc.).", Type: TypeList, Children: archiveFields},
	{Key: "nfpms", Doc: "Linux package generation (deb, rpm, apk, archlinux).", Type: TypeList, Children: nfpmFields},
	{Key: "snapcrafts", Doc: "Snap package configuration.", Type: TypeList},
	{Key: "flatpak", Doc: "Flatpak package configuration.", Type: TypeList},
	{Key: "checksum", Doc: "Checksum file generation.", Type: TypeObject, Children: checksumFields},
	{Key: "signs", Doc: "Artifact signing configuration (GPG, cosign, etc.).", Type: TypeList, Children: signFields},
	{Key: "binary_signs", Doc: "Binary-specific signing configuration.", Type: TypeList},
	{Key: "docker_signs", Doc: "Docker image signing configuration.", Type: TypeList},
	{Key: "dockers_v2", Doc: "Docker image build configuration.", Type: TypeList, Children: dockerFields},
	{Key: "docker_manifests", Doc: "Docker manifest list configuration.", Type: TypeList, Deprecated: "Use dockers_v2 with manifests.", Replacement: "dockers_v2"},
	{Key: "dockers", Doc: "Docker image builds (legacy).", Type: TypeList, Deprecated: "Use dockers_v2.", Replacement: "dockers_v2"},
	{Key: "source", Doc: "Source archive configuration.", Type: TypeObject, Children: sourceFields},
	{Key: "snapshot", Doc: "Snapshot version template for non-tagged builds.", Type: TypeObject, Children: snapshotFields},
	{Key: "release", Doc: "GitHub/GitLab/Gitea release configuration.", Type: TypeObject, Children: releaseFields},
	{Key: "changelog", Doc: "Changelog generation settings.", Type: TypeObject, Children: changelogFields},
	{Key: "milestones", Doc: "Milestone management after release.", Type: TypeList},
	{Key: "sboms", Doc: "Software Bill of Materials generation.", Type: TypeList, Children: sbomFields},
	{Key: "notarize", Doc: "macOS notarization configuration.", Type: TypeList},
	{Key: "announce", Doc: "Announcement integrations (Slack, Discord, etc.).", Type: TypeObject, Children: announceFields},
	{Key: "brews", Doc: "Homebrew formula configuration.", Type: TypeList, Children: brewFields, Deprecated: "Use homebrew_casks.", Replacement: "homebrew_casks"},
	{Key: "homebrew_casks", Doc: "Homebrew cask definitions.", Type: TypeList, Children: brewFields},
	{Key: "nix", Doc: "Nix package configuration.", Type: TypeList},
	{Key: "winget", Doc: "Windows Package Manager configuration.", Type: TypeList},
	{Key: "aurs", Doc: "Arch User Repository package configuration.", Type: TypeList},
	{Key: "aur_sources", Doc: "AUR source package configuration.", Type: TypeList},
	{Key: "krews", Doc: "Krew (kubectl plugin) configuration.", Type: TypeList},
	{Key: "kos", Doc: "Ko container image configuration.", Type: TypeList},
	{Key: "scoops", Doc: "Scoop (Windows package manager) configuration.", Type: TypeList},
	{Key: "chocolateys", Doc: "Chocolatey package configuration.", Type: TypeList},
	{Key: "artifactories", Doc: "JFrog Artifactory upload configuration.", Type: TypeList},
	{Key: "uploads", Doc: "Generic HTTP upload configuration.", Type: TypeList},
	{Key: "blobs", Doc: "Cloud blob storage uploads (S3, GCS, Azure).", Type: TypeList, Children: blobFields},
	{Key: "publishers", Doc: "Custom publisher configuration.", Type: TypeList},
	{Key: "force_token", Doc: "Force a specific token type.", Type: TypeEnum, EnumValues: []string{"github", "gitlab", "gitea"}},
	{Key: "github_urls", Doc: "Custom GitHub Enterprise API URLs.", Type: TypeObject, Children: urlFields},
	{Key: "gitlab_urls", Doc: "Custom GitLab API URLs.", Type: TypeObject, Children: urlFields},
	{Key: "gitea_urls", Doc: "Custom Gitea API URLs.", Type: TypeObject, Children: urlFields},
	{Key: "report_sizes", Doc: "Report artifact sizes in the log.", Type: TypeBool},
	{Key: "metadata", Doc: "Project metadata.", Type: TypeObject},
	{Key: "pro", Doc: "Enable GoReleaser Pro features.", Type: TypeBool},
}

var envFilesFields = []*Field{
	{Key: "github_token", Doc: "Path to file containing the GitHub token.", Type: TypeString},
	{Key: "gitlab_token", Doc: "Path to file containing the GitLab token.", Type: TypeString},
	{Key: "gitea_token", Doc: "Path to file containing the Gitea token.", Type: TypeString},
}

var beforeFields = []*Field{
	{Key: "hooks", Doc: "Shell commands to run before the pipeline.", Type: TypeList},
}

var gitFields = []*Field{
	{Key: "tag_sort", Doc: "Tag sort order (`-version:refname` for semver).", Type: TypeString},
	{Key: "prerelease_suffix", Doc: "Suffix to identify pre-release tags.", Type: TypeString},
	{Key: "ignore_tags", Doc: "Tags to ignore.", Type: TypeList},
}

var gomodFields = []*Field{
	{Key: "proxy", Doc: "Enable Go module proxy.", Type: TypeBool},
	{Key: "env", Doc: "Environment variables for go mod.", Type: TypeList},
	{Key: "gobinary", Doc: "Path to Go binary.", Type: TypeString},
	{Key: "mod", Doc: "Go module mode.", Type: TypeString},
}

var buildFields = []*Field{
	{Key: "id", Doc: "Unique build identifier, referenced by archives, nfpms, etc.", Type: TypeString},
	{Key: "dir", Doc: "Working directory for the build.", Type: TypeString},
	{Key: "main", Doc: "Path to main package (`.` or `./cmd/myapp`).", Type: TypeString},
	{Key: "binary", Doc: "Output binary name.", Type: TypeString},
	{Key: "goos", Doc: "Target operating systems.", Type: TypeList, Children: goosValues},
	{Key: "goarch", Doc: "Target architectures.", Type: TypeList, Children: goarchValues},
	{Key: "goarm", Doc: "ARM versions for GOARM.", Type: TypeList},
	{Key: "goamd64", Doc: "AMD64 microarchitecture levels (v1-v4).", Type: TypeList},
	{Key: "gomips", Doc: "MIPS float ABI.", Type: TypeList},
	{Key: "ignore", Doc: "GOOS/GOARCH combinations to skip.", Type: TypeList},
	{Key: "targets", Doc: "Explicit build targets (overrides goos/goarch).", Type: TypeList},
	{Key: "env", Doc: "Environment variables for this build.", Type: TypeList},
	{Key: "flags", Doc: "Go build flags.", Type: TypeList},
	{Key: "ldflags", Doc: "Linker flags (e.g. `-s -w -X main.version={{.Version}}`).", Type: TypeList},
	{Key: "tags", Doc: "Build tags.", Type: TypeList},
	{Key: "mod_timestamp", Doc: "Timestamp for reproducible builds.", Type: TypeString},
	{Key: "hooks", Doc: "Pre/post build hooks.", Type: TypeObject, Children: buildHookFields},
	{Key: "no_unique_dist_dir", Doc: "Disable unique dist directories per target.", Type: TypeBool},
	{Key: "tool", Doc: "Go binary to use for building.", Type: TypeString},
	{Key: "command", Doc: "Build command (default: build).", Type: TypeString},
	{Key: "gobinary", Doc: "Path to Go binary.", Type: TypeString, Deprecated: "Use `tool`.", Replacement: "tool"},
	{Key: "buildmode", Doc: "Go build mode (e.g. `c-shared`, `pie`).", Type: TypeString},
	{Key: "overrides", Doc: "Per-target build overrides.", Type: TypeList},
}

var buildHookFields = []*Field{
	{Key: "pre", Doc: "Commands to run before building.", Type: TypeList},
	{Key: "post", Doc: "Commands to run after building.", Type: TypeList},
}

// Not real child fields — used to signal that completion should offer these values.
var goosValues = []*Field{
	{Key: "linux", Doc: "Linux."}, {Key: "darwin", Doc: "macOS."},
	{Key: "windows", Doc: "Windows."}, {Key: "freebsd", Doc: "FreeBSD."},
	{Key: "openbsd", Doc: "OpenBSD."}, {Key: "netbsd", Doc: "NetBSD."},
	{Key: "dragonfly", Doc: "DragonFly BSD."}, {Key: "android", Doc: "Android."},
	{Key: "ios", Doc: "iOS."}, {Key: "js", Doc: "JavaScript/Wasm."},
	{Key: "wasip1", Doc: "WASI Preview 1."},
}

var goarchValues = []*Field{
	{Key: "amd64", Doc: "64-bit x86."}, {Key: "arm64", Doc: "64-bit ARM."},
	{Key: "386", Doc: "32-bit x86."}, {Key: "arm", Doc: "32-bit ARM."},
	{Key: "mips", Doc: "MIPS."}, {Key: "mips64", Doc: "MIPS 64-bit."},
	{Key: "mipsle", Doc: "MIPS little-endian."}, {Key: "mips64le", Doc: "MIPS 64-bit little-endian."},
	{Key: "ppc64", Doc: "PowerPC 64-bit."}, {Key: "ppc64le", Doc: "PowerPC 64-bit little-endian."},
	{Key: "riscv64", Doc: "RISC-V 64-bit."}, {Key: "s390x", Doc: "IBM Z."},
	{Key: "loong64", Doc: "LoongArch 64-bit."}, {Key: "wasm", Doc: "WebAssembly."},
}

var universalBinaryFields = []*Field{
	{Key: "id", Doc: "Unique identifier for this universal binary.", Type: TypeString},
	{Key: "ids", Doc: "Build IDs to combine into a universal binary.", Type: TypeList},
	{Key: "name_template", Doc: "Output name template.", Type: TypeString},
	{Key: "replace", Doc: "Replace original binaries with the universal binary.", Type: TypeBool},
	{Key: "hooks", Doc: "Pre/post hooks.", Type: TypeObject},
}

var upxFields = []*Field{
	{Key: "ids", Doc: "Build IDs to compress.", Type: TypeList},
	{Key: "enabled", Doc: "Enable UPX compression.", Type: TypeBool},
	{Key: "goos", Doc: "Target OS filter.", Type: TypeList},
	{Key: "goarch", Doc: "Target arch filter.", Type: TypeList},
	{Key: "goarm", Doc: "ARM version filter.", Type: TypeList},
	{Key: "goamd64", Doc: "AMD64 version filter.", Type: TypeList},
	{Key: "compress", Doc: "Compression level (1-9, best, brute).", Type: TypeString},
	{Key: "lzma", Doc: "Use LZMA compression.", Type: TypeBool},
}

var archiveFields = []*Field{
	{Key: "id", Doc: "Unique archive identifier.", Type: TypeString},
	{Key: "ids", Doc: "Build IDs to include in this archive.", Type: TypeList},
	{Key: "builds", Doc: "Build IDs to include.", Type: TypeList, Deprecated: "Use `ids`.", Replacement: "ids"},
	{Key: "name_template", Doc: "Archive filename template.", Type: TypeString},
	{Key: "formats", Doc: "Archive formats (tar.gz, zip, binary, etc.).", Type: TypeList},
	{Key: "format", Doc: "Archive format.", Type: TypeString, Deprecated: "Use `formats` (list).", Replacement: "formats"},
	{Key: "format_overrides", Doc: "Per-OS format overrides.", Type: TypeList},
	{Key: "wrap_in_directory", Doc: "Wrap files in a directory inside the archive.", Type: TypeBool},
	{Key: "files", Doc: "Extra files to include in the archive.", Type: TypeList},
	{Key: "rlcp", Doc: "Deprecated.", Type: TypeBool},
	{Key: "strip_binary_directory", Doc: "Remove the binary directory prefix.", Type: TypeBool},
}

var nfpmFields = []*Field{
	{Key: "id", Doc: "Unique nfpm identifier.", Type: TypeString},
	{Key: "ids", Doc: "Build IDs to package.", Type: TypeList},
	{Key: "builds", Doc: "Build IDs to package.", Type: TypeList, Deprecated: "Use `ids`.", Replacement: "ids"},
	{Key: "package_name", Doc: "Package name.", Type: TypeString},
	{Key: "vendor", Doc: "Package vendor.", Type: TypeString},
	{Key: "homepage", Doc: "Project homepage URL.", Type: TypeString},
	{Key: "maintainer", Doc: "Package maintainer.", Type: TypeString},
	{Key: "description", Doc: "Package description.", Type: TypeString},
	{Key: "license", Doc: "Package license (SPDX identifier).", Type: TypeString},
	{Key: "formats", Doc: "Package formats to generate (deb, rpm, apk, archlinux).", Type: TypeList},
	{Key: "bindir", Doc: "Binary installation directory.", Type: TypeString},
	{Key: "section", Doc: "Package section (deb).", Type: TypeString},
	{Key: "priority", Doc: "Package priority (deb).", Type: TypeString},
	{Key: "dependencies", Doc: "Package dependencies.", Type: TypeList},
	{Key: "recommends", Doc: "Recommended packages.", Type: TypeList},
	{Key: "suggests", Doc: "Suggested packages.", Type: TypeList},
	{Key: "conflicts", Doc: "Conflicting packages.", Type: TypeList},
	{Key: "replaces", Doc: "Packages this replaces.", Type: TypeList},
	{Key: "contents", Doc: "Files and directories to include.", Type: TypeList},
	{Key: "scripts", Doc: "Installation scripts.", Type: TypeObject},
	{Key: "overrides", Doc: "Format-specific overrides.", Type: TypeObject},
}

var checksumFields = []*Field{
	{Key: "name_template", Doc: "Checksum filename template.", Type: TypeString},
	{Key: "algorithm", Doc: "Hash algorithm (sha256, sha512, md5, etc.).", Type: TypeEnum, EnumValues: []string{"sha256", "sha512", "sha384", "sha1", "md5", "crc32"}},
	{Key: "disable", Doc: "Disable checksum generation.", Type: TypeBool},
	{Key: "ids", Doc: "Artifact IDs to checksum.", Type: TypeList},
}

var signFields = []*Field{
	{Key: "id", Doc: "Unique sign configuration identifier.", Type: TypeString},
	{Key: "cmd", Doc: "Signing command (e.g. gpg, cosign).", Type: TypeString},
	{Key: "args", Doc: "Arguments for the signing command.", Type: TypeList},
	{Key: "artifacts", Doc: "Which artifacts to sign.", Type: TypeEnum, EnumValues: []string{"all", "checksum", "source", "archive", "binary", "sbom", "package"}},
	{Key: "ids", Doc: "Artifact IDs to sign.", Type: TypeList},
	{Key: "signature", Doc: "Signature filename template.", Type: TypeString},
	{Key: "stdin", Doc: "Standard input for the signing command.", Type: TypeString},
	{Key: "stdin_file", Doc: "File to pipe to stdin.", Type: TypeString},
}

var dockerFields = []*Field{
	{Key: "id", Doc: "Unique docker configuration identifier.", Type: TypeString},
	{Key: "ids", Doc: "Build IDs to include.", Type: TypeList},
	{Key: "goos", Doc: "Target OS.", Type: TypeString},
	{Key: "goarch", Doc: "Target architecture.", Type: TypeString},
	{Key: "goarm", Doc: "ARM version.", Type: TypeString},
	{Key: "image_templates", Doc: "Docker image name templates.", Type: TypeList},
	{Key: "dockerfile", Doc: "Path to Dockerfile.", Type: TypeString},
	{Key: "use", Doc: "Docker builder to use (docker, buildx).", Type: TypeEnum, EnumValues: []string{"docker", "buildx"}},
	{Key: "build_flag_templates", Doc: "Extra docker build flags.", Type: TypeList},
	{Key: "push_flags", Doc: "Extra docker push flags.", Type: TypeList},
	{Key: "extra_files", Doc: "Extra files to copy into the Docker build context.", Type: TypeList},
}

var sourceFields = []*Field{
	{Key: "enabled", Doc: "Enable source archive.", Type: TypeBool},
	{Key: "name_template", Doc: "Source archive filename template.", Type: TypeString},
	{Key: "format", Doc: "Source archive format (tar.gz, zip).", Type: TypeString},
	{Key: "prefix_template", Doc: "Prefix template for files in the archive.", Type: TypeString},
}

var snapshotFields = []*Field{
	{Key: "version_template", Doc: "Version template for snapshot builds.", Type: TypeString},
	{Key: "name_template", Doc: "Name template for snapshots.", Type: TypeString, Deprecated: "Use `version_template`.", Replacement: "version_template"},
}

var releaseFields = []*Field{
	{Key: "github", Doc: "GitHub release target.", Type: TypeObject, Children: repoFields},
	{Key: "gitlab", Doc: "GitLab release target.", Type: TypeObject, Children: repoFields},
	{Key: "gitea", Doc: "Gitea release target.", Type: TypeObject, Children: repoFields},
	{Key: "name_template", Doc: "Release name template.", Type: TypeString},
	{Key: "disable", Doc: "Disable release creation.", Type: TypeBool},
	{Key: "draft", Doc: "Create release as draft.", Type: TypeBool},
	{Key: "prerelease", Doc: "Mark as prerelease (`auto` detects from tag).", Type: TypeString},
	{Key: "make_latest", Doc: "Mark as latest release.", Type: TypeBool},
	{Key: "mode", Doc: "Release mode (`append`, `replace`, `keep-existing`).", Type: TypeEnum, EnumValues: []string{"append", "replace", "keep-existing"}},
	{Key: "header", Doc: "Header template for release notes.", Type: TypeString},
	{Key: "footer", Doc: "Footer template for release notes.", Type: TypeString},
	{Key: "extra_files", Doc: "Additional files to upload to the release.", Type: TypeList},
	{Key: "ids", Doc: "Artifact IDs to include in the release.", Type: TypeList},
}

var repoFields = []*Field{
	{Key: "owner", Doc: "Repository owner (user or org).", Type: TypeString},
	{Key: "name", Doc: "Repository name.", Type: TypeString},
}

var changelogFields = []*Field{
	{Key: "use", Doc: "Changelog source (`git`, `github`, `github-native`, `gitlab`).", Type: TypeEnum, EnumValues: []string{"git", "github", "github-native", "gitlab"}},
	{Key: "sort", Doc: "Sort order for commits.", Type: TypeEnum, EnumValues: []string{"asc", "desc", ""}},
	{Key: "disable", Doc: "Disable changelog generation.", Type: TypeBool},
	{Key: "filters", Doc: "Commit message filters.", Type: TypeObject, Children: changelogFilterFields},
	{Key: "groups", Doc: "Group commits by pattern.", Type: TypeList},
	{Key: "abbrev", Doc: "Abbreviate commit hashes to this length.", Type: TypeInt},
}

var changelogFilterFields = []*Field{
	{Key: "exclude", Doc: "Patterns to exclude from changelog.", Type: TypeList},
	{Key: "include", Doc: "Patterns to include in changelog.", Type: TypeList},
}

var sbomFields = []*Field{
	{Key: "id", Doc: "Unique SBOM configuration identifier.", Type: TypeString},
	{Key: "cmd", Doc: "SBOM generation command (e.g. syft).", Type: TypeString},
	{Key: "args", Doc: "Arguments for the SBOM command.", Type: TypeList},
	{Key: "artifacts", Doc: "Which artifacts to generate SBOMs for.", Type: TypeString},
	{Key: "ids", Doc: "Artifact IDs.", Type: TypeList},
	{Key: "documents", Doc: "SBOM output filename templates.", Type: TypeList},
}

var announceFields = []*Field{
	{Key: "slack", Doc: "Slack announcement.", Type: TypeObject},
	{Key: "discord", Doc: "Discord announcement.", Type: TypeObject},
	{Key: "reddit", Doc: "Reddit announcement.", Type: TypeObject},
	{Key: "teams", Doc: "Microsoft Teams announcement.", Type: TypeObject},
	{Key: "telegram", Doc: "Telegram announcement.", Type: TypeObject},
	{Key: "twitter", Doc: "Twitter announcement.", Type: TypeObject},
	{Key: "mastodon", Doc: "Mastodon announcement.", Type: TypeObject},
	{Key: "mattermost", Doc: "Mattermost announcement.", Type: TypeObject},
	{Key: "smtp", Doc: "Email announcement.", Type: TypeObject},
	{Key: "webhook", Doc: "Webhook announcement.", Type: TypeObject},
}

var brewFields = []*Field{
	{Key: "name", Doc: "Formula/cask name.", Type: TypeString},
	{Key: "ids", Doc: "Build IDs to include.", Type: TypeList},
	{Key: "repository", Doc: "Target Homebrew tap repository.", Type: TypeObject, Children: repoFields},
	{Key: "tap", Doc: "Target Homebrew tap.", Type: TypeObject, Deprecated: "Use `repository`.", Replacement: "repository"},
	{Key: "commit_author", Doc: "Git commit author for tap updates.", Type: TypeObject},
	{Key: "commit_msg_template", Doc: "Commit message template.", Type: TypeString},
	{Key: "directory", Doc: "Directory within the tap repo.", Type: TypeString},
	{Key: "homepage", Doc: "Formula homepage.", Type: TypeString},
	{Key: "description", Doc: "Formula description.", Type: TypeString},
	{Key: "license", Doc: "Formula license.", Type: TypeString},
	{Key: "url_template", Doc: "Download URL template.", Type: TypeString},
	{Key: "download_strategy", Doc: "Homebrew download strategy.", Type: TypeString},
	{Key: "dependencies", Doc: "Formula dependencies.", Type: TypeList},
	{Key: "conflicts", Doc: "Conflicting formulae.", Type: TypeList},
	{Key: "test", Doc: "Test block content.", Type: TypeString},
	{Key: "install", Doc: "Install block content.", Type: TypeString},
	{Key: "extra_install", Doc: "Extra install commands.", Type: TypeString},
	{Key: "skip_upload", Doc: "Skip uploading the formula.", Type: TypeBool},
}

var blobFields = []*Field{
	{Key: "provider", Doc: "Cloud provider (s3, gcs, azblob).", Type: TypeEnum, EnumValues: []string{"s3", "gcs", "azblob"}},
	{Key: "bucket", Doc: "Bucket name.", Type: TypeString},
	{Key: "region", Doc: "Bucket region.", Type: TypeString},
	{Key: "directory", Doc: "Directory within the bucket.", Type: TypeString},
	{Key: "ids", Doc: "Artifact IDs to upload.", Type: TypeList},
	{Key: "extra_files", Doc: "Additional files to upload.", Type: TypeList},
	{Key: "disable", Doc: "Disable this upload.", Type: TypeBool},
}

var urlFields = []*Field{
	{Key: "api", Doc: "API base URL.", Type: TypeString},
	{Key: "upload", Doc: "Upload base URL.", Type: TypeString},
	{Key: "download", Doc: "Download base URL.", Type: TypeString},
	{Key: "skip_tls_verify", Doc: "Skip TLS verification.", Type: TypeBool},
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

// IsValidTopLevelKey returns true if the key is a known top-level goreleaser key.
func IsValidTopLevelKey(key string) bool {
	for _, f := range TopLevel {
		if f.Key == key {
			return true
		}
	}
	return false
}
