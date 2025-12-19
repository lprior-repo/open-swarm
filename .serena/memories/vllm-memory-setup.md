# vLLM Memory Server Setup - Production Configuration (VERIFIED)

## Full System Stack (FULLY OPERATIONAL ✅)
- **LLM Server:** vLLM at http://localhost:8001/v1 (OpenAI-compatible API)
- **LLM Model:** Qwen/Qwen3-30B-A3B-GPTQ-Int4 (30B parameters, GPTQ quantized)
- **Embeddings:** sentence-transformers/all-MiniLM-L6-v2 (384-dim vectors)
- **Vector Store:** PostgreSQL 18.1 at localhost:5432 (database: mem0)
- **MCP Transport:** stdio via /home/lewis/src/mcp-mem0/src/main.py

## Authentication Setup (VERIFIED WORKING)
**Generated Dummy API Key:** `sk-RIBRDSXhDicKkVTx6YCAlBdmyx81b6OwKGOCzcDqmWCOcnVR4y5PjYYACE7Zu_1V`

vLLM with OpenAI-compatible API requires `LLM_API_KEY` even for local instances.
- This key is set in both `/home/lewis/src/mcp-mem0/.env` and `~/.claude/settings.json`
- Prevents "api_key client option must be set" error
- mem0 client initializes successfully with this key

## Configuration Files (Source of Truth)
1. **~/.claude/settings.json** - Claude Code MCP configuration
   - Sets env vars for mem0 server at startup
   - `LLM_API_KEY`: Generated dummy key (see above)
   - `VECTOR_STORE_PROVIDER`: "supabase" (PostgreSQL backend)
   
2. **/home/lewis/src/mcp-mem0/.env** - mem0 startup config
   - Loads during `get_mem0_client()` initialization
   - Must match settings.json for consistency
   - `LLM_API_KEY`: Same generated key as above

## Verified Status (2025-12-19)
- vLLM: ✅ Running, Qwen model loaded and responding
- PostgreSQL: ✅ Running in Docker, mem0 database created and accessible
- mem0 Client: ✅ Initializes successfully with vLLM + PostgreSQL + dummy API key
- Embeddings: Ready (auto-downloads from HuggingFace on first use)

## Connection Details (Docker-based PostgreSQL)
- PostgreSQL running as Docker container
- Accessible via TCP: localhost:5432 (not socket /var/run/postgresql/)
- Default credentials: postgres:postgres
- mem0 database created via: `CREATE DATABASE mem0`

## Verification Commands
```bash
# vLLM health
curl http://localhost:8001/v1/models

# PostgreSQL health
PGPASSWORD=postgres psql -h localhost -U postgres -d mem0 -c "SELECT 1"

# mem0 client initialization (test with generated key)
cd /home/lewis/src/mcp-mem0 && python -c "from src.utils import get_mem0_client; print(get_mem0_client())"
```

## Key Migration from Ollama Setup
- **Old:** Ollama local models + Qdrant vector store + socket-based access
- **New:** vLLM + PostgreSQL via TCP + OpenAI-compatible API + dummy key for local use
- **Benefit:** PostgreSQL provides relational + vector storage, vLLM easier to integrate
