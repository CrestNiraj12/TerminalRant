# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.2] - 2026-02-17

### Added

- `go install` support as primary installation method for Go developers
- Homebrew package manager support for macOS/Linux installation
- Scoop package manager support for Windows installation
- Comprehensive installation documentation in README

### Changed

- Restructured `.goreleaser.yaml` for improved clarity and maintainability
- Reorganized README installation section with multiple installation methods

### Fixed

- Removed outdated installer version notes

### Removed

- Unused `getSelectedRantID()` function from feed helpers
- Unused `feedTotalLines()` function from feed helpers
- Unused `firstMediaPreviewURL()` function from media module
- Unused `firstMediaOpenURL()` function from media module

## [v0.2.1] - 2026-02-17

### Changed

- Updated module path to `github.com/CrestNiraj12/terminalrant` for proper Go module resolution

### Refactored

- Split monolithic `tui/feed/model.go` (3364 lines) into focused, modular components
- Improved code organization with separate concerns:
  - State management
  - Command handling
  - Feed utilities
  - Media handling
  - Thread utilities
  - Update handlers

## [v0.2.0] - Previous

For previous changes, please refer to [GitHub releases](https://github.com/CrestNiraj12/terminalrant/releases).

---

## Guidelines for Future Releases

### Commit Message Format

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification with gitmojis:

- `✨ feat` - A new feature
- `🐛 fix` - A bug fix
- `📚 docs` - Documentation changes
- `💄 style` - Code style changes (formatting, etc.)
- `♻️ refactor` - Code refactoring
- `⚡ perf` - Performance improvements
- `✅ test` - Adding or updating tests
- `👷 build` - Build system or dependency changes
- `💚 ci` - CI/CD configuration changes
- `🔧 chore` - Maintenance tasks
- `🔥 remove` - Removing code or files
- `🚑 hotfix` - Urgent bug fixes

### Changelog Sections

Releases are automatically categorized into:

1. **Features** - New functionality added
2. **Bug Fixes** - Issues resolved
3. **Documentation** - Documentation improvements
4. **Performance** - Performance optimizations
5. **Refactoring** - Code refactoring
6. **Build & CI** - Build and CI/CD changes
7. **Chore & Maintenance** - Maintenance tasks

### Release Process

1. Commits are automatically collected and categorized by type
2. Release notes are generated with installation instructions
3. Changelog is updated automatically from git history
4. Supported platforms and package managers are documented


