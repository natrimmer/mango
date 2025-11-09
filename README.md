# Claude Commit

A simple CLI tool that uses the Claude API to generate Git commit messages from staged changes, following conventional commit best practices.

## Installation

### Option 1: Download binary

Download the pre-built binary for your platform from the [GitHub Releases page](https://github.com/natrimmer/claude_commit/releases/latest):

```bash
# Example for Linux (amd64)
curl -L https://github.com/natrimmer/claude_commit/releases/latest/download/claude_commit_linux_amd64 -o claude_commit
chmod +x claude_commit
sudo mv claude_commit /usr/local/bin/

# Example for macOS (intel)
curl -L https://github.com/natrimmer/claude_commit/releases/latest/download/claude_commit_darwin_amd64 -o claude_commit
chmod +x claude_commit
sudo mv claude_commit /usr/local/bin/

# Example for macOS (Apple Silicon)
curl -L https://github.com/natrimmer/claude_commit/releases/latest/download/claude_commit_darwin_arm64 -o claude_commit
chmod +x claude_commit
sudo mv claude_commit /usr/local/bin/
```

### Option 2: Using Go

```bash
go install github.com/natrimmer/claude_commit@latest
```

### Option 3: Build from source

```bash
git clone https://github.com/natrimmer/claude_commit.git
cd claude_commit
build  # or: go build
```

## Quick Start

```bash
# Get help
claude_commit
# or
claude_commit --help

# Check version
claude_commit --version

# Configure
claude_commit config -api-key "your-api-key"

# Generate commit message
git add .
claude_commit commit
```

## Commands

### Help and Version

```bash
claude_commit              # Show help
claude_commit --help       # Show help
claude_commit help         # Show help
claude_commit --version    # Show version info
```

### Configuration

```bash
# Configure with your API key (uses claude-3-7-sonnet-latest by default)
claude_commit config -api-key "your-api-key"

# Configure with specific model
claude_commit config -api-key "your-api-key" -model "claude-3-5-sonnet-latest"

# View current configuration
claude_commit view

# List available models
claude_commit models
```

### Generate Commit Messages

```bash
git add .                # Stage your changes
claude_commit commit     # Generate a commit message

# Advanced options
claude_commit commit --type feat                    # Force specific commit type
claude_commit commit --context "fixing bug #123"    # Provide additional context
claude_commit commit --count 3                      # Generate 3 options to choose from
claude_commit commit --dry-run                      # Show prompt without API call
claude_commit commit --verbose                      # Show full API interaction
claude_commit commit -v                             # Short form of --verbose
claude_commit commit --type fix --context "auth issue" --count 2  # Combine flags
```

## Available Models

- `claude-opus-4-0` - Most capable, slower and more expensive
- `claude-sonnet-4-0` - Balanced performance and speed
- `claude-3-7-sonnet-latest` - **Default** - Fast and efficient
- `claude-3-5-sonnet-latest` - Previous generation, reliable
- `claude-3-5-haiku-latest` - Fastest and most cost-effective
- `claude-3-opus-latest` - Previous generation, most capable

## Example Usage

### Configuration

```bash
$ claude_commit config -api-key "sk-ant-api03-..." -model "claude-3-7-sonnet-latest"
Configuration saved successfully
API Key: sk-a****...
Model: claude-3-7-sonnet-latest

$ claude_commit view
Current Configuration:
API Key: sk-a****...
Model: claude-3-7-sonnet-latest

$ claude_commit models
Available Models:
claude-opus-4-0
claude-sonnet-4-0
claude-3-7-sonnet-latest [CURRENT]
claude-3-5-sonnet-latest
claude-3-5-haiku-latest
claude-3-opus-latest
```

### Generating Commits

```bash
# Basic usage
$ git add .
$ claude_commit commit
⚙️  Analyzing git diff with Claude AI...
✓ Commit message generated

git commit -m "feat: add user authentication and password reset functionality"

# Force a specific commit type
$ claude_commit commit --type fix
⚙️  Analyzing git diff with Claude AI...
✓ Commit message generated

git commit -m "fix: resolve authentication timeout issue"

# Provide additional context
$ claude_commit commit --context "resolves issue #123"
⚙️  Analyzing git diff with Claude AI...
✓ Commit message generated

git commit -m "fix: prevent null pointer in user profile handler"

# Generate multiple options
$ claude_commit commit --count 3
⚙️  Analyzing git diff with Claude AI...
✓ Commit message options generated

1. feat: add user authentication system
2. feat: implement login and registration endpoints
3. feat: add JWT-based authentication middleware

# Dry run to see the prompt without API call
$ claude_commit commit --dry-run
Prompt being sent to Claude:
─────────────────────────────────────────
Generate a conventional commit message based on the following git diff.
[... full prompt displayed ...]
─────────────────────────────────────────

⚠️  Dry run mode - API not called

# Verbose mode to see full interaction
$ claude_commit commit --verbose
Prompt being sent to Claude:
─────────────────────────────────────────
Generate a conventional commit message based on the following git diff.
[... full prompt displayed ...]
─────────────────────────────────────────

⚙️  Analyzing git diff with Claude AI...
Raw API Response:
─────────────────────────────────────────
feat: add user authentication system
─────────────────────────────────────────

✓ Commit message generated

git commit -m "feat: add user authentication system"
```

### Version Information

```bash
$ claude_commit --version
Claude Commit v1.2.3
Build Date: 2024-01-15T10:30:00Z
Commit: abc1234
Generate conventional commit messages with Anthropic's Claude
```

## Advanced Commit Options

### --type flag

Force the LLM to use a specific commit type. Useful when you know the category of your changes and want to avoid ambiguity.

```bash
claude_commit commit --type feat    # Force feature type
claude_commit commit --type fix     # Force bug fix type
claude_commit commit --type docs    # Force documentation type
```

### --context flag

Provide additional context to help the LLM generate a more accurate commit message. This is especially useful when:
- The changes relate to a specific issue or ticket
- The purpose isn't obvious from the diff alone
- You want to emphasize a particular aspect of the changes

```bash
claude_commit commit --context "resolves issue #123"
claude_commit commit --context "breaking change for API v2"
claude_commit commit --context "performance optimization for large datasets"
```

### --count flag

Generate multiple commit message options. This addresses ambiguity by giving you choices when the nature of the change could be interpreted different ways (e.g., is it a "feat" or a "fix"?).

```bash
claude_commit commit --count 3    # Get 3 different options
claude_commit commit --count 5    # Get 5 different options
```

When using `--count`, the tool displays numbered options instead of a git command, allowing you to pick the most appropriate one.

### --dry-run flag

Show the prompt that will be sent to Claude without actually calling the API. Perfect for:
- Testing how different flags affect the prompt
- Understanding what information is sent to Claude
- Saving API costs during experimentation

```bash
claude_commit commit --dry-run
claude_commit commit --type feat --count 3 --dry-run    # See how flags affect prompt
```

When using `--dry-run`, the tool displays the complete prompt and exits without making an API call.

### --verbose flag (or -v)

Show the full API interaction including both the prompt sent and the raw response received. Useful for:
- Debugging issues with commit message generation
- Understanding how Claude interprets your changes
- Seeing the complete request/response cycle

```bash
claude_commit commit --verbose
claude_commit commit -v                    # Short form
claude_commit commit -v --type fix         # Combine with other flags
```

When using `--verbose`, the tool shows:
1. The prompt being sent to Claude
2. The "Analyzing..." message
3. The raw API response
4. The final formatted output

### Combining Flags

All flags can be combined for maximum control:

```bash
# Get 3 fix-type options with context
claude_commit commit --type fix --context "resolves #123" --count 3

# Get 2 feature options with context
claude_commit commit --type feat --context "new payment gateway" --count 2

# Test prompt generation without API call
claude_commit commit --type feat --context "new feature" --dry-run

# Debug full API interaction
claude_commit commit --verbose --count 3
```

## Commit Message Format

- Type prefix (feat, fix, docs, etc.)
- Lowercase throughout
- Imperative mood
- No period at end
- Format: `<type>: <description>`

## Conventional Commit Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring without functionality changes
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks, dependency updates, etc.
- `ci`: Continuous integration changes
- `build`: Changes that affect the build system or external dependencies
- `revert`: Reverts a previous commit

## How It Works

1. Reads your Anthropic API key from config (stored in `~/.claude-commit/config.json`)
2. Gets staged changes with `git diff --staged`
3. Sends the diff and detailed prompt to Claude API
4. Returns a formatted git commit command

## Configuration Storage

Your configuration is stored in a JSON file at `~/.claude-commit/config.json`. The API key is stored in plaintext, so ensure appropriate file permissions are set.

## Features

- Zero dependencies
- Follows conventional commit best practices
- Uses conventional commit format
- Configuration stored in `~/.claude-commit/config.json`
- API key masking for display security
- Colorized terminal output
- Version information with build details
- Comprehensive help system
- Multiple model support with easy switching
- Clean error handling and user feedback
- **Advanced commit options:**
  - Force specific commit types with `--type`
  - Provide additional context with `--context`
  - Generate multiple options with `--count`
  - Show prompt without API call with `--dry-run`
  - Debug with full visibility using `--verbose` or `-v`
  - Combine all flags for maximum control

## Development

### Building from Source

```bash
git clone https://github.com/natrimmer/claude_commit.git
cd claude_commit

# The devenv environment provides all necessary tools
# Install dependencies are handled automatically by devenv

# Run tests
test

# Build with version info
build

# Build release version
build-release

# Run all quality checks
ci
```

### Available Commands

When you enter the devenv shell, you'll have access to these commands:

- `build` - Build with version info
- `build-release` - Build optimized release binary
- `test` - Run tests
- `test-coverage` - Run tests with coverage
- `test-race` - Run tests with race detection
- `bench` - Run benchmark tests
- `lint` - Run linter
- `fmt` - Format code
- `vet` - Run go vet
- `version` - Show version information
- `clean` - Clean build artifacts
- `ci` - Run all CI checks

### Version Management

This project uses **Semantic Versioning (SemVer)**. Versions are managed through git tags:

```bash
# Create a new version tag
git tag v1.2.3
git push origin v1.2.3

# Build will automatically use the tag
build
./claude_commit --version  # Shows: Claude Commit v1.2.3
```

### Release Process

#### Quick Release (Recommended)

The development environment includes automated version increment scripts that handle the entire release process:

```bash
# For bug fixes and small changes
patch    # Increments v1.2.3 → v1.2.4

# For new features (backwards compatible)
minor    # Increments v1.2.3 → v1.3.0

# For breaking changes
major    # Increments v1.2.3 → v2.0.0
```

**What These Scripts Do:**

1. **Safety checks**: Ensure your working directory is clean (no uncommitted changes)
2. **Version calculation**: Automatically determine the next version number
3. **Confirmation prompt**: Show you what will happen and ask for confirmation
4. **Tag creation**: Create the new git tag locally
5. **Push tag**: Push the tag to trigger the automated build

Each script will show you:
- Current version
- Proposed new version
- Warning that this triggers a release build
- Confirmation prompt (defaults to "No" for safety)

#### Manual Release Process

If you prefer to handle versioning manually:

1. Update the code and make any necessary changes
2. Commit and push your changes to the main branch
3. Create and push a new tag with a version number (following semver):

```bash
git tag v0.1.0  # Change version as appropriate
git push origin v0.1.0
```

#### Automated Build Process

Once a tag is pushed (either via the scripts or manually), GitHub Actions will automatically:

- Build binaries for all supported platforms (Linux, macOS, Windows)
- Create a GitHub Release
- Upload the binaries to the release

**Supported Platforms:**

- Linux (amd64, arm64)
- macOS (amd64 - Intel, arm64 - Apple Silicon)
- Windows (amd64)

#### Version Strategy

We follow [Semantic Versioning (semver)](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes

#### Rollback

If you need to remove a tag:

```bash
# Delete local tag
git tag -d v1.2.3

# Delete remote tag
git push origin --delete v1.2.3
```

**Note**: If GitHub Actions has already created a release, you'll need to delete it manually from the GitHub web interface.

### Development Workflow

```bash
# Enter the development environment
cd claude_commit  # devenv activates automatically with direnv

# Make changes, then test
fmt      # Format code
lint     # Check for issues
test     # Run tests
ci       # Run full CI suite

# Build and test
build
./claude_commit --version
```
