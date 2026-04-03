# GoReleaser Language Server

A language server for `.goreleaser.yml` — provides real IDE features for GoReleaser configuration.

![Demo](https://raw.githubusercontent.com/owenrumney/goreleaser-ls/main/.github/images/demo.gif)

## Features

| Feature | What it does |
|---|---|
| **Hover** | Inline documentation for configuration properties, enum values, deprecation notices |
| **Completion** | Context-aware suggestions for keys, values, and template variables |
| **Go to Definition** | Jump from build ID references to their definitions |
| **Document Symbols** | Outline view of your configuration structure |
| **Diagnostics** | Unknown keys, deprecated fields, unknown template variables, invalid build ID references |
| **Code Actions** | Quick-fix to replace deprecated field names |

## Install

Install from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=owenrumney.goreleaser-ls), or from a `.vsix` file (see [Releases](https://github.com/owenrumney/goreleaser-ls/releases)):

```sh
code --install-extension goreleaser-ls-darwin-arm64-0.1.0.vsix
```

## Configuration

| Setting | Default | Description |
|---|---|---|
| `goreleaser-ls.serverPath` | `""` | Path to the `goreleaser-ls` binary. Leave empty to use the bundled binary. |

## License

MIT
