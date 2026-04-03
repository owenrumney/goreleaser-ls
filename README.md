# GoReleaser Language Server

A language server for `.goreleaser.yml` files, providing editor support via the Language Server Protocol.

![demo](.github/images/demo.gif)

## Features

- **Completion** — context-aware suggestions for keys, values, and templates
- **Hover** — inline documentation for configuration properties
- **Diagnostics** — real-time validation of your GoReleaser config
- **Go to Definition** — jump to anchors, aliases, and referenced targets
- **Document Symbols** — outline view of your configuration structure
- **Code Actions** — quick fixes for common issues

## Installation

### VS Code Marketplace

Search for **GoReleaser Language Server** in the Extensions panel, or install from the command line:

```sh
code --install-extension owenrumney.goreleaser-ls
```

### Manual

Download a `.vsix` from the [releases page](https://github.com/owenrumney/goreleaser-ls/releases) and install it:

```sh
code --install-extension goreleaser-ls-*.vsix
```

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `goreleaser-ls.serverPath` | `""` | Path to the `goreleaser-ls` binary. Leave empty to use the bundled binary. |

## License

MIT
