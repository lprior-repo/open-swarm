---
description: Complete session start protocol for multi-agent coordination
---

Execute the session start protocol:

1. **Register with Agent Mail**
   - Use absolute project path as project key: `!pwd`
   - Program: opencode
   - Model: anthropic/claude-sonnet-4-5
   - Auto-generate agent name (adjective+noun format)
   - Set task description based on current work focus

2. **Fetch Agent Mail Inbox**
   - Retrieve recent messages (last 20)
   - Filter for urgent messages
   - Identify messages requiring acknowledgment
   - Display in formatted table

3. **Check Beads for Ready Work**
   - Run: `!bd ready --json`
   - Parse and display unblocked tasks
   - Show task IDs, titles, priorities, and tags
   - Recommend highest priority task

4. **Check Active File Reservations**
   - Query Agent Mail for active reservations
   - Show which agents are working on which files
   - Identify any potential conflicts with planned work

5. **Check Active Agents**
   - List all registered agents in this project
   - Show their current task descriptions
   - Display last activity timestamps

6. **Provide Session Summary**

Display a summary including:
- Your agent name and identity
- Unread messages count
- Urgent messages requiring attention
- Ready tasks count and recommended task
- Active agents count
- Any file reservation conflicts to be aware of
- Suggested next steps

Format as a clear, structured report.
