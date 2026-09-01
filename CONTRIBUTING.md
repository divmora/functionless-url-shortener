# Contributing to Functionless URL Shortener

Thank you for your interest in contributing to **Functionless URL Shortener**! We welcome bug reports, documentation improvements, feature suggestions, and code contributions.

Please review this guide before submitting issues or pull requests.

---

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](https://github.com/divmora/.github/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

## Getting Started

### Prerequisites

Ensure you have the following tools installed on your development machine:
- **AWS CLI**: [aws.amazon.com/cli](https://aws.amazon.com/cli/) (configured with appropriate deployment credentials)
- **AWS SAM CLI v0.37.0+**: [docs.aws.amazon.com/serverless-application-model](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- **Make**: Standard build automation tool
- **cfn-lint** *(optional but recommended)*: AWS CloudFormation Linter

### Development Setup

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/<your-username>/functionless-url-shortener.git
   cd functionless-url-shortener
   ```

2. **Validate template and OpenAPI specifications:**
   ```bash
   make validate
   ```

---

## Development Workflow

### Available Make Targets

- `make validate` - Validates the SAM template (`template.yaml`) and OpenAPI specification (`api.yaml`)
- `make build` - Builds the SAM application (`sam build`)
- `make deploy` - Deploys the stack using existing `samconfig.toml` parameters
- `make deploy-guided` - Runs guided interactive deployment (`sam deploy -g`)
- `make clean` - Cleans temporary `.aws-sam` build directories and artifacts

---

## Conventional Commits

This project uses [Release Please](https://github.com/googleapis/release-please) to automate semantic versioning and release notes. All commit messages and Pull Request titles **must** adhere to the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format
```
<type>(<optional scope>): <description>
```

### Common Types
- `feat:` A new feature or endpoint (triggers minor version bump)
- `fix:` A bug fix or patch (triggers patch version bump)
- `docs:` Documentation-only changes
- `refactor:` Code or template refactoring without functional changes
- `test:` Adding or updating tests / validation
- `chore:` Tooling, CI/CD, dependency updates, or internal cleanup

---

## Submitting Pull Requests

1. Create a descriptive branch from `main`:
   ```bash
   git checkout -b feat/my-new-feature
   ```
2. Make your template or specification changes.
3. Validate templates locally:
   ```bash
   make validate
   ```
4. Commit your changes with a conventional commit message:
   ```bash
   git commit -m "feat(api): add custom header support"
   ```
5. Push to your fork and open a Pull Request against the `main` branch.
6. Ensure all CI checks pass. Maintainers will review your PR promptly.

---

## Licensing & Contributor Terms

By submitting a pull request, you agree that your contributions will be licensed under the project's [Business Source License 1.1 (BSL 1.1)](LICENSE), with the understanding that they will convert to the Apache License 2.0 under the standard Change Date schedule.
