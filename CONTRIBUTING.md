# Contributing to Event Trigger Platform

Thank you for your interest in contributing to the Event Trigger Platform! We welcome contributions from the community and are grateful for any help you can provide.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Code Review Guidelines](#code-review-guidelines)
- [Community](#community)

## Code of Conduct

This project adheres to a code of conduct that all contributors are expected to follow. Please be respectful, inclusive, and considerate in all interactions. We are committed to providing a welcoming and harassment-free experience for everyone.

## Getting Started

Before you begin contributing, please:

1. **Read the README**: Familiarize yourself with the project by reading the [README.md](README.md)
2. **Check existing issues**: Browse [open issues](../../issues) to see if someone is already working on what you have in mind
3. **Open a new issue**: If you're planning a significant change, open an issue first to discuss it with the maintainers

## Development Setup

### Prerequisites

- **Go** 1.22 or higher
- **Docker** and **Docker Compose**
- **Git**
- **Make** (optional, for convenience)

### Local Environment Setup

1. **Fork and clone the repository**:

   ```bash
   git clone https://github.com/YOUR_USERNAME/event-trigger-platform.git
   cd event-trigger-platform
   ```

2. **Start infrastructure services** (MySQL and Kafka):

   ```bash
   cd deploy
   docker-compose up -d mysql kafka
   cd ..
   ```

3. **Install Go dependencies**:

   ```bash
   go mod download
   ```

4. **Run database migrations**:

   ```bash
   # Using MySQL client
   mysql -h localhost -u appuser -p event_trigger < db/migrations/001_initial_schema.sql
   mysql -h localhost -u appuser -p event_trigger < db/migrations/002_add_indexes.sql
   mysql -h localhost -u appuser -p event_trigger < db/migrations/003_add_webhook_url.sql
   mysql -h localhost -u appuser -p event_trigger < db/migrations/004_setup_retention_events.sql
   # Default password: apppassword
   ```

5. **Set environment variables**:

   ```bash
   export DATABASE_URL="appuser:apppassword@tcp(localhost:3306)/event_trigger?parseTime=true"
   export KAFKA_BROKERS="localhost:9092"
   export LOG_LEVEL="debug"
   ```

6. **Run the API server**:

   ```bash
   go run ./cmd/api
   ```

7. **Run the scheduler** (in a separate terminal):

   ```bash
   go run ./cmd/scheduler
   ```

8. **Verify the setup**:

   ```bash
   curl http://localhost:8080/health
   ```

### Generate Swagger Documentation

After making changes to API endpoints:

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

## How to Contribute

### Types of Contributions

We welcome various types of contributions:

- **Bug Fixes**: Found a bug? Submit a fix!
- **New Features**: Have an idea? Discuss it in an issue first
- **Documentation**: Help improve our docs
- **Tests**: Increase test coverage
- **Code Refactoring**: Improve code quality
- **Performance Improvements**: Optimize existing code

### Contribution Workflow

1. **Fork the repository** on GitHub
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Make your changes** following our coding standards
4. **Write or update tests** for your changes
5. **Run tests** to ensure nothing breaks:
   ```bash
   go test ./...
   ```
6. **Commit your changes** with clear, descriptive messages:
   ```bash
   git commit -m "Add feature: description of what you added"
   ```
7. **Push to your fork**:
   ```bash
   git push origin feature/amazing-feature
   ```
8. **Open a Pull Request** against the `main` branch

## Coding Standards

### Go Best Practices

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code (most editors do this automatically)
- Run `golangci-lint` to catch common issues:
  ```bash
  golangci-lint run ./...
  ```
- Write idiomatic Go code that follows community conventions

### Code Style Guidelines

- **Naming Conventions**:
  - Use `camelCase` for unexported variables and functions
  - Use `PascalCase` for exported variables, functions, and types
  - Use descriptive names that clearly indicate purpose
  
- **Error Handling**:
  - Always check and handle errors explicitly
  - Wrap errors with context using `fmt.Errorf` with `%w` verb
  - Return errors rather than panicking (except in initialization)

- **Comments**:
  - Add comments for exported functions, types, and packages
  - Use complete sentences with proper punctuation
  - Explain *why* not *what* for complex logic

- **Package Organization**:
  - Keep packages focused and cohesive
  - Avoid circular dependencies
  - Place internal packages under `internal/`

### Project-Specific Guidelines

- **Configuration**: Use environment variables, not hardcoded values
- **Logging**: Use the structured logger from `internal/logging`
- **Database**: All queries should use prepared statements or the ORM patterns
- **API Responses**: Follow the existing response format patterns
- **Swagger Annotations**: Add swagger comments for all API endpoints

## Testing Guidelines

### Writing Tests

- **Test Coverage**: Aim for at least 70% code coverage for new code (this is a project guideline that balances quality with development velocity)
- **Test Files**: Place test files alongside the code they test (`*_test.go`)
- **Table-Driven Tests**: Use table-driven tests for multiple test cases
- **Test Names**: Use descriptive test names that explain what is being tested

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/triggers/...

# Run integration tests (when implemented)
go test -tags=integration ./...
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods in isolation
2. **Integration Tests**: Test interactions between components
3. **API Tests**: Test HTTP endpoints end-to-end
4. **Database Tests**: Test repository layer with real database

### Mocking

- Use interfaces to enable mocking of dependencies
- Consider using [testify/mock](https://github.com/stretchr/testify) for complex mocking needs
- Keep mocks simple and focused

## Pull Request Process

### Before Submitting

- [ ] Tests pass locally (`go test ./...`)
- [ ] Code is formatted (`gofmt -w .`)
- [ ] Linter passes (`golangci-lint run ./...`)
- [ ] Swagger docs are updated (if API changes)
- [ ] Documentation is updated (README, comments, etc.)
- [ ] Commits are atomic and have clear messages

### PR Description Template

When opening a PR, please include:

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## Testing
Describe the tests you ran and how to reproduce them

## Checklist
- [ ] My code follows the project's coding standards
- [ ] I have added tests that prove my fix/feature works
- [ ] I have updated the documentation accordingly
- [ ] All tests pass locally
```

### Review Process

1. **Automated Checks**: CI/CD will run tests and linters automatically
2. **Code Review**: At least one maintainer will review your code
3. **Feedback**: Address any feedback or requested changes
4. **Approval**: Once approved, a maintainer will merge your PR

### Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples**:
```
feat(triggers): add support for webhook retry configuration
fix(scheduler): correct timezone calculation for CRON triggers
docs(readme): update installation instructions
test(api): add integration tests for trigger endpoints
```

## Code Review Guidelines

### For Contributors

- **Be Responsive**: Respond to review comments promptly
- **Be Open**: Be open to feedback and suggestions
- **Ask Questions**: If feedback is unclear, ask for clarification
- **Iterate**: Make requested changes and push updates

### For Reviewers

- **Be Constructive**: Provide actionable, helpful feedback
- **Be Specific**: Point out exact lines and explain concerns
- **Be Timely**: Review PRs within a reasonable timeframe
- **Be Encouraging**: Acknowledge good work and improvements

### Review Checklist

- [ ] Code follows project standards and conventions
- [ ] Changes are well-tested with appropriate coverage
- [ ] Documentation is updated and accurate
- [ ] No unnecessary complexity or over-engineering
- [ ] Security considerations are addressed
- [ ] Performance implications are considered
- [ ] Error handling is appropriate

## Community

### Getting Help

- **Documentation**: Start with the [README.md](README.md)
- **Issues**: Search [existing issues](../../issues)
- **Discussions**: Use [GitHub Discussions](../../discussions) for questions

### Reporting Bugs

When reporting bugs, please include:

1. **Description**: Clear description of the issue
2. **Steps to Reproduce**: Detailed steps to reproduce the behavior
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Environment**:
   - OS and version
   - Go version
   - Docker version (if applicable)
   - Any relevant configuration
6. **Logs**: Relevant log output or error messages

### Suggesting Features

When suggesting features, please:

1. **Search First**: Check if the feature has already been suggested
2. **Describe the Problem**: Explain the use case and why it's valuable
3. **Propose a Solution**: Share your ideas on how it could be implemented
4. **Consider Alternatives**: Mention alternative approaches you've considered

### Communication Guidelines

- Be respectful and professional
- Stay on topic and be constructive
- Assume good intentions
- Welcome newcomers and help them get started
- Give credit where credit is due

## Recognition

We value all contributions and maintain a list of contributors. Significant contributions will be acknowledged in release notes.

Thank you for contributing to Event Trigger Platform! 🎉
