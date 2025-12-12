---
description: Coordinate with other agents for task handoff or collaboration
agent: coordinator
---

Coordinate with another agent:

**Arguments expected:** `<agent-name> <subject> [message]`

1. **Verify Agent Exists**
   - Query Agent Mail for registered agents
   - Confirm target agent is registered
   - If not found, list available agents

2. **Check Contact Permissions**
   - Use Agent Mail to check if contact relationship exists
   - If not, request contact with reason: $ARGUMENTS

3. **Compose Message**
   - Subject: Provided subject
   - Body: Provided message or prompt for details
   - Include relevant context:
     - Current task ID (from Beads)
     - Files involved
     - Dependencies or blockers
   - Set importance level appropriately
   - Mark ack_required if coordination is critical

4. **Send via Agent Mail**
   - Use thread ID matching Beads task if applicable
   - Send message through Agent Mail MCP
   - Confirm delivery

5. **Update Beads**
   - If this creates a dependency, add to Beads
   - Example: `bd dep add <this-task> <their-task> --type blocks`
   - Add note about coordination

6. **Report Outcome**
   - Confirm message sent
   - Show message ID and thread ID
   - Provide expected response timeframe
   - Suggest next actions
