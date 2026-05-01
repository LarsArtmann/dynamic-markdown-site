# Migration to Nix Flakes — Proposal

**Author:** Lars Artmann
**Date:** 2026-04-21
**Status:** Draft — Pending Review
**Scope:** Replace ad-hoc toolchain management with a deterministic Nix Flakes setup

---

## Table of Contents

- [1. Executive Summary](#1-executive-summary)
- [2. Current State Analysis](#2-current-state-analysis)
  - [2.1 Toolchain Dependencies](#21-toolchain-dependencies)
  - [2.2 Current Development Workflow](#22-current-development-workflow)
  - [2.3 Current Build & CI Pipeline](#23-current-build--ci-pipeline)
  - [2.4 Pain Points](#24-pain-points)
- [3. Proposed Architecture](#3-proposed-architecture)
  - [3.1 Flake Structure](#31-flake-structure)
  - [3.2 Package Outputs](#32-package-outputs)
  - [3.3 Development Shell](#33-development-shell)
  - [3.4 OCI Image via Nix](#34-oci-image-via-nix)
  - [3.5 Checks (CI Replacements)](#35-checks-ci-replacements)
  - [3.6 Formatter](#36-formatter)
  - [3.7 Apps (CLI Entry Points)](#37-apps-cli-entry-points)
- [4. File Layout](#4-file-layout)
- [5. Detailed Implementation](#5-detailed-implementation)
  - [5.1 flake.nix — Full Reference](#51-flakenix--full-reference)
  - [5.2 justfile Integration](#52-justfile-integration)
  - [5.3 .envrc for direnv](#53-envrc-for-direnv)
  - [5.4 CI Migration Path](#54-ci-migration-path)
- [6. Migration Steps (Action Plan)](#6-migration-steps-action-plan)
- [7. What Gets Replaced vs. Kept](#7-what-gets-replaced-vs-kept)
- [8. Risks & Mitigations](#8-risks--mitigations)
- [9. Success Criteria](#9-success-criteria)
- [10. Open Questions](#10-open-questions)

---

## 1. Executive Summary

This project currently relies on manually installed tools (Go 1.26.1, `templ`, `golangci-lint`, `golines`, `just`) with version pinning scattered across `go.mod`, the Dockerfile, and the CI workflow. This creates friction:

- Onboarding requires reading docs and installing 5+ tools
- CI and local environments can drift (e.g., `TEMPL_VERSION` pinned in `.github/workflows/docker.yml` but not enforced locally)
- The Dockerfile duplicates build logic that the justfile already defines
- Cache corruption issues (documented in `TODO_LIST.md`) are harder to reproduce

**Nix Flakes** solve this by declaring **all** dependencies, build steps, and dev environments in a single `flake.nix` that is:

- **Locked** — `flake.lock` ensures bit-for-bit reproducibility
- **Hermetic** — no hidden dependency on the host system
- **Multi-output** — one file produces the binary, OCI image, dev shell, lints, and tests
- **Instant** — `nix develop` gives you a fully equipped shell in seconds (after first build)

---

## 2. Current State Analysis

### 2.1 Toolchain Dependencies

The following tools are required to develop, build, and release this project:

| Tool               | Version             | Source                    | Pinned Where                      |
| ------------------ | ------------------- | ------------------------- | --------------------------------- |
| Go                 | 1.26.1              | `go.mod` directive        | `go.mod` + CI `setup-go`          |
| `templ` CLI        | v0.3.1001           | `go install` / Dockerfile | Dockerfile line 22, CI env var    |
| `golangci-lint`    | latest              | system install            | CI action (latest), local: manual |
| `golines`          | latest              | `go install`              | justfile `fix` task               |
| `just`             | latest              | system install            | README prerequisite               |
| `d2` (Terrastruct) | v0.7.1 (via go.mod) | Go dependency             | `go.mod` (runtime dep, not CLI)   |
| Docker / BuildKit  | any                 | system install            | CI service                        |

### 2.2 Current Development Workflow

```
1. Install Go 1.26.1 manually
2. go install github.com/a-h/templ/cmd/templ@v0.3.1001
3. Install golangci-lint (brew, curl, etc.)
4. Install just (brew, cargo, etc.)
5. templ generate          # code-gen .templ → .go
6. just test               # go test ./... -cover
7. just lint               # golangci-lint run ./...
8. just run-dev            # go run ./cmd/dynamic-markdown-site -dev -root ./content
```

### 2.3 Current Build & CI Pipeline

**Dockerfile** (multi-stage):

```
Stage 1 (builder):
  golang:1.26-alpine → apk add git → go install templ → go mod download → templ generate → CGO_ENABLED=0 go build (static, ldflags for version)

Stage 2 (runtime):
  distroless/static-debian13:nonroot → copy binary → expose 8080
```

**GitHub Actions** (`docker.yml`):

```
Jobs:
  build:  checkout → setup-go → templ generate → test -race → docker build+push (amd64+arm64) → attestation
  lint:   checkout → setup-go → templ generate → golangci-lint
  security-scan: trivy vulnerability scan on built image
```

### 2.4 Pain Points

| Pain Point                                                                                                                               | Impact                                              |
| ---------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| **templ version drift** — pinned in Dockerfile (line 22) and CI (`TEMPL_VERSION` env), but local installs use `@latest` or forget to pin | Broken builds when templ API changes                |
| **golangci-lint version drift** — CI uses `latest` via action, local may differ                                                          | Lint passes locally but fails in CI (or vice versa) |
| **golines not pinned** — `just fix` calls `golines -w .` but no version constraint                                                       | Formatting differs across machines                  |
| **Onboarding friction** — 5 tools to install manually before first contribution                                                          | New contributors give up                            |
| **Dockerfile duplicates justfile** — build logic exists in both places                                                                   | Maintenance burden, divergence risk                 |
| **Cache corruption** (TODO_LIST.md) — Go build cache issues hard to reproduce                                                            | "Works on my machine" debugging                     |
| **No `nix develop` equivalent** — can't enter a fully equipped shell instantly                                                           | Time wasted setting up environments                 |

---

## 3. Proposed Architecture

### 3.1 Flake Structure

```
flake.nix         # Single source of truth for all builds and environments
flake.lock        # Pinned dependency graph (auto-generated, committed)
```

The flake will provide these outputs:

| Output                  | Type               | Purpose                                                             |
| ----------------------- | ------------------ | ------------------------------------------------------------------- |
| `packages.default`      | Go binary          | The `dynamic-markdown-site` executable                              |
| `packages.oci-image`    | Docker/OCI tarball | Minimal container image (replaces Dockerfile)                       |
| `devShells.default`     | Shell env          | All tools for development (Go, templ, golangci-lint, golines, just) |
| `checks.test`           | Nix check          | Run `go test ./...`                                                 |
| `checks.lint`           | Nix check          | Run `golangci-lint run ./...`                                       |
| `checks.templ-generate` | Nix check          | Verify `templ generate` produces no diff                            |
| `formatter`             | Nix formatter      | `nix fmt` runs `golines`                                            |
| `apps.run-dev`          | Nix app            | `nix run .#run-dev` starts dev server                               |

### 3.2 Package Outputs

**`packages.default`** — The compiled binary:

```nix
packages.default = pkgs.buildGoModule {
  pname = "dynamic-markdown-site";
  version = "0.1.0"; # or derived from git tag

  src = ./.;

  vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # updated via nix build

  nativeBuildInputs = [ templ ];

  preBuild = ''
    templ generate
  '';

  ldflags = [
    "-s" "-w"
    "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${version}"
    "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${self.rev or "dirty"}"
    "-X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=1970-01-01T00:00:00Z"
  ];

  tags = [ "netgo" ];

  CGO_ENABLED = 0;

  meta = {
    description = "Type-safe markdown-to-website converter";
    mainProgram = "dynamic-markdown-site";
  };
};
```

Key decisions:

- `vendorHash` — computed by `nix build`, then pinned. Every `go.sum` change requires updating this hash (same pattern as all Go packages in nixpkgs).
- `preBuild` runs `templ generate` so the Go compilation sees generated `_templ.go` files.
- `ldflags` inject version information matching the current Dockerfile behavior (lines 51-56).
- `CGO_ENABLED = 0` with `tags = [ "netgo" ]` produces a statically linked binary, matching the Dockerfile.

### 3.3 Development Shell

**`devShells.default`** — Complete development environment:

```nix
devShells.default = pkgs.mkShell {
  buildInputs = with pkgs; [
    go_1_26
    templ
    golangci-lint
    golines
    just
  ];

  shellHook = ''
    echo "dynamic-markdown-site dev shell"
    echo "Go:        $(go version)"
    echo "templ:     $(templ version)"
    echo "golangci:  $(golangci-lint version)"
    echo "just:      $(just --version)"
    echo ""
    echo "Run 'just' to see available tasks."
  '';
};
```

This replaces all manual installation steps. `nix develop` (or `direnv` with `.envrc`) provides the entire toolchain.

**Why not `go` from `buildGoModule`'s go?** The devShell should use the same Go version as the build. We'll define the Go version once and reference it in both places.

### 3.4 OCI Image via Nix

**`packages.oci-image`** — Replaces the Dockerfile entirely:

```nix
packages.oci-image = pkgs.dockerTools.buildLayeredImage {
  name = "ghcr.io/larsartmann/dynamic-markdown-site";
  tag = self.shortRev or "latest";

  contents = [
    self.packages.${system}.default
    pkgs.cacert
    pkgs.tzdata
  ];

  config = {
    User = "65532:65532";
    ExposedPorts = { "8080/tcp" = { }; };
    Env = [
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
      "DYNAMIC_MARKDOWN_PORT=8080"
      "DYNAMIC_MARKDOWN_LOG_LEVEL=info"
      "DYNAMIC_MARKDOWN_CACHE=true"
      "DYNAMIC_MARKDOWN_ROOT=/content"
    ];
    Volumes = { "/content" = { }; };
    Cmd = [ "/bin/dynamic-markdown-site" "-root" "/content" "-port" "8080" "-cache" ];
  };

  maxLayers = 120;
};
```

**Advantages over the Dockerfile:**

| Aspect          | Dockerfile                       | Nix OCI Image                                       |
| --------------- | -------------------------------- | --------------------------------------------------- |
| Reproducibility | Depends on Alpine package state  | Bit-for-bit reproducible (fixed-output derivations) |
| Layer caching   | Manual `COPY` ordering           | Automatic optimal layer splitting                   |
| Build time      | Re-downloads deps on cache miss  | Nix store caches every dependency                   |
| Security        | Alpine + ca-certificates install | Only what's explicitly listed in `contents`         |
| Cross-platform  | Requires Docker/BuildKit         | Pure Nix, no Docker daemon needed for build         |
| Size            | distroless (~2MB base)           | Comparable (only binary + certs + tzdata)           |

### 3.5 Checks (CI Replacements)

```nix
checks = {
  test = pkgs.runCommand "test" { } ''
    ${pkgs.go_1_26}/bin/go test ./... -cover -race
    touch $out
  '';

  lint = pkgs.runCommand "lint" { } ''
    ${pkgs.golangci-lint}/bin/golangci-lint run ./...
    touch $out
  '';

  templ-generate = pkgs.runCommand "templ-generate-check" { } ''
    cp -r ${./.} src && chmod -R u+w src && cd src
    ${pkgs.templ}/bin/templ generate
    if [ -n "$(git diff --name-only '*_templ.go')" ]; then
      echo "ERROR: templ generate produces diff. Run 'templ generate' and commit."
      exit 1
    fi
    touch $out
  '';
};
```

These run via `nix flake check` — a single command that validates everything.

### 3.6 Formatter

```nix
formatter = pkgs.writeShellApplication {
  name = "format";
  runtimeInputs = [ pkgs.golines ];
  text = "golines -w .";
};
```

Usage: `nix fmt`

### 3.7 Apps (CLI Entry Points)

```nix
apps = {
  run-dev = {
    type = "app";
    program = "${pkgs.writeShellScriptBin "run-dev" ''
      ${pkgs.go_1_26}/bin/go run ./cmd/dynamic-markdown-site -dev -root ./content
    ''}/bin/run-dev";
  };
};
```

Usage: `nix run .#run-dev`

---

## 4. File Layout

Files to **add**:

| File         | Purpose                                       |
| ------------ | --------------------------------------------- |
| `flake.nix`  | All build/dev/ci definitions                  |
| `flake.lock` | Pinned dependency versions (auto-generated)   |
| `.envrc`     | direnv integration (optional but recommended) |

Files to **modify**:

| File         | Change                                                           |
| ------------ | ---------------------------------------------------------------- |
| `justfile`   | Add `nix-` prefixed tasks, keep existing tasks for non-Nix users |
| `.gitignore` | Add `result`, `result-*` (Nix build symlinks)                    |

Files to **keep unchanged** (at least initially):

| File                           | Reason                                                                        |
| ------------------------------ | ----------------------------------------------------------------------------- |
| `Dockerfile`                   | Backwards compatibility; can be deprecated once Nix OCI image is proven in CI |
| `.github/workflows/docker.yml` | Will be replaced by a Nix-based workflow in Phase 3                           |
| `go.mod` / `go.sum`            | Nix reads these; no changes needed                                            |
| `.golangci.yml`                | Used by `golangci-lint` regardless of how it's installed                      |

---

## 5. Detailed Implementation

### 5.1 flake.nix — Full Reference

```nix
{
  description = "dynamic-markdown-site — Type-safe markdown-to-website converter";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        version = "0.1.0";

        # Shared Go version — single source of truth
        go = pkgs.go_1_26;

        # Shared tool versions (matching current project requirements)
        templ = pkgs.templ;
        golangci-lint = pkgs.golangci-lint;
        golines = pkgs.golines;

        # The main Go package
        dynamic-markdown-site = pkgs.buildGoModule {
          pname = "dynamic-markdown-site";
          inherit version;

          src = self;

          # Update this hash after changing go.mod/go.sum:
          #   nix build .#default 2>&1 | grep "got:"
          vendorHash = ""; # Set to empty string initially, nix will tell you the correct hash

          nativeBuildInputs = [ templ ];

          preBuild = ''
            templ generate
          '';

          ldflags = [
            "-s" "-w"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${version}"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${self.rev or "dirty"}"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=1970-01-01T00:00:00Z"
          ];

          tags = [ "netgo" ];

          env.CGO_ENABLED = 0;

          meta = {
            description = "Type-safe, high-performance markdown-to-website converter";
            mainProgram = "dynamic-markdown-site";
            license = pkgs.lib.licenses.unfree; # Proprietary per LICENSE
          };
        };

      in
      {
        # --- Packages ---
        packages = {
          default = dynamic-markdown-site;

          oci-image = pkgs.dockerTools.buildLayeredImage {
            name = "ghcr.io/larsartmann/dynamic-markdown-site";
            tag = self.shortRev or "latest";

            contents = [
              dynamic-markdown-site
              pkgs.cacert
              pkgs.tzdata
            ];

            config = {
              User = "65532:65532";
              ExposedPorts = { "8080/tcp" = { }; };
              Env = [
                "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
                "DYNAMIC_MARKDOWN_PORT=8080"
                "DYNAMIC_MARKDOWN_LOG_LEVEL=info"
                "DYNAMIC_MARKDOWN_CACHE=true"
                "DYNAMIC_MARKDOWN_ROOT=/content"
              ];
              Volumes = { "/content" = { }; };
              Cmd = [ "/bin/dynamic-markdown-site" "-root" "/content" "-port" "8080" "-cache" ];
            };

            maxLayers = 120;
          };
        };

        # --- Development Shell ---
        devShells.default = pkgs.mkShell {
          buildInputs = [
            go
            templ
            golangci-lint
            golines
            pkgs.just
          ];

          shellHook = ''
            echo "dynamic-markdown-site dev shell"
            echo "Go:       $(go version)"
            echo "templ:    $(templ version)"
            echo "lint:     $(golangci-lint version --short)"
            echo "golines:  $(golines --version)"
            echo "just:     $(just --version)"
            echo ""
            echo "Quick start:"
            echo "  just run-dev    # Start dev server"
            echo "  just test       # Run tests"
            echo "  just lint       # Run linter"
            echo "  just generate   # Regenerate templ files"
          '';
        };

        # --- Checks ---
        checks =
          let
            buildInputs = [ go templ ];
          in
          {
            test = pkgs.runCommand "test" { inherit buildInputs; } ''
              cd ${self}
              templ generate
              go test ./... -cover -race
              touch $out
            '';

            lint = pkgs.runCommand "lint" { buildInputs = buildInputs ++ [ golangci-lint ]; } ''
              cd ${self}
              templ generate
              golangci-lint run ./...
              touch $out
            '';
          };

        # --- Formatter ---
        formatter = pkgs.writeShellApplication {
          name = "format";
          runtimeInputs = [ golines ];
          text = "golines -w .";
        };

        # --- Apps ---
        apps.run-dev = {
          type = "app";
          program = "${pkgs.writeShellScriptBin "run-dev" ''
            cd ${self}
            templ generate
            go run ./cmd/dynamic-markdown-site -dev -root ./content
          ''}/bin/run-dev";
        };
      }
    );
}
```

### 5.2 justfile Integration

Add Nix-aware tasks to the existing justfile. The original tasks remain for non-Nix users:

```justfile
# --- Nix tasks (new) ---

# Enter Nix development shell
nix-shell:
    nix develop

# Build via Nix
nix-build:
    nix build .#default
    @echo "Binary: ./result/bin/dynamic-markdown-site"

# Build OCI image via Nix
nix-image:
    nix build .#oci-image
    @echo "Image: ./result"
    @echo "Load:  docker load < ./result"

# Run all Nix checks (test + lint)
nix-check:
    nix flake check

# Update Nix flake inputs
nix-update:
    nix flake update

# Show flake metadata
nix-info:
    nix flake metadata

# --- Existing tasks (unchanged) ---
# ... (all existing tasks remain as-is)
```

### 5.3 .envrc for direnv

For automatic shell activation when entering the project directory:

```bash
# .envrc
use flake
```

This provides the full devShell automatically — no `nix develop` needed. Every terminal in the project directory has all tools available.

### 5.4 CI Migration Path

**Phase 1** (additive — no disruption):

```yaml
# .github/workflows/nix.yml
name: Nix CI
on:
  push:
    branches: [master]
  pull_request:
    branches: [master]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: DeterminateSystems/nix-installer-action@main
      - uses: DeterminateSystems/magic-nix-cache-action@main
      - run: nix flake check
```

**Phase 2** (dual pipeline):

- Both `docker.yml` and `nix.yml` run in parallel
- Compare outputs for equivalence
- Build OCI image from Nix, push alongside Docker-built image

**Phase 3** (full migration):

- Remove `docker.yml`, replace with `nix.yml`
- Dockerfile kept for reference but no longer used in CI
- Nix builds the OCI image for all platforms

---

## 6. Migration Steps (Action Plan)

### Phase 1: Foundation (Day 1)

| Step | Action                                           | Verify                     |
| ---- | ------------------------------------------------ | -------------------------- |
| 1.1  | Create `flake.nix` with `devShells.default` only | `nix develop` works        |
| 1.2  | Create `.envrc` with `use flake`                 | `direnv allow` loads tools |
| 1.3  | Add `result` and `result-*` to `.gitignore`      | Git ignores Nix symlinks   |
| 1.4  | Run `nix develop -c just test`                   | Tests pass in Nix shell    |
| 1.5  | Run `nix develop -c just lint`                   | Lint passes in Nix shell   |
| 1.6  | Commit `flake.nix`, `flake.lock`, `.envrc`       | Repo has Nix support       |

### Phase 2: Package Build (Day 2)

| Step | Action                                                         | Verify                                     |
| ---- | -------------------------------------------------------------- | ------------------------------------------ |
| 2.1  | Add `packages.default` to flake.nix                            | `nix build` produces binary                |
| 2.2  | Fix `vendorHash` (run build, copy reported hash)               | Build succeeds                             |
| 2.3  | Verify binary runs: `./result/bin/dynamic-markdown-site -help` | Output matches expected                    |
| 2.4  | Add `checks.test` and `checks.lint`                            | `nix flake check` passes                   |
| 2.5  | Add `formatter` and `apps.run-dev`                             | `nix fmt` and `nix run .#run-dev` work     |
| 2.6  | Update `justfile` with Nix tasks                               | `just nix-build` and `just nix-check` work |

### Phase 3: OCI Image (Day 3)

| Step | Action                                                  | Verify                                   |
| ---- | ------------------------------------------------------- | ---------------------------------------- |
| 3.1  | Add `packages.oci-image` to flake.nix                   | `nix build .#oci-image` succeeds         |
| 3.2  | Load and test: `docker load < result` then `docker run` | Container serves on port 8080            |
| 3.3  | Compare image size with Dockerfile-built image          | Within 20% of current size               |
| 3.4  | Verify health endpoint works in container               | `curl localhost:8080/health` returns 200 |

### Phase 4: CI Integration (Day 4-5)

| Step | Action                                          | Verify                               |
| ---- | ----------------------------------------------- | ------------------------------------ |
| 4.1  | Create `.github/workflows/nix.yml`              | Runs alongside existing `docker.yml` |
| 4.2  | Add `DeterminateSystems/nix-installer-action`   | CI has Nix                           |
| 4.3  | Add `magic-nix-cache-action` for caching        | CI builds are fast                   |
| 4.4  | Configure OCI image push from Nix build         | Nix-built image pushed to GHCR       |
| 4.5  | Run both pipelines in parallel for 2 weeks      | No discrepancies found               |
| 4.6  | Remove `docker.yml` and Dockerfile (or archive) | Single pipeline                      |

### Phase 5: Polish (Day 6-7)

| Step | Action                                                    | Verify                    |
| ---- | --------------------------------------------------------- | ------------------------- |
| 5.1  | Update `README.md` with Nix instructions                  | Docs reflect new workflow |
| 5.2  | Update `AGENTS.md` with Nix commands                      | AI agents use Nix         |
| 5.3  | Add `CONTRIBUTING.md` with `nix develop` as primary setup | New contributors use Nix  |
| 5.4  | Remove manual installation instructions from README       | No conflicting docs       |
| 5.5  | Add `nix fmt` to git pre-commit hook (optional)           | Formatting enforced       |

---

## 7. What Gets Replaced vs. Kept

| Component                             | Status                 | Notes                                             |
| ------------------------------------- | ---------------------- | ------------------------------------------------- |
| `Dockerfile`                          | **Archived** (Phase 4) | Kept in repo as `Dockerfile.legacy` for reference |
| `.github/workflows/docker.yml`        | **Replaced** (Phase 4) | By `nix.yml`                                      |
| `justfile`                            | **Kept + extended**    | Nix tasks added; existing tasks work in devShell  |
| `go.mod` / `go.sum`                   | **Kept**               | Nix reads these for `buildGoModule`               |
| `.golangci.yml`                       | **Kept**               | Used by `golangci-lint` from devShell             |
| `.editorconfig`                       | **Kept**               | Unrelated to Nix                                  |
| `.buildflow.yml`                      | **Kept**               | Unrelated to Nix                                  |
| `.gitattributes`                      | **Kept**               | Unrelated to Nix                                  |
| Manual tool installs                  | **Eliminated**         | `nix develop` replaces all manual setup           |
| `templ` version pinning in Dockerfile | **Consolidated**       | Single pin in `flake.nix`                         |
| `TEMPL_VERSION` CI env var            | **Eliminated**         | Nix provides pinned `templ`                       |

---

## 8. Risks & Mitigations

| Risk                                            | Severity | Mitigation                                                                                                                      |
| ----------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `go_1_26` not yet in nixpkgs                    | High     | Check `nixpkgs` for Go 1.26 availability; if missing, use `gotools` overlay or build Go from source via `pkgs.go_1_26.override` |
| `templ` not in nixpkgs or wrong version         | Medium   | Package `templ` as a flake input or use `buildGoModule` to build it inline                                                      |
| `golines` not in nixpkgs                        | Low      | Build via `buildGoModule` inline or find in nixpkgs                                                                             |
| Nix learning curve for contributors             | Medium   | Keep justfile tasks; Nix is optional for development, not required                                                              |
| `vendorHash` updates on every `go.mod` change   | Low      | Document the process; automate with `nix-update` or `nvfetcher`                                                                 |
| CI runner performance with Nix                  | Medium   | Use `magic-nix-cache-action` and GitHub Actions cache                                                                           |
| macOS vs Linux differences                      | Low      | `flake-utils.lib.eachDefaultSystem` handles both; CI runs on Linux                                                              |
| D2 CLI not needed at build time (runtime dep)   | None     | D2 is a Go library dependency; `buildGoModule` handles it via `go.mod`                                                          |
| Cross-compilation (amd64 + arm64) for OCI image | Medium   | Use `pkgsCross.aarch64-multiplatform` for arm64, or keep Docker buildx for multi-arch                                           |

---

## 9. Success Criteria

The migration is complete when:

- [ ] `nix develop` provides all development tools (Go, templ, golangci-lint, golines, just)
- [ ] `nix build` produces a working static binary
- [ ] `nix flake check` runs tests and lint
- [ ] `nix build .#oci-image` produces a container image under 20MB
- [ ] CI uses Nix exclusively (no Dockerfile, no manual tool installs)
- [ ] New contributors can start developing with `nix develop` alone
- [ ] All tool versions are pinned in exactly one place (`flake.nix` + `flake.lock`)
- [ ] The existing justfile tasks work identically inside the Nix devShell
- [ ] `direnv` + `.envrc` provides seamless shell integration

---

## 10. Open Questions

1. **Go 1.26 in nixpkgs?** — Need to verify `pkgs.go_1_26` exists in the targeted nixpkgs revision. If not, an overlay or alternative Go version strategy is needed.

2. **templ in nixpkgs?** — `pkgs.templ` may not be available or may be an older version. Alternatives:
   - Build from source: `buildGoModule { src = fetchFromGitHub { ... } }`
   - Use `gomod2nix` for precise Go tool versioning
   - Define as a separate flake input from a templ flake registry

3. **Multi-arch OCI images?** — `dockerTools.buildLayeredImage` builds for one architecture. For amd64+arm64:
   - Build twice with different `pkgsCross` and push both manifests
   - Or keep Docker buildx for multi-arch CI (hybrid approach)
   - Or use `nix-community/nixix` or similar multi-arch tooling

4. **D2 system dependencies?** — The D2 Go library (`oss.terrastruct.com/d2`) may require system libraries (fontconfig, etc.) for SVG rendering. Need to verify if `buildGoModule` handles this or if additional `buildInputs` are needed.

5. **`vendorHash` automation?** — Should we use `gomod2nix` or `nvfetcher` to automate the `vendorHash` update process? This adds a dependency but reduces manual friction.

6. **Nix on CI runner image size?** — Nix installation on GitHub Actions runners takes ~30s with the DeterminateSystems installer. Acceptable, but worth monitoring.

7. **Flake registry?** — Should this flake be published to the Nix flake registry for discoverability, or kept private?

---

_This proposal is a living document. Update as implementation progresses and open questions are resolved._
