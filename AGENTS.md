# Open Swarm - Multi-Agent Coordination Framework

## ⚠️ CRITICAL RULES ⚠️

### 🔴 RULE #1: BEADS IS MANDATORY
**EVERY code change requires a Beads task. NO EXCEPTIONS.**
- Code change → Beads task first
- Bug fix → Beads task first  
- Feature → Beads task first
- Refactor → Beads task first
- Tests → Beads task first

No Beads task ID (e.g., `open-swarm-xyz`)? **DO NOT** make changes.

### 🔴 RULE #2: SERENA IS THE ONLY WAY TO EDIT CODE
**ALL Go code editing uses Serena's semantic tools.**
- ✅ USE: `serena_find_symbol`, `serena_replace_symbol_body`, `serena_insert_after_symbol`, `serena_rename_symbol`
- ❌ NEVER: Read + Edit, bash `sed`/`awk`

Exception: Non-code files (`.md`, `.json`, `.yaml`) use Edit tool.

### 🔴 RULE #3: NEVER CREATE MARKDOWN FILES
**DO NOT create docs unless explicitly requested.**
- No README.md, CHANGELOG.md, or .md files
- No proactive documentation
- User will ask if needed

### ✅ Workflow
1. Get/Create Beads task → `bd create` or `bd ready --json`
2. Start task → `bd update task-id --status in_progress`
3. Navigate with Serena → `serena_find_symbol`, `serena_find_referencing_symbols`
4. Edit with Serena → `serena_replace_symbol_body` or `serena_insert_after_symbol`
5. Complete → `bd close task-id --reason "description"`

---

## Stack

- **Agent Mail MCP** - Git-backed messaging, file reservations
- **Beads MCP** - Git-backed issue tracking (CRITICAL) 
- **Serena MCP** - LSP-powered semantic navigation (MANDATORY)

All tools accessed via MCP servers, configured in `opencode.json`.

## Prerequisites

- Go 1.25+, Agent Mail (`am`), Beads MCP (`beads-mcp`), Serena, OpenCode

## Setup

```bash
go mod download
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
bd init
am  # separate terminal
opencode
```

## Session Start

```bash
bd ready --json
opencode && /sync
bd update bd-xxxx --status in_progress
/reserve <pattern>
```

## Beads (via MCP)

Beads accessed through OpenCode MCP tools (`beads_*`):

```bash
# Check ready tasks
beads_ready

# Update task status  
beads_status taskId="bd-xxxx" status="in_progress"

# Create new task
beads_create title="Issue" parent="bd-xxxx"

# Close task
beads_close taskId="bd-xxxx" reason="Description"
```

**Note:** `bd` CLI commands also work directly for quick operations.

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
beads_close taskId="bd-xxxx" reason="Description"
/release
git add .beads/issues.jsonl && git commit -m "Update tasks" && git push
```

## Troubleshooting

```bash
curl http://localhost:8765/health  # Agent Mail
am                                  # Restart Agent Mail
bd sync                            # Beads sync
bd list --json                     # Check tasks
```

## Quick Reference

| Task | Command |
|------|---------|
| Check work | `bd ready --json` |
| Start task | `bd update bd-xxxx --status in_progress` |
| Reserve | `/reserve pattern` |
| Find | `serena_find_symbol: "Name"` |
| Edit | `serena_replace_symbol_body` |
| Test | `go test ./...` |
| Complete | `bd close bd-xxxx` |
| Release | `/release` |

**Critical:**
- ✅ ALWAYS: Beads, Serena, reserve files
- ❌ NEVER: Edit Go without Serena, no Beads task, create .md files

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Agent Mail](https://github.com/Dicklesworthstone/mcp_agent_mail)
- [Beads](https://github.com/steveyegge/beads)
- [Serena](https://oraios.github.io/serena/)
- [OpenCode](https://opencode.ai/docs/)
