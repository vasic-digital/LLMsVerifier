# Contributing to LLM Verifier

Welcome! We're thrilled that you're interested in contributing to LLM Verifier. This document outlines the process for contributing to our project.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)
- [Security](#security)
- [Community](#community)

## 🤝 Code of Conduct

This project adheres to a code of conduct to ensure a welcoming environment for all contributors. By participating, you agree to:

- Be respectful and inclusive
- Focus on constructive feedback
- Accept responsibility for mistakes
- Show empathy towards other community members
- Help create a positive environment

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Git
- Make (optional, for using Makefiles)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/llm-verifier.git
   cd llm-verifier
   ```

3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/original-owner/llm-verifier.git
   ```

4. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## 🛠️ Development Setup

### Local Development Environment

1. **Install Dependencies**:
   ```bash
   # Install Go dependencies
   go mod download

   # Install development tools
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

2. **Environment Setup**:
   ```bash
   # Copy environment configuration
   cp config.yaml.example config.yaml

   # Edit configuration as needed
   nano config.yaml
   ```

3. **Run Tests**:
   ```bash
   # Run all tests
   make test

   # Run tests with coverage
   make test-coverage

   # Run integration tests
   make test-integration
   ```

4. **Start Development Server**:
   ```bash
   # Start the API server
   make run

   # Or run directly
   go run cmd/main.go server
   ```

### Docker Development

```bash
# Build development container
docker build -t llm-verifier-dev -f Dockerfile.dev .

# Run development container
docker run -it --rm \
  -v $(pwd):/app \
  -p 8080:8080 \
  llm-verifier-dev

# Inside container, run tests
go test ./...
```

## 📝 Contributing Guidelines

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use `goimports` for import organization
- Follow Go naming conventions
- Write clear, concise commit messages
- Add comments for exported functions and types

### Commit Messages

Use conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Testing changes
- `chore`: Maintenance tasks

Examples:
```
feat(auth): add LDAP authentication support
fix(api): resolve memory leak in request handler
docs(readme): update installation instructions
```

### Branch Naming

- `feature/description`: New features
- `bugfix/issue-number-description`: Bug fixes
- `hotfix/description`: Critical fixes
- `docs/description`: Documentation updates

## 🔄 Pull Request Process

### Before Submitting

1. **Update your branch**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run quality checks**:
   ```bash
   make lint        # Run linter
   make test        # Run tests
   make security    # Run security checks
   ```

3. **Update documentation** if needed

4. **Add tests** for new functionality

### Creating a Pull Request

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create PR on GitHub**:
   - Use a clear, descriptive title
   - Fill out the PR template
   - Reference any related issues
   - Add screenshots for UI changes

3. **PR Template**:
   ```markdown
   ## Description
   Brief description of changes

   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## Testing
   - [ ] Unit tests added/updated
   - [ ] Integration tests added/updated
   - [ ] Manual testing completed

   ## Checklist
   - [ ] Code follows style guidelines
   - [ ] Documentation updated
   - [ ] Tests pass
   - [ ] Security checks pass
   ```

### Review Process

1. **Automated Checks**: CI/CD will run tests, linting, and security scans
2. **Code Review**: At least one maintainer review required
3. **Approval**: Maintainers will approve and merge
4. **Merge**: Use "Squash and merge" for clean history

## 🧪 Testing

### Unit Tests

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/auth

# Run with race detection
go test -race ./...
```

### Integration Tests

```bash
# Run integration tests
make test-integration

# Run with specific environment
TEST_ENV=staging make test-integration
```

### Test Coverage

- Maintain >80% code coverage
- Include tests for error conditions
- Test edge cases and boundary conditions

### Writing Tests

```go
func TestUserAuthentication(t *testing.T) {
    // Arrange
    auth := NewAuthManager("test-secret")
    user := &User{Username: "testuser"}

    // Act
    token, err := auth.Authenticate(user)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

## 📚 Documentation

### API Documentation

- Use Go doc comments for all exported functions
- Update OpenAPI/Swagger specs for API changes
- Include examples in documentation

### User Documentation

- Update README for new features
- Add tutorials and guides
- Update configuration examples

### Internal Documentation

- Document complex algorithms
- Explain design decisions
- Include architecture diagrams

## 🔒 Security

### Security Checklist

- [ ] No hardcoded secrets or credentials
- [ ] Input validation and sanitization
- [ ] SQL injection prevention
- [ ] XSS protection for web interfaces
- [ ] CSRF protection for forms
- [ ] Secure session management
- [ ] Proper error handling (no sensitive data in errors)

### Reporting Security Issues

- **DO NOT** report security issues in public GitHub issues
- Email security@llm-verifier.com with details
- Allow 48 hours for initial response
- Responsible disclosure policy applies

### Security Testing

```bash
# Run security scans
make security-scan

# Check for vulnerabilities
make vuln-check

# Run SAST (Static Application Security Testing)
make sast
```

## 🌍 Community

### Communication

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Discord/Slack**: Real-time chat (link in README)
- **Mailing List**: Announcements and newsletters

### Recognition

Contributors are recognized through:
- GitHub contributor statistics
- Mention in release notes
- Contributor badges
- Hall of Fame in documentation

### Governance

- **Maintainers**: Core team managing the project
- **Contributors**: Community members with write access
- **Community**: All participants in discussions and issues

## 🎯 Development Workflow

### Daily Development

1. Pull latest changes: `git pull upstream main`
2. Create feature branch: `git checkout -b feature/name`
3. Make changes with tests
4. Run quality checks: `make check`
5. Commit changes: `git commit -m "feat: description"`
6. Push branch: `git push origin feature/name`
7. Create PR

### Release Process

1. **Feature Freeze**: No new features for 1 week
2. **Testing Phase**: Comprehensive testing and bug fixes
3. **Documentation**: Update all docs and changelogs
4. **Security Review**: Final security assessment
5. **Release**: Tag version and create GitHub release
6. **Deployment**: Automated deployment to production

## 📋 Issue Templates

### Bug Report Template

```markdown
## Bug Report

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Environment:**
 - OS: [e.g. Ubuntu 20.04]
 - Version: [e.g. v1.2.3]
 - Browser: [e.g. Chrome 91]

**Additional context**
Add any other context about the problem here.
```

### Feature Request Template

```markdown
## Feature Request

**Is your feature request related to a problem? Please describe.**
A clear and concise description of what the problem is.

**Describe the solution you'd like**
A clear and concise description of what you want to happen.

**Describe alternatives you've considered**
A clear and concise description of any alternative solutions or features you've considered.

**Additional context**
Add any other context or screenshots about the feature request here.
```

## 🙏 Acknowledgments

Thank you for contributing to LLM Verifier! Your contributions help make AI more accessible, reliable, and secure for everyone.

---

*This contributing guide is based on industry best practices and is actively maintained. Last updated: December 2025*