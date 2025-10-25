# Contributing to Butane Operator

Thank you for your interest in contributing to the Butane Operator! We welcome contributions from the community.

## Getting Started

### Prerequisites

- Go version v1.22.0+
- Docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/butane-operator.git
   cd butane-operator
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```

## Development Workflow

### Making Changes

1. Create a new branch for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards below

3. Test your changes:
   ```bash
   make test
   ```

4. Run linting:
   ```bash
   make lint
   ```

5. Build and test locally:
   ```bash
   make build
   make docker-build
   ```

### Coding Standards

- Follow Go best practices and conventions
- Use `gofmt` to format your code
- Run `golangci-lint` and fix any issues
- Add tests for new functionality
- Update documentation as needed

### Commit Messages

- Use clear and descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Keep the first line under 50 characters
- Reference issues and pull requests where applicable

Example:
```
Add support for custom storage configurations

- Extend ButaneConfig CRD to support additional storage options
- Update controller logic to handle new storage types
- Add comprehensive tests for storage configuration

Fixes #123
```

## Testing

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests (requires cluster access)
make test-integration

# Run end-to-end tests
make test-e2e
```

### Writing Tests

- Write unit tests for all new functions and methods
- Include integration tests for controller logic
- Use table-driven tests where appropriate
- Mock external dependencies

## Submitting Changes

1. Push your changes to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a pull request from your fork to the main repository

3. Ensure your PR:
   - Has a clear title and description
   - References any related issues
   - Includes tests for new functionality
   - Passes all CI checks
   - Updates documentation if needed

## Code Review Process

1. All submissions require review from maintainers
2. Reviews will focus on:
   - Code quality and style
   - Test coverage
   - Documentation completeness
   - Security considerations
3. Address review feedback promptly
4. Maintainers will merge approved PRs

## Issue Reporting

### Bug Reports

When reporting bugs, please include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Kubernetes version, etc.)
- Relevant logs or error messages

### Feature Requests

For feature requests, please provide:
- Clear description of the proposed feature
- Use case and motivation
- Possible implementation approach
- Any breaking changes involved

## Security

Please report security vulnerabilities responsibly by following our [Security Policy](SECURITY.md).

## Community

- Be respectful and inclusive
- Follow our [Code of Conduct](CODE_OF_CONDUCT.md)
- Help others in discussions and issues
- Share knowledge and best practices

## License

By contributing to this project, you agree that your contributions will be licensed under the LGPL 3.0 License.

## Questions?

If you have questions about contributing, please:
- Check existing issues and documentation
- Create a new issue with the "question" label
- Reach out to maintainers in discussions

Thank you for contributing to Butane Operator!