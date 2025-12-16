# Open Swarm - Multi-Agent Coordination Framework

## ⚠️ CRITICAL RULES ⚠️

### 🔴 RULE #1: SERENA IS THE ONLY WAY TO EDIT CODE
**ALL Go code editing uses Serena's semantic tools.**
- ✅ USE: `serena_find_symbol`, `serena_replace_symbol_body`, `serena_insert_after_symbol`, `serena_rename_symbol`
- ❌ NEVER: Read + Edit, bash `sed`/`awk`

Exception: Non-code files (`.md`, `.json`, `.yaml`) use Edit tool.

### 🔴 RULE #2: NEVER CREATE MARKDOWN FILES
**DO NOT create docs unless explicitly requested.**
- No README.md, CHANGELOG.md, or .md files
- No proactive documentation
- User will ask if needed

### 🔴 RULE #3: TDD IS MANDATORY
**ALL Go code changes follow Test-Driven Development.**
- Test file must exist BEFORE implementation
- Test must fail first (RED)
- Minimal implementation makes test pass (GREEN)
- Use testify for assertions
- Tests must be atomic, small, deterministic

Validate with: `validateTDD filePath="internal/api/handler.go"`

### ✅ Workflow
1. Navigate with Serena → `serena_find_symbol`, `serena_find_referencing_symbols`
2. Edit with Serena → `serena_replace_symbol_body` or `serena_insert_after_symbol`
3. Test → `go test ./...`
4. Commit changes

---

## Stack

- **Agent Mail MCP** - Git-backed messaging, file reservations
- **Serena MCP** - LSP-powered semantic navigation (MANDATORY)

All tools accessed via MCP servers, configured in `opencode.json`.

## Prerequisites

- Go 1.25+, Agent Mail (`am`), Serena, OpenCode

## Setup

```bash
go mod download
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
am  # separate terminal
opencode
```

## Session Start

```bash
opencode && /sync
/reserve <pattern>
```

## Agent Mail

```bash
/reserve internal/api/**/*.go    # Reserve before editing
/release                         # Release when done
```

Send message:
```
To: <AgentName>
Subject: [bd-xxxx] Task complete
Thread: bd-xxxx
Body: Description
```

## Serena (MANDATORY)

**Navigate:**
```
Find symbol: "FunctionName"
Find references: "FunctionName"
Get symbols overview: "path/to/file.go"
```

**Edit:**
```
Replace symbol body: "FunctionName" with implementation
Insert after symbol: "StructName.Method" with new method
Rename symbol: "OldName" to "NewName"
```

**Never edit Go files without Serena.**

## Standards

- Follow [Effective Go](https://go.dev/doc/effective_go)
- `gofmt`, `golangci-lint`
- Handler → Service → Repository
- Interface-driven DI
- 80%+ test coverage
- Table-driven tests

## Commands

```bash
go build -o bin/open-swarm ./cmd/open-swarm
go test ./...
make test-coverage
make lint
make lint-fix
make ci
```

## Session End

```bash
/release
git add . && git commit -m "Description" && git push
```

## Troubleshooting

```bash
curl http://localhost:8765/health  # Agent Mail
am                                  # Restart Agent Mail
```

## Quick Reference

| Task | Command |
|------|---------|
| Sync | `/sync` |
| Reserve | `/reserve pattern` |
| Find | `serena_find_symbol: "Name"` |
| Edit | `serena_replace_symbol_body` |
| Test | `go test ./...` |
| Release | `/release` |

**Critical:**
- ✅ ALWAYS: Serena for Go code editing, reserve files
- ❌ NEVER: Edit Go without Serena, create .md files without request

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Agent Mail](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [Serena](https://oraios.github.io/serena/)
- [OpenCode](https://opencode.ai/docs/)
