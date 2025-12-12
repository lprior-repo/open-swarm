# Scripts

Utility scripts for the open-swarm project.

## Pre-commit Hooks

### Quick Start

**Option 1: Simple shell script (recommended for most developers)**

```bash
# Install the hook
./scripts/install-hooks.sh

# Test it
git add .
git commit -m "Test commit"
```

**Option 2: Using pre-commit framework (recommended for teams)**

```bash
# Install pre-commit (one time)
pip install pre-commit
# or: brew install pre-commit
# or: conda install -c conda-forge pre-commit

# Install hooks
pre-commit install

# Test on all files
pre-commit run --all-files
```

### What Gets Checked

The pre-commit hooks run the following checks on staged Go files:

1. **gofmt** - Ensures code is properly formatted
2. **go vet** - Catches common Go mistakes
3. **golangci-lint** - Fast linting on changed files only
4. **go mod tidy** - Ensures dependencies are clean

### Performance

The hooks are optimized for speed:
- Only checks **staged files**, not entire codebase
- Uses **`--fast`** mode for golangci-lint
- Uses **`--new`** flag to check only changed code
- Skips checks if no Go files are staged
- Timeout after 60 seconds to prevent hanging

Typical run time: **2-5 seconds** for small changes.

### Bypassing Hooks

If you need to commit without running hooks (not recommended):

```bash
git commit --no-verify -m "Emergency fix"
```

### Troubleshooting

**Hook not running?**

```bash
# Check if hook is installed
ls -la .git/hooks/pre-commit

# Should show: .git/hooks/pre-commit -> ../../scripts/pre-commit.sh

# Reinstall
./scripts/install-hooks.sh
```

**golangci-lint not found?**

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Or use the script to skip linting if not installed
# (it will show a warning but won't fail)
```

**Hook is too slow?**

The hook should be fast (2-5s). If it's slower:
1. Check if golangci-lint is timing out (60s limit)
2. Reduce number of files in single commit
3. Temporarily skip with `--no-verify` if urgent

**go mod tidy failing?**

```bash
# Run manually to see the issue
go mod tidy

# Check what changed
git diff go.mod go.sum

# Stage the changes
git add go.mod go.sum
```

### Files

- **`pre-commit.sh`** - Shell script that can be symlinked to `.git/hooks/pre-commit`
- **`install-hooks.sh`** - Helper script to install hooks
- **`../.pre-commit-config.yaml`** - Configuration for the pre-commit framework

### Integration with Make

You can also run the checks manually:

```bash
# Format code
make fmt

# Run tests
make test

# Run tests with race detector
make test-race
```

### CI/CD

The same checks should run in CI/CD pipelines. See `.github/workflows/ci.yml` for the full CI configuration.

### Adding More Hooks

To add additional checks:

1. **Edit `scripts/pre-commit.sh`** for the shell script version
2. **Edit `.pre-commit-config.yaml`** for the pre-commit framework version

Example: Adding a test check (not recommended for pre-commit due to speed):

```bash
# In pre-commit.sh, add:
echo -n "5. Running tests... "
if ! go test -short ./... >/dev/null 2>&1; then
    echo -e "${RED}✗${NC}"
    FAILED=1
else
    echo -e "${GREEN}✓${NC}"
fi
```

### Best Practices

1. **Keep it fast** - Pre-commit hooks should complete in seconds
2. **Check only changed files** - Don't run checks on entire codebase
3. **Provide clear error messages** - Tell developers how to fix issues
4. **Make it easy to bypass** - Allow `--no-verify` for emergencies
5. **Document everything** - This README helps new contributors

### References

- [Git Hooks Documentation](https://git-scm.com/docs/githooks)
- [pre-commit framework](https://pre-commit.com/)
- [golangci-lint](https://golangci-lint.run/)
