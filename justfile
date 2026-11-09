[private]
default:
    @just --list

version := `git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev"`
build_date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
commit_sha := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`

# Shared by build and build-release so the two stay in sync.
ldflags := "-X main.version=" + version + " -X main.buildDate=" + build_date + " -X main.commitSHA=" + commit_sha

# Build with version info
build:
    go build -ldflags "{{ ldflags }}" -o mango .

# Build an optimized static binary
build-release:
    CGO_ENABLED=0 go build -ldflags "-w -s {{ ldflags }}" -o mango .

# Show version info
version:
    @echo {{ version }}

# Run tests
test:
    go test ./...

# Run tests with the race detector
test-race:
    go test ./... -race

# Run tests with an HTML coverage report
test-coverage:
    go test ./... -cover -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run benchmarks
bench:
    go test ./... -bench=. -benchmem

# Format code
fmt:
    go fmt ./...

# Vet code
vet:
    go vet ./...

# Lint code
lint:
    golangci-lint run

# Lint, vet, and run race and coverage tests
ci: lint vet test-race
    go test ./... -cover
    @echo "CI checks passed"

# Remove build and coverage artifacts
clean:
    rm -f mango coverage.out coverage.html
    go clean -testcache

# Tag and push a release
bump level:
    #!/usr/bin/env bash
    set -euo pipefail
    case "{{ level }}" in major|minor|patch) ;; *)
      echo "usage: just bump <major|minor|patch>"; exit 1 ;;
    esac
    if [ -n "$(git status --porcelain)" ]; then
      echo "Working directory is not clean; commit or stash first."; exit 1
    fi

    CURRENT=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    NEW=$(echo "$CURRENT" | awk -F. -v l="{{ level }}" '
      { ma=$1; sub(/^v/,"",ma); mi=$2; pa=$3; sub(/[-+].*/,"",pa) }
      l=="major"{print "v" ma+1 ".0.0"}
      l=="minor"{print "v" ma "." mi+1 ".0"}
      l=="patch"{print "v" ma "." mi "." pa+1}
    ')

    echo "$CURRENT -> $NEW (this pushes a tag and triggers a release build)"
    printf "Continue? (y/N): "
    read -r CONFIRM
    case "$CONFIRM" in
      [yY]|[yY][eE][sS])
        git tag "$NEW" && git push origin "$NEW"
        echo "Pushed $NEW"
        # Prime the Go proxy so `go install @latest` sees $NEW right away
        # instead of waiting for the proxy to index it lazily. Best-effort:
        # 5 tries over ~10s; if it never lands, @latest self-heals.
        MOD=$(go list -m)
        for i in 1 2 3 4 5; do
          if curl -sf "https://proxy.golang.org/$MOD/@v/$NEW.info" >/dev/null; then
            echo "Primed Go proxy for $NEW"; break
          fi
          sleep 2
        done
        ;;
      *) echo "Cancelled." ;;
    esac
