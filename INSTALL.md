# Installation

`read_lsp` is a standalone Go binary that communicates over stdin/stdout. It
does not need a Node.js runtime, a database, or a network service.

## Requirements

- Go 1.21 or newer to build from source.
- Neovim 0.8 or newer for the included Neovim integration (optional).
- Any editor or tool that can launch a stdio LSP server.

## Build From Source

```sh
git clone https://github.com/atharva-0123-cs/read_lsp.git
cd read_lsp
make build
```

The binary is written to `./read_lsp`.

Verify the build:

```sh
./read_lsp -version
```

Expected output:

```text
read_lsp 0.4.0
```

## Install on Your PATH

Using Go:

```sh
go install .
```

Or install the built binary manually:

```sh
sudo install -m 0755 read_lsp /usr/local/bin/read_lsp
```

Confirm it is discoverable:

```sh
which read_lsp
```

## Neovim Setup

### Neovim 0.11+ (recommended)

Create `~/.config/nvim/lua/custom/read_lsp.lua`:

```lua
vim.lsp.config("read_lsp", {
  cmd = { "read_lsp", "-log", "/tmp/read_lsp.log" },
  filetypes = { "markdown" },
  root_markers = { ".git", "go.mod" },
})

vim.lsp.enable("read_lsp")
```

Load it from `~/.config/nvim/init.lua`:

```lua
require("custom.read_lsp")
```

### Neovim 0.8–0.10

Use the included [`markdown_lsp.lua`](markdown_lsp.lua) file:

```sh
cp markdown_lsp.lua ~/.config/nvim/lua/custom/read_lsp.lua
```

Then in `~/.config/nvim/init.lua`:

```lua
require("custom.read_lsp")
```

The script searches for `read_lsp` on `PATH`, then `~/read_lsp/read_lsp`. You
can override the binary path with the `READ_LSP_BIN` environment variable:

```sh
export READ_LSP_BIN=/opt/read_lsp/read_lsp
```

Or start the client manually:

```lua
vim.lsp.start({
  name = "read_lsp",
  cmd = { "read_lsp", "-log", "/tmp/read_lsp.log" },
  root_dir = vim.fn.getcwd(),
})
```

## Verify the Integration

1. Open a Markdown file in Neovim.
2. Run `:LspInfo` and confirm `read_lsp` is attached.
3. Add a heading with no space after the marker:

   ```markdown
   #BadHeading
   ```

4. Confirm the diagnostic appears in the editor.
5. Try hovering over a link, finding references, or using completion at the
   start of an empty line.
6. Confirm document symbols and folding ranges appear for headings.
7. Inspect `/tmp/read_lsp.log` if the client does not attach.

## Server Options

```text
read_lsp [flags]

  -log string
        path to the server log file (default "./log.txt")
  -version
        print the server version and exit
```

Example:

```sh
read_lsp -log /tmp/read_lsp.log
```

The server never writes log output to stdout because stdout is reserved for LSP
messages.

## Troubleshooting

| Problem | Resolution |
| --- | --- |
| Client does not attach | Run `:LspInfo`, check that the binary is on `PATH`, and inspect the log file. |
| `permission denied` | Run `chmod +x read_lsp` or reinstall with `install -m 0755`. |
| No diagnostics | Ensure the buffer filetype is `markdown` and that the client supports `textDocument/publishDiagnostics`. |
| Log file not created | Use an absolute `-log` path or run the server from a writable directory. |
| Old binary is used | Run `which -a read_lsp` and remove or replace older copies. |

## Uninstall

```sh
sudo rm /usr/local/bin/read_lsp
rm -rf /path/to/read_lsp
```
