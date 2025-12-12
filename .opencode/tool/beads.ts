import { tool } from "@opencode-ai/plugin"
import { $ } from "bun"

export const ready = tool({
  description: "Get ready (unblocked) tasks from Beads issue tracker",
  args: {
    format: tool.schema.enum(["json", "text"]).optional().default("json").describe("Output format")
  },
  async execute(args) {
    try {
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
    } catch (error) {
      throw new Error(`Failed to fetch ready tasks: ${error.message}`)
    }
  }
})

export const status = tool({
  description: "Update status of a Beads task",
  args: {
    taskId: tool.schema.string().describe("Task ID (e.g., bd-a1b2)"),
    status: tool.schema.enum(["ready", "in_progress", "blocked", "done"]).describe("New status")
  },
  async execute(args) {
    try {
      await $`bd update ${args.taskId} --status ${args.status}`
      return `Task ${args.taskId} updated to status: ${args.status}`
    } catch (error) {
      throw new Error(`Failed to update task ${args.taskId}: ${error.message}`)
    }
  }
})

export const close = tool({
  description: "Close a Beads task with a completion reason",
  args: {
    taskId: tool.schema.string().describe("Task ID to close"),
    reason: tool.schema.string().describe("Reason for completion")
  },
  async execute(args) {
    try {
      await $`bd close ${args.taskId} --reason ${args.reason}`
      return `Task ${args.taskId} closed: ${args.reason}`
    } catch (error) {
      throw new Error(`Failed to close task ${args.taskId}: ${error.message}`)
    }
  }
})

export const create = tool({
  description: "Create a new Beads task",
  args: {
    title: tool.schema.string().describe("Task title"),
    type: tool.schema.enum(["feature", "bug", "chore", "doc", "task"]).optional().describe("Task type"),
    priority: tool.schema.enum(["low", "normal", "high", "urgent"]).optional().describe("Priority level"),
    parent: tool.schema.string().optional().describe("Parent task ID")
  },
  async execute(args) {
    try {
      const cmdArgs = ["bd", "create", args.title]
      
      if (args.type) {
        cmdArgs.push("-t", args.type)
      }
      if (args.priority) {
        cmdArgs.push("-p", args.priority)
      }
      if (args.parent) {
        cmdArgs.push("--parent", args.parent)
      }

      const proc = Bun.spawn(cmdArgs, {
        stdout: "pipe",
        stderr: "pipe"
      })
      
      const output = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      
      if (exitCode !== 0) {
        const error = await new Response(proc.stderr).text()
        throw new Error(error || "Command failed")
      }
      
      return output
    } catch (error) {
      throw new Error(`Failed to create task: ${error.message}`)
    }
  }
})

export const list = tool({
  description: "List Beads tasks with optional filters",
  args: {
    status: tool.schema.enum(["ready", "in_progress", "blocked", "done", "closed"]).optional().describe("Filter by status"),
    tag: tool.schema.string().optional().describe("Filter by tag"),
    format: tool.schema.enum(["json", "text"]).optional().default("json").describe("Output format")
  },
  async execute(args) {
    try {
      const cmdArgs = ["bd", "list", "--json"]

      if (args.status) {
        cmdArgs.push("--status", args.status)
      }
      if (args.tag) {
        cmdArgs.push("--tag", args.tag)
      }

      const proc = Bun.spawn(cmdArgs, {
        stdout: "pipe",
        stderr: "pipe"
      })
      
      const result = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      
      if (exitCode !== 0) {
        const error = await new Response(proc.stderr).text()
        throw new Error(error || "Command failed")
      }
      
      return result
    } catch (error) {
      throw new Error(`Failed to list tasks: ${error.message}`)
    }
  }
})

export const addDependency = tool({
  description: "Add a dependency between two Beads tasks",
  args: {
    childId: tool.schema.string().describe("Child task ID"),
    parentId: tool.schema.string().describe("Parent task ID"),
    type: tool.schema.enum(["blocks", "related", "discovered-from", "parent-child"]).optional().default("blocks").describe("Dependency type")
  },
  async execute(args) {
    try {
      await $`bd dep add ${args.childId} ${args.parentId} --type ${args.type}`
      return `Added ${args.type} dependency: ${args.childId} → ${args.parentId}`
    } catch (error) {
      throw new Error(`Failed to add dependency: ${error.message}`)
    }
  }
})
