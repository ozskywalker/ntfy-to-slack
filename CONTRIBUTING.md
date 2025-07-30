# Contributing to ntfy-to-slack

Thank you for your interest in contributing! This guide will help you understand our development process.

## Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automated changelog generation. All commit messages must follow this format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

- **`feat:`** - New features (appears in "Features" section)
- **`fix:`** - Bug fixes (appears in "Bug fixes" section)  
- **`sec:`** - Security-related changes (appears in "Security" section)
- **`perf:`** - Performance improvements (appears in "Performance" section)
- **`docs:`** - Documentation changes (filtered out of changelog)
- **`test:`** - Test changes (filtered out of changelog)
- **`build:`** - Build system changes (filtered out of changelog)
- **`ci:`** - CI/CD changes (filtered out of changelog)
- **`refactor:`** - Code refactoring (filtered out of changelog)
- **`style:`** - Code style changes (filtered out of changelog)

### Examples

**Good commit messages:**
```
feat: add user authentication system
fix: resolve memory leak in data processing
sec: validate input parameters to prevent injection
perf: optimize database query performance
docs: update API documentation
```

### Breaking Changes

For breaking changes, add `BREAKING CHANGE:` in the footer or use `!` after the type:

```
feat!: change API response format

BREAKING CHANGE: API now returns data in different structure
```

## Development Workflow

1. **Fork the repository** and create a feature branch
2. **Make your changes** following the code style
3. **Test your changes** using `go test ./...`
4. **Format your code** using `go fmt ./...`
5. **Vet your code** using `go vet ./...`
6. **Lint your code** using `golangci-lint run`
7. **Commit your changes** using conventional commit format
8. **Push to your fork** and create a pull request

## Code Guidelines

- Follow standard Go conventions and formatting
- Add tests for new functionality (maintain >70% coverage)
- Update documentation as needed
- Ensure all CI checks pass
- Keep commits focused and atomic

## Pull Request Process

1. Ensure your branch is up to date with the main branch
2. Include a clear description of the changes
3. Reference any related issues
4. Ensure all CI checks pass
5. Request review from maintainers