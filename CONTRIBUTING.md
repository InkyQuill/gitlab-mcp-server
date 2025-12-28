# Contributing

Thank you for your interest in contributing to the GitLab MCP Server! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

This project adheres to a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to abide by its terms.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

1. **Go 1.23 or later**
   - [Download Go](https://go.dev/doc/install)
   - [Install via Homebrew](https://formulae.brew.sh/formula/go) (macOS)

2. **golangci-lint**
   - [Installation Guide](https://golangci-lint.run/welcome/install/#local-installation)

3. **Git** - For version control

4. **Make** (optional) - For using Makefile targets

### Development Setup

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gitlab-mcp-server.git
   cd gitlab-mcp-server
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/InkyQuill/gitlab-mcp-server.git
   ```
4. Install dependencies:
   ```bash
   make setup
   ```
5. Build the project:
   ```bash
   make build
   ```

## Development Workflow

### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. **Make your changes** following the project's coding standards

3. **Write tests** for your changes (see [Testing Guide](TESTING.md))

4. **Run tests** to ensure everything passes:
   ```bash
   make test
   # or
   go test -v ./...
   ```

5. **Run the linter**:
   ```bash
   golangci-lint run
   ```

6. **Commit your changes** with a clear, descriptive commit message:
   ```bash
   git add .
   git commit -m "Add feature: description of your change"
   ```

7. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

8. **Create a Pull Request** on GitHub

### Commit Message Guidelines

Write clear, descriptive commit messages:

- Use the imperative mood ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Provide context in the body if needed
- Reference issues when applicable: "Fix #123"

Example:
```
Add support for project labels

Implements listProjectLabels and createProjectLabel tools.
Includes tests and documentation updates.

Fixes #456
```

## Coding Standards

### Style Guide

Follow the [Go style guide](https://golang.org/doc/effective_go) and the project's [golangci-lint configuration](.golangci.yml).

Key points:
- Use `gofmt` for formatting
- Follow Go naming conventions
- Write clear, self-documenting code
- Add comments for exported functions and types
- Keep functions focused and small

### Code Review Checklist

Before submitting a PR, ensure:

- [ ] Code follows the project's style guide
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Tests are added for new functionality
- [ ] Documentation is updated if needed
- [ ] Commit messages are clear and descriptive
- [ ] PR description explains the changes and motivation

## Testing

This project maintains high test coverage (88.9%). See [TESTING.md](TESTING.md) for detailed testing guidelines.

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Run with race detection
go test -race ./pkg/... ./internal/...

# Run specific package tests
go test ./pkg/gitlab -v
```

### Writing Tests

- Write tests for all new functionality
- Use table-driven tests when appropriate
- Test both success and error cases
- Mock external dependencies (GitLab API)
- Aim for >80% coverage for new code

## Pull Request Process

1. **Update your branch** with the latest changes from `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-branch
   git rebase main
   ```

2. **Ensure all checks pass**:
   - Tests pass
   - Linter passes
   - CI checks pass

3. **Create the Pull Request**:
   - Provide a clear title and description
   - Reference related issues
   - Include screenshots/examples if applicable

4. **Respond to feedback**:
   - Address review comments
   - Update the PR as needed
   - Keep discussions focused and constructive

## What to Contribute

We welcome contributions in many forms:

- **Bug fixes**: Report and fix issues
- **New features**: Implement planned features from the [Roadmap](ROADMAP.md)
- **Documentation**: Improve docs, add examples, fix typos
- **Tests**: Increase test coverage
- **Performance**: Optimize code
- **Refactoring**: Improve code quality

### Feature Requests

Before implementing a major feature:
1. Check the [Roadmap](ROADMAP.md) to see if it's planned
2. Open an issue to discuss the feature
3. Wait for maintainer feedback before implementing

## Project Structure

```
gitlab-mcp-server/
├── cmd/              # Application entry points
├── pkg/              # Main package code
│   ├── gitlab/      # GitLab API integration
│   ├── toolsets/    # Toolset management
│   └── ...
├── internal/         # Internal packages
├── docs/             # Documentation
├── scripts/          # Utility scripts
└── ...
```

## Getting Help

- **Questions**: Open a GitHub issue with the "question" label
- **Bugs**: Open a GitHub issue with the "bug" label
- **Discussions**: Use GitHub Discussions for general questions

## Resources

- [Go Documentation](https://golang.org/doc/)
- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Using Pull Requests](https://help.github.com/articles/about-pull-requests/)
- [GitHub Help](https://help.github.com)

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
