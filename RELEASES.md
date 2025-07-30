# Release Guide

Quick release process for maintainers.

## Creating a Release

1. **Ensure all tests pass**:
   ```bash
   go test ./tests/...
   ```

2. **Create and push tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. **Verify release** at GitHub Releases page

## Semantic Versioning

- **MAJOR** (`v2.0.0`): Breaking changes
- **MINOR** (`v1.1.0`): New features, backward compatible
- **PATCH** (`v1.0.1`): Bug fixes, backward compatible

## Release Assets

Each release includes:
- Cross-platform binaries (Windows, Linux, macOS)
- SHA256 checksums
- GPG signatures (if configured)
- SBOM (Software Bill of Materials)

## GPG Signing Setup

1. Generate GPG key: `gpg --full-generate-key`
2. Add to GitHub secrets:
   - `GPG_PRIVATE_KEY`: Your private key
   - `PASSPHRASE`: Your GPG passphrase