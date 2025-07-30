# Version and Build Information

This document explains versioning and build-time information injection.

## Versioning Strategy

The application follows [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., 1.2.3)
- **MAJOR**: Incompatible API changes
- **MINOR**: Backward compatible functionality additions
- **PATCH**: Backward compatible bug fixes

## Build-Time Version Injection

Version information is injected at build time using Go's `-ldflags` parameter in the `internal/version` package.

### Release Process

1. **Create and push a semantic version tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically**:
   - Runs all tests
   - Builds cross-platform binaries
   - Injects version information
   - Creates GitHub release with changelog
   - Signs artifacts with GPG
   - Generates SBOM

3. **Check version information**:
   ```bash
   ./ntfy-to-slack --version
   ```

## Manual Testing

Test releases locally:
```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test build
goreleaser build --single-target --snapshot --clean

# Full test (requires tag)
goreleaser release --snapshot --clean
```