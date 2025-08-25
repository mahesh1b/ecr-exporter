# Contributing to ECR Prometheus Exporter

Thank you for your interest in contributing! This document provides guidelines and information for contributors.

## Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/ecr-prometheus-exporter.git
   cd ecr-prometheus-exporter
   ```

2. **Install dependencies**
   ```bash
   go mod download
   make install-tools
   ```

3. **Set up pre-commit hooks** (optional but recommended)
   ```bash
   pip install pre-commit
   pre-commit install
   ```

## Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write code following Go best practices
   - Add tests for new functionality
   - Update documentation as needed

3. **Run local checks**
   ```bash
   make check  # Runs deps, lint, test, and security checks
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push and create a pull request**
   ```bash
   git push origin feature/your-feature-name
   ```

## Code Standards

### Go Code Style
- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and small
- Handle errors appropriately

### Testing
- Write unit tests for new functionality
- Aim for good test coverage
- Use table-driven tests where appropriate
- Mock external dependencies (AWS API calls)

### Commit Messages
Follow conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test additions/changes
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

## Pull Request Process

1. **Ensure CI passes**
   - All tests pass
   - Linting passes
   - Security scan passes
   - Build succeeds

2. **Update documentation**
   - Update README.md if needed
   - Add/update code comments
   - Update metrics documentation

3. **Describe your changes**
   - Clear PR title and description
   - Reference any related issues
   - Explain the motivation for changes

4. **Request review**
   - Wait for maintainer review
   - Address feedback promptly
   - Keep PR focused and small

## Testing

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -run TestTimestampLogic -v
```

### Testing with AWS
For integration testing with real AWS resources:

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_REGION=us-east-1

# Run with debug logging
make dev
```

## Code Quality Tools

We use several tools to maintain code quality:

- **golangci-lint**: Comprehensive Go linter
- **gosec**: Security vulnerability scanner
- **gofmt/goimports**: Code formatting
- **go vet**: Static analysis
- **pre-commit**: Git hooks for quality checks

## Documentation

- Keep README.md up to date
- Document new metrics in the metrics section
- Add code comments for complex logic
- Update environment variable documentation

## Release Process

Releases are handled by maintainers:

1. Version bump in appropriate files
2. Update CHANGELOG.md
3. Create GitHub release with binaries
4. Update Docker image tags

## Getting Help

- Open an issue for bugs or feature requests
- Use discussions for questions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.