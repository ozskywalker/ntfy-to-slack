# Claude Development Guidelines

This file contains guidelines and reminders for AI assistants working on this project.

## Conventional Commits for Build Pipeline

When making changes to the build pipeline, CI/CD workflows, or related infrastructure:

### Use `build:` prefix for build system changes
- ✅ `build: update golangci-lint configuration`
- ✅ `build: add security scanning to CI pipeline`  
- ✅ `build: configure cross-platform releases`
- ❌ `fix: resolve golangci-lint configuration issues` (appears in changelog)
- ❌ `feat: add automated release pipeline` (appears in changelog)

### Build-related changes that should use `build:` prefix:
- GitHub Actions workflows (`.github/workflows/`)
- GoReleaser configuration (`.goreleaser.yml`)
- Linting configuration (`.golangci.yml`, lint settings)
- Docker build configurations
- Makefile changes
- CI/CD pipeline modifications
- Build script updates
- Package manager configuration changes
- Dependency management for build tools

### The `build:` type is filtered out of changelogs
According to the GoReleaser configuration in `.goreleaser.yml`, the following commit types are excluded from user-facing changelogs:
- `docs:`
- `test:`
- `build:`
- `ci:`
- `refactor:`
- `style:`

### When to use other commit types:
- `feat:` - New user-facing features (appears in changelog)
- `fix:` - Bug fixes affecting users (appears in changelog)
- `sec:` - Security-related changes (appears in changelog)
- `perf:` - Performance improvements (appears in changelog)
- `docs:` - Documentation changes (filtered out)
- `test:` - Test changes (filtered out)
- `ci:` - CI/CD changes (filtered out, synonym for `build:`)
- `refactor:` - Code refactoring (filtered out)
- `style:` - Code style changes (filtered out)

## Testing Commands

When working on this project, always run tests using:
```bash
go test -v ./tests/...
```

## Build and Version Commands

To test the application:
```bash
go build -v ./cmd/ntfy-to-slack
./ntfy-to-slack -v
```

To test releases locally:
```bash
goreleaser build --single-target --snapshot --clean
```