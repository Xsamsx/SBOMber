# Contributing to SBOMber

Thanks for contributing.

## Development Setup

1. Install Go `1.26` or newer.
2. Clone the repository.
3. Run `make tidy`.
4. Run `make ci` before opening a pull request.

## Project Principles

- Keep the CLI predictable and automation-friendly.
- Prefer small, testable packages over tightly coupled logic.
- Add tests for behavior changes.
- Keep stack-specific parsing isolated so new ecosystems can be added cleanly.

## Pull Requests

- Open focused pull requests with a clear scope.
- Update docs when behavior changes.
- Include tests for new parsing or detection logic.
- Avoid unrelated refactors in feature PRs.

## Commit Style

Commits do not need to follow a strict convention, but they should describe the change clearly and specifically.
