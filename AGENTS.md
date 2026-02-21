# AGENTS.md — Cod project context for AI and developers

## What is Cod

**Cod** is a completion daemon for **bash**, **fish**, and **zsh**. It watches for `--help` usage, parses the output, and generates shell completions. Data: `$XDG_DATA_HOME/cod`, config: `$XDG_CONFIG_HOME/cod/config.toml`.

- **Build:** `go build` (Go 1.19+)
- **Run tests:** `make test` (builds binary and runs full suite; see [Tests](#tests) below)

---

## Repository structure

| Path | Purpose |
|------|--------|
| `main.go` | Entry point, kingpin CLI, `CodBinaryPath`, `Version` |
| `application.go` | Application lifecycle |
| `commands.go` | CLI subcommands (learn, init, list, etc.) |
| `daemon.go` | Daemon process |
| `api_attach.go` | Attach to running daemon |
| `ui.go` | User prompts / TUI |
| `example_configuration.go` | Example config generation |
| `datastore/` | SQLite storage, `HelpPage`, `Completion`, `FlagContext` |
| `parse_doc/` | **Help text parsing** (argparse + default) |
| `server/` | Daemon server and client |
| `shells/` | Bash/zsh/fish script generation, quoting, tokenization |
| `util/` | Helpers (e.g. selector, find_path) |
| `test/` | Integration tests (require `COD_TEST_BINARY` for some) |

---

## Help parsing (parse_doc)

- **Entry:** `ParseHelp(args []string, help string) (*datastore.HelpPage, error)` in `parse_help.go`.
- **Parsers:** Tried in order: **argparse** → **default**. First success wins; failed attempts are only logged.
- **Argparse** (`argparse.go`): Used when help looks like Python argparse (e.g. line starts with `usage:`, application name matches `filepath.Base(args[0])`).

### Argparse parser in short

1. **Usage line:** Must start with `usage:` at the beginning of the text. `makeUsageLexer` + `parseArgparseUsage` parse the usage line (app name, subcommands, optional `[...]`, choice `{a,b,c}`).
2. **Application name:** `filepath.Base(usage.applicationName)` must equal `filepath.Base(context.args[0])` (e.g. `./script.py` → `script.py`).
3. **Sections:**
   - First loop: `FindIndentedParagraph("arguments:", start)` — matches both **"positional arguments:"** and **"optional arguments:"** (substring `"arguments:"`). For each paragraph: `tryParseFlagsParagraph`, `tryParseNamedPositionalParagraph`, or `tryParseUnnamedPositionalParagraph`.
   - Second loop: `FindIndentedParagraph("options:", start)` — only **"options:"** (Python 3.10+ style). Same paragraphs are parsed with `tryParseFlagsParagraph`.
4. **Flags:** Regex `flagRe` extracts tokens like `-h`, `--help` from each line; lines are identified by `flagsLineRe`. Only direct children of the section are used (continuation lines with greater indent are nested and not scanned for flags).
5. **Text layout:** `textutil.go` — `preparedText`, `FindIndentedParagraph`, `lineTree` (indent-based). Paragraphs are built by indent; deeper indent = child node.

### Important details

- **"optional arguments:"** is handled only in the first loop (via `"arguments:"`). Do not add a second loop that also searches for `"optional arguments:"` or you will double-process that section and duplicate completions.
- **"options:"** is only handled in the second loop. Old Python uses **"optional arguments:"**, new uses **"options:"** — both are supported as above.
- Two-line option entries (e.g. `--parser-argument PARSER_ARGUMENT` on one line, description on the next with more indent) work: the option line is one child, the description is a nested child; only the option line is scanned for flags.

---

## Tests

- **All tests (recommended):** `make test` — builds the `cod` binary and runs the full suite with `COD_TEST_BINARY` set, so unit and integration tests pass. Use this to verify changes end-to-end.
- **Parse_doc (unit):** `go test ./parse_doc/...` — no env vars needed. Covers argparse, default parser, `ParseHelp`, `FindIndentedParagraph`, etc.
- **Integration:** Tests in `test/` use `SetupWorkbench` and **require** `COD_TEST_BINARY` (path to the built `cod` binary); without it they panic. Running plain `go test ./...` will fail on those. Use `make test` to run them. These tests start the cod daemon (unix socket in `/tmp`); in restricted sandboxes (e.g. Cursor’s default) you may see `bind: operation not permitted` — run `make test` in a normal terminal or with sandbox disabled.

---

## Conventions

- **Go:** Standard library + kingpin.v2, go-toml, go-sqlite3, testify. Module: `github.com/dim-an/cod`.
- **Parsers:** Implement `HelpParser` (`Name() string`, `Parse(parseContext) (*parseResult, error)`). Register in `parsers` in `parse_help.go`.
- **Completions:** Stored as `datastore.Completion` (Flag + Context). Context includes `Framework` (e.g. `"argparse"`) and `SubCommand` slice for subcommand-aware completions.

---

## Quick reference for common changes

- **Change how argparse detects options:** `parse_doc/argparse.go` — `tryParseFlagsParagraph`, regexes `flagsLineRe` / `flagRe`, and the two loops (arguments / options).
- **Add a new help parser:** Implement `HelpParser` in `parse_doc/`, append to `parsers` in `parse_help.go`.
- **Change CLI:** `main.go` (kingpin), `commands.go` (subcommand handlers).
- **Change completion storage or schema:** `datastore/` (e.g. `data.go`, `sqlitedb.go`).
