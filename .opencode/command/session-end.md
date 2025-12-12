---
description: Complete session end protocol to ensure clean handoff
---

Execute the session end protocol:

1. **Update Beads Task Status**
   - If task was in progress, update status to: done, in_progress, or blocked
   - If completed, close with `bd close <id> --reason "<completion description>"`
   - If incomplete, ensure status accurately reflects current state

2. **File Discovered Issues**
   - Review work done this session
   - File any new issues discovered as Beads tasks
   - Link to parent task if applicable
   - Add appropriate tags and priorities

3. **Release File Reservations**
   - Release ALL active file reservations held by this agent
   - Verify release was successful
   - Report which files were released

4. **Send Status Updates**
   - If work is complete and unblocks other agents, send Agent Mail message
   - Include task ID in subject
   - Provide clear summary of what was accomplished
   - Mention any blockers or dependencies

5. **Sync Beads to Git**
   - Run: `!git status .beads/`
   - If changes exist: `!git add .beads/issues.jsonl`
   - Create commit: `!git commit -m "Update task tracking"`
   - Note: DO NOT push automatically (requires user approval)

6. **Generate Session Summary**

Create a summary report including:
- Task(s) worked on
- Current status of each task
- Files modified
- Tests written/updated
- Issues filed
- Messages sent
- File reservations released
- Commits created (not pushed)
- Recommended next steps for next agent or session

Format as a clear, structured report.
