# d2-lsp

`d2-lsp` is a Go language server for [D2](https://d2lang.com/) intended for editors such as Helix.

Current support:

- LSP 3.17 JSON-RPC framing
- explicit UTF-16 position encoding
- `initialize`, `shutdown`, and `exit`
- `rootUri`, LSP 3.17 `workspaceFolders`, and dynamic workspace folder changes
- incremental text document synchronization for `.d2` buffers, with diagnostics refreshed on save
- publish syntax and semantic diagnostics from upstream D2 parser/compiler, including imported workspace files
- workspace diagnostic scan after initialization
- watched file changes refresh workspace diagnostics
- completion from upstream D2 language tooling with enriched detail and documentation
- document formatting from the upstream D2 formatter
- code action for applying D2 formatting as a source edit
- document symbols from the D2 AST
- workspace symbols from workspace `.d2` files and open buffers
- folding ranges for nested D2 maps
- references via upstream D2 reference resolution, including imported workspace files
- go-to-definition via upstream D2 reference resolution, including imported workspace files
- document highlights via upstream D2 reference resolution
- enriched hover for D2 keywords, style keys, and selected values
- inlay hints for resolved D2 import paths
- semantic tokens from the D2 AST
- rename via upstream D2 reference resolution
- selection ranges from the D2 AST
- document links for D2 `link` and `icon` URL values
- document colors and color presentations for D2 style color values

## Development

Enter the Nix development shell:

```sh
nix develop
```

Run tests:

```sh
go test ./...
```

Run the server:

```sh
go run ./cmd/d2-lsp
```

Build with Nix:

```sh
nix build
```

Run with Nix:

```sh
nix run
```
