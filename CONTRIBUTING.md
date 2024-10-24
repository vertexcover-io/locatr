# Contributing to Locatr

Thank you for your interest in contributing to Locatr! We appreciate your time and effort in improving the project. Please follow the guidelines below to ensure that your contributions are consistent and valuable.

## Getting Started

### 1. Fork the Repository
Start by forking the repository on GitHub. You can do this by clicking the “Fork” button at the top of the repository page.

### 2. Clone the Fork
Once the repository is forked, clone it to your local machine.

```bash
git clone https://github.com/YOUR_USERNAME/locatr.git
cd locatr
```

### 3. Install Dependencies
Make sure you have all the necessary tools and dependencies installed, including Go and pre-commit.
- Go: Install Go from the [official site](https://go.dev/doc/install).
- Pre-commit: Install pre-commit to ensure that the hooks run before each commit [link](https://pre-commit.com/#install).
```
pip install pre-commit
pre-commit install
```

### 4. Create a New Branch
Create a new branch for your feature or bugfix.
```
git checkout -b your-branch-name
```
---
## Development Guidelines
### Code Style
Please ensure that your code follows the Go language standards and style. We use the following pre-commit hooks to enforce code quality and style:

- **go-fmt**: Formats Go code.
- **go-imports**: Orders and groups imports properly.
- **golangci-lint**: Runs various linters to ensure code quality.

Install golangcli-lint from [this link](https://golangci-lint.run/)

The pre-commit hooks will automatically run before each commit. You can also run them manually at any time using:

```
pre-commit run --all-files
```

## Running Tests
Make sure that any changes you make are properly tested. You can run the test suite with:
```
go test ./...
```
If you add a new feature, ensure that it is covered by tests. If you fix a bug, add a test that reproduces the issue.

## Documentation
Please document your code appropriately, especially public functions and methods. If you are adding a new feature, update the relevant documentation (if applicable).

## Submitting a Pull Request
1. Push your branch to your fork:
```
git push origin your-branch-name
```
2. Open a Pull Request (PR) from your branch to the main branch in the original repository. In the PR description, explain the changes you have made and why they are necessary.
3. Ensure that your code passes the continuous integration (CI) checks.
4. Be responsive to feedback and make necessary updates based on code reviews.