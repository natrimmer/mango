# mango

Generate conventional commit messages from your staged changes, using AI.

```bash
git add .
mango commit
# git commit -m "feat: add user authentication and password reset"
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
mango config --api-key "sk-ant-..."   # one-time setup
git add .
mango commit                          # prints a ready-to-run git commit command
```

`mango commit` flags:

| Flag | What it does |
|------|--------------|
| `--type <t>` | Force a commit type (`feat`, `fix`, `docs`, …) |
| `--context "<text>"` | Extra context to guide the message (e.g. `"resolves #123"`) |
| `--count <n>` | Offer N options to pick from instead of one |
| `--dry-run` | Show the prompt without calling the API |
| `--verbose`, `-v` | Show the full prompt and raw API response |

Other commands: `mango view` (show config), `mango models` (list models), `mango --version`.

## Models

Set with `mango config --model <name>`:

- `claude-opus-4-8` — most capable, slower, pricier
- `claude-sonnet-4-6` — **default**, balanced
- `claude-haiku-4-5` — fastest, cheapest

<details>
<summary>Commit message format & conventional types</summary>

Messages follow `<type>: <description>` — lowercase, imperative mood, no trailing period, ≤50 chars.

`feat` new feature · `fix` bug fix · `docs` documentation · `style` formatting · `refactor` · `perf` · `test` · `chore` · `ci` · `build` · `revert`

</details>

<details>
<summary>Configuration & storage</summary>

Config lives at `~/.mango/config.json`. The API key is stored in **plaintext** — set appropriate file permissions. The key is masked (`sk-a****...`) whenever displayed.

</details>

<details>
<summary>Development</summary>

Flat `package main` on [Cobra](https://github.com/spf13/cobra), one file per concern:

```
main.go    entrypoint
root.go    root command, version
config.go  config + config/view/models commands
commit.go  prompt building, API call, git helpers, commit command
colors.go  ANSI helpers
```

Tests exercise real behavior (temp `HOME` for config, `httptest` for the API) rather than mocks.

The [devenv](https://devenv.sh) shell provides the tooling: `build`, `test`, `test-coverage`, `test-race`, `lint`, `fmt`, `vet`, `ci`, `clean`.

</details>

<details>
<summary>Releasing</summary>

Versions are git tags ([SemVer](https://semver.org)). In the devenv shell, `patch` / `minor` / `major` bump and push a tag after a confirmation prompt; pushing a `vX.Y.Z` tag triggers GitHub Actions to build binaries (Linux/macOS/Windows, amd64 + arm64) and publish a release.

Manual: `git tag v1.2.3 && git push origin v1.2.3`. Rollback: `git tag -d v1.2.3 && git push origin --delete v1.2.3` (delete the GitHub release by hand if it already built).

</details>
```