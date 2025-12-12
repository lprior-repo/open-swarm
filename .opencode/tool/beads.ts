import { tool } from "@opencode-ai/plugin"
import { $ } from "bun"

export const ready = tool({
  description: "Get ready (unblocked) tasks from Beads issue tracker",
  args: {
    format: tool.schema.string().optional().default("json").describe("Output format: json or text")
  },
  async execute(args) {
    const result = await $`bd ready --json`.text()

    if (args.format === "json") {
      return result
    }

    // Parse and format for human reading
    const tasks = JSON.parse(result)
    if (tasks.length === 0) {
      return "No ready tasks found. All tasks are either blocked or completed."
    }

    let output = `Ready Tasks (${tasks.length}):\n\n`
    for (const task of tasks) {
      output += `${task.id}: ${task.title}\n`
      output += `  Status: ${task.status} | Priority: ${task.priority || 'normal'}\n`
      if (task.tags && task.tags.length > 0) {
        output += `  Tags: ${task.tags.join(', ')}\n`
      }
      output += `\n`
    }

    return output
  }
})

export const status = tool({
  description: "Update status of a Beads task",
  args: {
    taskId: tool.schema.string().describe("Task ID (e.g., bd-a1b2)"),
    status: tool.schema.string().describe("New status: ready, in_progress, blocked, done")
  },
  async execute(args) {
    await $`bd update ${args.taskId} --status ${args.status}`
    return `Task ${args.taskId} updated to status: ${args.status}`
  }
})

export const close = tool({
  description: "Close a Beads task with a completion reason",
  args: {
    taskId: tool.schema.string().describe("Task ID to close"),
    reason: tool.schema.string().describe("Reason for completion")
  },
  async execute(args) {
    await $`bd close ${args.taskId} --reason ${args.reason}`
    return `Task ${args.taskId} closed: ${args.reason}`
  }
})

export const create = tool({
  description: "Create a new Beads task",
  args: {
    title: tool.schema.string().describe("Task title"),
    type: tool.schema.string().optional().describe("Task type: feature, bug, chore, doc"),
    priority: tool.schema.string().optional().describe("Priority: low, normal, high, urgent"),
    parent: tool.schema.string().optional().describe("Parent task ID")
  },
  async execute(args) {
    let cmd = `bd create "${args.title}"`

    if (args.type) {
      cmd += ` -t ${args.type}`
    }
    if (args.priority) {
      cmd += ` -p ${args.priority}`
    }
    if (args.parent) {
      cmd += ` --parent ${args.parent}`
    }

    const result = await $`${cmd}`.text()
    return result
  }
})

export const list = tool({
  description: "List Beads tasks with optional filters",
  args: {
    status: tool.schema.string().optional().describe("Filter by status"),
    tag: tool.schema.string().optional().describe("Filter by tag"),
    format: tool.schema.string().optional().default("json")
  },
  async execute(args) {
    let cmd = "bd list --json"

    if (args.status) {
      cmd += ` --status ${args.status}`
    }
    if (args.tag) {
      cmd += ` --tag ${args.tag}`
    }

    const result = await $`${cmd}`.text()
    return result
  }
})

export const addDependency = tool({
  description: "Add a dependency between two Beads tasks",
  args: {
    childId: tool.schema.string().describe("Child task ID"),
    parentId: tool.schema.string().describe("Parent task ID"),
    type: tool.schema.string().optional().default("blocks").describe("Dependency type: blocks, related, discovered-from")
  },
  async execute(args) {
    await $`bd dep add ${args.childId} ${args.parentId} --type ${args.type}`
    return `Added ${args.type} dependency: ${args.childId} → ${args.parentId}`
  }
})
