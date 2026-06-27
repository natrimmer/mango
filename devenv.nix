{
  pkgs,
  ...
}:

{
  languages.go.enable = true;
  packages = [
    pkgs.golangci-lint
    pkgs.pkgsite
    pkgs.git
    pkgs.nixd
  ];

  scripts = {
    # menu is the single source of truth for the command list, printed on shell
    # entry. hand-maintained.
    menu.exec = ''
      echo "mango devenv commands:"
      echo "  build / build-release  - build (with version info / optimized)"
      echo "  version                - show version info"
      echo "  test-code              - run tests"
      echo "  test-coverage          - tests with HTML coverage report"
      echo "  test-race              - tests with race detector"
      echo "  bench                  - run benchmarks"
      echo "  fmt / vet / lint       - format, vet, lint"
      echo "  ci                     - lint + vet + race + coverage"
      echo "  clean                  - remove build/coverage artifacts"
      echo "  bump <major|minor|patch> - tag and push a release"
    '';

    build.exec = ''
      go build -ldflags "$(ldflags)" -o mango .
    '';

    build-release.exec = ''
      CGO_ENABLED=0 go build -ldflags "-w -s $(ldflags)" -o mango .
    '';

    version.exec = ''
      git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev"
    '';

    test-code.exec = "go test ./...";
    test-race.exec = "go test ./... -race";
    bench.exec = "go test ./... -bench=. -benchmem";
    fmt.exec = "go fmt ./...";
    vet.exec = "go vet ./...";
    lint.exec = "golangci-lint run";

    test-coverage.exec = ''
      go test ./... -cover -coverprofile=coverage.out
      go tool cover -html=coverage.out -o coverage.html
      echo "Coverage report: coverage.html"
    '';

    clean.exec = ''
      rm -f mango coverage.out coverage.html
      go clean -testcache
    '';

    ci.exec = ''
      set -e
      golangci-lint run
      go vet ./...
      go test ./... -race
      go test ./... -cover
      echo "CI checks passed"
    '';

    # Shared ldflags so build/build-release stay in sync.
    ldflags.exec = ''
      V=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
      D=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
      C=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
      echo "-X main.version=$V -X main.buildDate=$D -X main.commitSHA=$C"
    '';

    bump.exec = ''
      LEVEL="$1"
      case "$LEVEL" in major|minor|patch) ;; *)
        echo "usage: bump <major|minor|patch>"; exit 1 ;;
      esac
      if [ -n "$(git status --porcelain)" ]; then
        echo "Working directory is not clean; commit or stash first."; exit 1
      fi

      CURRENT=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
      NEW=$(echo "$CURRENT" | awk -F. -v l="$LEVEL" '
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
    '';
  };

  enterShell = "menu";

  git-hooks.hooks = {
    # Formatting
    beautysh.enable = true;
    gofmt.enable = true;
    nixfmt-rfc-style.enable = true;
    # Linting
    shellcheck.enable = true;
    golangci-lint.enable = true;
    statix.enable = true;
    deadnix.enable = true;
    # Safety
    detect-private-keys.enable = true;
    check-added-large-files.enable = true;
    check-case-conflicts.enable = true;
    check-merge-conflicts.enable = true;
    check-executables-have-shebangs.enable = true;
    check-shebang-scripts-are-executable.enable = true;
  };
}
