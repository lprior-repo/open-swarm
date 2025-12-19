## MCP Servers

**IMPORTANT**: Always leverage MCP servers as much as possible for all tasks.

### Unified Knowledge System (Graphiti + Beads + Serena)

The project uses a **three-tier knowledge synchronization system** to maintain consistent, queryable context:

1. **Graphiti (falkordb)** - Semantic knowledge graph for architecture and patterns
   - **Group ID**: `open-swarm-codebase`
   - **Purpose**: Query-optimized semantic search over architecture, patterns, dependencies
   - **Format**: JSON episodes with structured entities and relationships
   - **Auto-index**: Major architecture changes trigger episode updates

2. **Beads (bd)** - Work tracking and issue management
   - **Purpose**: Task tracking with dependencies, status, and acceptance criteria
   - **Integration**: Each task links to relevant Graphiti nodes for context
   - **Sync point**: Task creation/completion triggers context updates

3. **Serena Memory** - Project-specific analysis and reference guides
   - **Purpose**: Narrative documentation and implementation guides
   - **Key files**:
     - `graphiti-codebase-indexing-setup.md`: Graph structure and query guide
     - `open-swarm-project-analysis.md`: Architecture and package structure
     - Other domain-specific memories created during work
   - **Context management**: Read only relevant memories to avoid context bloat

### Serena (Semantic Code Analysis + Memory)

- **Auto-activate on session start**: Immediately activate the serena project with `mcp__plugin_serena_serena__activate_project` using project name "open-swarm"
- **Prefer serena tools** for all code exploration, analysis, and editing tasks:
  - Use `find_symbol` instead of grep for finding functions/classes/methods
  - Use `get_symbols_overview` to understand file structure before reading
  - Use `search_for_pattern` for flexible pattern matching in code
  - Use `replace_symbol_body` for precise code modifications
  - Use `find_referencing_symbols` to understand code dependencies
  - **Session startup**: Read `graphiti-codebase-indexing-setup.md` memory for query guide
  - Read other memories only when relevant to current task
- **Memory System**: Use Serena's `write_memory` and `read_memory` tools for project-specific memories (local, no external auth required)
- **Symbolic editing first**: Always prefer symbol-based tools over file-based editing when working with complete functions/classes
- **Resource-efficient**: Use targeted symbol queries instead of reading entire files when possible

### Knowledge Synchronization Protocol

#### Initialization (Session Start)
1. Activate Serena project: `activate_project("open-swarm")`
2. Set Beads context: `set_context("/home/lewis/src/open-swarm")`
3. Check ready tasks: `ready(limit=5)`
4. Read `graphiti-codebase-indexing-setup.md` for graph structure (only file to read auto)
5. Query Graphiti only when architecture questions arise (don't read preemptively)

#### During Task Execution
- **New patterns discovered**: Update relevant Graphiti episode via `add_memory`
- **New packages/components**: Record in Graphiti JSON structure
- **Gate implementations**: Update `Agent System and Anti-Cheating Gates` episode
- **Workflow changes**: Update `Temporal Workflow Engine` episode
- Keep Serena memories focused (max 2-3 open per task)

#### Git Hook: Pre-Commit Synchronization
File: `.git/hooks/pre-commit` (auto-created by initialization)
```bash
#!/bin/bash
# Sync Graphiti, Beads, and Serena before commit
if [ -f .beads/issues.jsonl ]; then
  bd sync  # Sync Beads state
fi
# Memories auto-sync via Serena file creation
# Graphiti episodes auto-sync via MCP
git add .beads/issues.jsonl .serena/memories/*.md
```

#### Git Hook: Post-Merge Synchronization
File: `.git/hooks/post-merge`
```bash
#!/bin/bash
# Refresh context after merge
bd validate  # Check Beads integrity
# Graphiti re-indexes during next query
echo "✓ Knowledge system ready"
```

#### Cleanup (Session End)
Before final commit:
1. `bd sync` - Ensure Beads state saved
2. `git add .beads/issues.jsonl .serena/memories/`
3. Commit with proper message
4. `git push`

### Context Window Management

**Avoid reading unnecessary memories**:
- Don't preemptively read all memories
- Read `graphiti-codebase-indexing-setup.md` once per session
- Read task-specific memories only when working on that task
- Use Graphiti semantic search instead of grepping memory files

**Smart context loading**:
```
Task relevance → Memory lookup → Graphiti query (if needed) → Proceed
```

**Memory lifecycle**:
- **Short-lived**: Task-specific implementation notes (delete after task close)
- **Medium-lived**: Feature-area guides (keep while working on feature)
- **Long-lived**: Architecture, patterns, synchronization guides (keep indefinitely)

### Other MCP Servers

- **mcp-agent-mail**: Use for agent coordination, messaging, and project context
- **playwright**: Use for browser automation tasks
- **Graphiti (local-graph)**: Use for semantic search and knowledge graph queries
- Prefer MCP server tools over equivalent bash commands or manual operations

Default to using Bun instead of Node.js.

- Use `bun <file>` instead of `node <file>` or `ts-node <file>`
- Use `bun test` instead of `jest` or `vitest`
- Use `bun build <file.html|file.ts|file.css>` instead of `webpack` or `esbuild`
- Use `bun install` instead of `npm install` or `yarn install` or `pnpm install`
- Use `bun run <script>` instead of `npm run <script>` or `yarn run <script>` or `pnpm run <script>`
- Bun automatically loads .env, so don't use dotenv.

## APIs

- `Bun.serve()` supports WebSockets, HTTPS, and routes. Don't use `express`.
- `bun:sqlite` for SQLite. Don't use `better-sqlite3`.
- `Bun.redis` for Redis. Don't use `ioredis`.
- `Bun.sql` for Postgres. Don't use `pg` or `postgres.js`.
- `WebSocket` is built-in. Don't use `ws`.
- Prefer `Bun.file` over `node:fs`'s readFile/writeFile
- Bun.$`ls` instead of execa.

## Testing

Use `bun test` to run tests.

```ts#index.test.ts
import { test, expect } from "bun:test";

test("hello world", () => {
  expect(1).toBe(1);
});
```

## Frontend

Use HTML imports with `Bun.serve()`. Don't use `vite`. HTML imports fully support React, CSS, Tailwind.

Server:

```ts#index.ts
import index from "./index.html"

Bun.serve({
  routes: {
    "/": index,
    "/api/users/:id": {
      GET: (req) => {
        return new Response(JSON.stringify({ id: req.params.id }));
      },
    },
  },
  // optional websocket support
  websocket: {
    open: (ws) => {
      ws.send("Hello, world!");
    },
    message: (ws, message) => {
      ws.send(message);
    },
    close: (ws) => {
      // handle close
    }
  },
  development: {
    hmr: true,
    console: true,
  }
})
```

HTML files can import .tsx, .jsx or .js files directly and Bun's bundler will transpile & bundle automatically. `<link>` tags can point to stylesheets and Bun's CSS bundler will bundle.

```html#index.html
<html>
  <body>
    <h1>Hello, world!</h1>
    <script type="module" src="./frontend.tsx"></script>
  </body>
</html>
```

With the following `frontend.tsx`:

```tsx#frontend.tsx
import React from "react";

// import .css files directly and it works
import './index.css';

import { createRoot } from "react-dom/client";

const root = createRoot(document.body);

export default function Frontend() {
  return <h1>Hello, world!</h1>;
}

root.render(<Frontend />);
```

Then, run index.ts

```sh
bun --hot ./index.ts
```

For more information, read the Bun API docs in `node_modules/bun-types/docs/**.md`.
