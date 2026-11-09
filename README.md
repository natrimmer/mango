# 🥭 mango

Generate conventional commit messages from your staged changes.

```bash
git add .
mango commit
# feat: add user authentication and password reset
# Commit with this message? [Y/n]  → commits for you
```

## Install

```bash
go install github.com/natrimmer/mango@latest
```

<details>
<summary>Other install options (prebuilt binary, from source)</summary>

**Prebuilt binary** — grab your platform from [Releases](https://github.com/natrimmer/mango/releases/latest):

```bash
# macOS (Apple Silicon); swap for _darwin_amd64 / _linux_amd64 / _linux_arm64 / _windows_amd64.exe
curl -L https://github.com/natrimmer/mango/releases/latest/download/mango_darwin_arm64 -o mango
chmod +x mango && sudo mv mango /usr/local/bin/
```

**From source:**

```bash
git clone https://github.com/natrimmer/mango.git && cd mango && go build
```

</details>

## Use

```bash
mango config set --api-key "sk-ant-..."     # one-time setup
# or, to use your Claude Code login instead of a key:
mango config set --provider claude-code

git add .
mango commit                              # shows the message, then commits on confirm
```

On a terminal, `mango commit` prompts and runs `git commit` for you (with `--count`, pick an option by number). When output is piped or in CI, it instead prints a ready-to-run `git commit` command and commits nothing.

`mango commit` flags:

| Flag | What it does |
|------|--------------|
| `--type`, `-t <t>` | Force a commit type (`feat`, `fix`, `docs`, …) |
| `--context`, `-c "<text>"` | Extra context to guide the message (e.g. `"resolves #123"`) |
| `--count`, `-n <n>` | Offer N options to pick from instead of one |
| `--dry-run` | Show the prompt without generating a message |
| `--verbose`, `-v` | Show the prompt (as `--dry-run` does) plus the raw model response |

Other commands: `mango config show` (show config), `mango config models` (list models), `mango --version`.

## Providers

Set with `mango config set --provider <name>`:

- `api` — **default**, calls the Anthropic API directly with the key from `mango config set --api-key`
- `claude-code` — runs the `claude` CLI on your machine, so it uses whatever Claude Code is already logged in as and mango stores no key

`claude-code` needs `claude` on your `PATH` ([install](https://claude.com/claude-code)). mango runs it with all tools off and outside your repo, so project `CLAUDE.md` files, hooks, and MCP servers don't affect the message.

## Models

Set with `mango config set --model <name>` (both providers accept the same names):

- `claude-opus-5` — most capable, slower, pricier
- `claude-opus-4-8` — previous Opus
- `claude-sonnet-5` — **default**, balanced
- `claude-sonnet-4-6` — previous Sonnet
- `claude-haiku-4-5` — fastest, cheapest

<details>
<summary>Commit message format & conventional types</summary>

Messages follow `<type>: <description>` — lowercase, imperative mood, no trailing period, ≤50 chars.

`feat` new feature · `fix` bug fix · `docs` documentation · `style` formatting · `refactor` · `perf` · `test` · `chore` · `ci` · `build` · `revert`

</details>

<details>
<summary>Configuration & storage</summary>

Config lives at `~/.mango/config.json`. Under the `api` provider the key is stored in **plaintext** — set appropriate file permissions. The key is masked (`sk-a****...`) whenever displayed. The `claude-code` provider stores no key at all.

</details>

<details>
<summary>Development</summary>

Flat `package main` on [Cobra](https://github.com/spf13/cobra), one file per concern:

```
main.go    entrypoint
root.go    root command, version
config.go  config + config/view/models commands
commit.go  prompt building, both generation paths, git helpers, commit command
colors.go  ANSI helpers
```

Tests exercise real behavior (temp `HOME` for config, `httptest` for the API, a stub executable for the `claude` CLI) rather than mocks.

Tasks live in the [justfile](justfile) (run `just` to list them): `build`, `build-release`, `test`, `test-coverage`, `test-race`, `bench`, `fmt`, `vet`, `lint`, `ci`, `clean`. You need Go and, for `lint` and `ci`, [golangci-lint](https://golangci-lint.run).

</details>

<details>
<summary>Releasing</summary>

Versions are git tags ([SemVer](https://semver.org)). `just bump <major|minor|patch>` bumps and pushes a tag after a confirmation prompt; pushing a `vX.Y.Z` tag triggers GitHub Actions to build binaries (Linux/macOS/Windows, amd64 + arm64) and publish a release.

Manual: `git tag v1.2.3 && git push origin v1.2.3`. Rollback: `git tag -d v1.2.3 && git push origin --delete v1.2.3` (delete the GitHub release by hand if it already built).

</details>