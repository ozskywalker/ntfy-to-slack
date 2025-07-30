# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Professional CI/CD pipeline with automated releases
- Cross-platform build support (Windows, Linux, macOS)
- Build-time version injection system
- Security scanning with Gosec
- Linting with golangci-lint
- GPG-signed releases with checksums
- SBOM (Software Bill of Materials) generation
- Conventional commits support for automated changelog generation

## [2.0.0] - 2025-07-23

### Added
- Post-processing support with Mustache templates and webhooks
- Improved error handling and resilience with automatic reconnection
- Enhanced structured logging with configurable levels
- Modular architecture with interface-driven design

[Unreleased]: https://github.com/ozskywalker/ntfy-to-slack/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/ozskywalker/ntfy-to-slack/releases/tag/v2.0.0