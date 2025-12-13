# Branch Hierarchy and Parent-Child Relationships

## Overview

The `createSpeculativeBranches` function has been enhanced to fully track parent-child relationships in the speculative branch hierarchy. This enables coordinated multi-branch workflows, hierarchical failure propagation through the kill switch, and efficient querying of branch genealogy.

## Data Structure: Parent-Child Relationships

Each `SpeculativeBranch` now maintains:

```go
type SpeculativeBranch struct {
    ID          string          // Unique branch ID
    Changes     []ChangeRequest // Changes being tested together
    Depth       int             // How many levels deep (1=base, 2=base+1, etc)
    Status      BranchStatus

    // Parent-child hierarchy tracking
    ParentID    string   // ID of parent branch (empty for base branch)
    ChildrenIDs []string // IDs of child branches spawned from this one

    // Kill switch metadata
    KilledAt    *time.Time // Timestamp when branch was killed (nil if not killed)
    KillReason  string     // Explanation of why branch was killed
}
```

## Hierarchy Structure

Speculative branches form a linear parent-child tree:

```
Depth 1 (Base):        [C1]
Depth 2:               [C1, C2]
Depth 3:               [C1, C2, C3]
Depth 4:               [C1, C2, C3, C4]
...
```

Each branch at depth `n` has:
- Exactly one parent branch at depth `n-1` (except root)
- Exactly one child branch at depth `n+1` (except leaf)
- No siblings (linear hierarchy)

## Creating Branches with Hierarchy

The `createSpeculativeBranchesImpl` function sets up parent-child relationships when creating branches:

```go
func (c *Coordinator) createSpeculativeBranchesImpl(ctx context.Context, batch []*ChangeRequest) {
    for depth := 1; depth <= len(batch); depth++ {
        changes := batch[:depth]
        branchID := c.generateBranchID(changes)

        // Determine parent branch (previous depth level)
        var parentID string
        if depth > 1 {
            parentChanges := batch[:depth-1]
            parentID = c.generateBranchID(parentChanges)
        }

        // Create branch with parent reference
        branch := &SpeculativeBranch{
            ID:          branchID,
            ParentID:    parentID,
            ChildrenIDs: []string{},
            Depth:       depth,
        }

        // Update parent's children list
        if parentID != "" {
            if parent, exists := c.activeBranches[parentID]; exists {
                parent.ChildrenIDs = append(parent.ChildrenIDs, branchID)
            }
        }

        c.activeBranches[branchID] = branch
    }
}
```

## Query Functions

### GetBranchAncestry

Returns the complete ancestry chain for a branch, from root to leaf.

```go
func (c *Coordinator) GetBranchAncestry(branchID string) []string
```

**Example:**
```
Given hierarchy: Branch A -> Branch B -> Branch C
GetBranchAncestry("branch-C") returns ["branch-A", "branch-B", "branch-C"]
```

**Use cases:**
- Understand the speculative depth chain
- Trace which changes are being tested together
- Collect all changes that must pass for a branch to be valid

### GetBranchDescendants

Returns all descendant branches (children, grandchildren, etc.) for a branch in breadth-first order.

```go
func (c *Coordinator) GetBranchDescendants(branchID string) []string
```

**Example:**
```
Given hierarchy: Branch A -> Branch B -> Branch C
                         -> Branch D
GetBranchDescendants("branch-A") returns ["branch-B", "branch-D", "branch-C"]
```

**Use cases:**
- Find all branches affected by a change
- Identify all branches dependent on a specific ancestor
- Determine the scope of cascade operations

### GetBranchSiblings

Returns all sibling branches (branches with same parent).

```go
func (c *Coordinator) GetBranchSiblings(branchID string) []string
```

**Example:**
```
Given hierarchy: Branch A -> Branch B
                         -> Branch C
GetBranchSiblings("branch-B") returns ["branch-C"]
```

**Use cases:**
- Compare parallel test runs
- Understand branching points
- Coordinate alternative test strategies

### IsAncestorOf

Checks if one branch is an ancestor of another.

```go
func (c *Coordinator) IsAncestorOf(ancestorID, branchID string) bool
```

**Example:**
```
Given hierarchy: Branch A -> Branch B -> Branch C
IsAncestorOf("branch-A", "branch-C") returns true
IsAncestorOf("branch-C", "branch-A") returns false
```

**Use cases:**
- Verify containment relationships
- Check if a change is being tested in a specific branch
- Prevent circular references

### GetBranchHierarchy

Returns the complete hierarchical structure rooted at the specified branch as a tree.

```go
type BranchNode struct {
    ID       string
    Depth    int
    Status   BranchStatus
    Children []*BranchNode
}

func (c *Coordinator) GetBranchHierarchy(branchID string) *BranchNode
```

**Example output:**
```
{
  "id": "branch-agent-1",
  "depth": 1,
  "status": "testing",
  "children": [
    {
      "id": "branch-agent-1-agent-2",
      "depth": 2,
      "status": "testing",
      "children": [
        {
          "id": "branch-agent-1-agent-2-agent-3",
          "depth": 3,
          "status": "pending",
          "children": []
        }
      ]
    }
  ]
}
```

**Use cases:**
- Visualize branch hierarchies
- Export to dashboards or monitoring systems
- Understand the complete test topology
- Debug branch creation issues

### CollectBranchFamily

Returns all branches in the same family (root and all descendants).

```go
func (c *Coordinator) CollectBranchFamily(branchID string) []string
```

**Example:**
```
Given hierarchy: Branch A -> Branch B -> Branch C
CollectBranchFamily("branch-B") returns ["branch-A", "branch-B", "branch-C"]
```

**Use cases:**
- Collect all related branches for bulk operations
- Clean up all branches in a family
- Aggregate metrics across a family
- Perform family-wide status updates

## Cascade Operations

### CascadeStatusUpdate

Updates a branch status and propagates the change to all descendants.

```go
func (c *Coordinator) CascadeStatusUpdate(branchID string, newStatus BranchStatus) error
```

**Example:**
```go
// Mark entire branch family as failed
err := coord.CascadeStatusUpdate("branch-agent-1", BranchStatusFailed)
// This marks branch-agent-1, branch-agent-1-agent-2, and
// branch-agent-1-agent-2-agent-3 as failed
```

**Use cases:**
- Cascade failure status through hierarchy
- Suspend entire branch families
- Mark branches as canceled
- Propagate retry decisions

**Thread Safety:**
- Safe to call from multiple goroutines
- Acquires lock briefly for each level
- Continues cascading even if errors occur

## Integration with Kill Switch

The parent-child relationships enable the kill switch to cascade failures:

```go
func (c *Coordinator) processTestResult(ctx context.Context, result *TestResult) {
    if !result.Passed {
        failedBranchID := c.findBranchByResult(result)

        // Kill all dependent branches first
        c.killDependentBranches(ctx, failedBranchID)

        // Then kill the failed branch itself
        c.killFailedBranch(ctx, failedBranchID, "tests failed")
    }
}

func (c *Coordinator) killDependentBranches(ctx context.Context, branchID string) error {
    // Get all children
    descendants := c.GetBranchDescendants(branchID)

    // Kill them in order (breadth-first ensures children are killed before grandchildren)
    for _, childID := range descendants {
        c.killFailedBranch(ctx, childID, fmt.Sprintf("parent %s failed", branchID))
    }

    return nil
}
```

## Example Workflow

```go
// 1. Create speculative branches with automatic hierarchy
batch := []*ChangeRequest{
    {ID: "agent-1"},
    {ID: "agent-2"},
    {ID: "agent-3"},
}
coord.createSpeculativeBranches(ctx, batch)

// 2. Query branch relationships
ancestry := coord.GetBranchAncestry("branch-agent-1-agent-2-agent-3")
// Returns: ["branch-agent-1", "branch-agent-1-agent-2", "branch-agent-1-agent-2-agent-3"]

descendants := coord.GetBranchDescendants("branch-agent-1")
// Returns: ["branch-agent-1-agent-2", "branch-agent-1-agent-2-agent-3"]

isAncestor := coord.IsAncestorOf("branch-agent-1", "branch-agent-1-agent-2")
// Returns: true

// 3. Get complete hierarchy for visualization
tree := coord.GetBranchHierarchy("branch-agent-1")
// Returns: BranchNode with all descendants as children

// 4. Collect family for bulk operations
family := coord.CollectBranchFamily("branch-agent-1-agent-2")
// Returns: ["branch-agent-1", "branch-agent-1-agent-2", "branch-agent-1-agent-2-agent-3"]

// 5. Cascade operations
coord.CascadeStatusUpdate("branch-agent-1", BranchStatusFailed)
// Marks all branches in the family as failed
```

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| `GetBranchAncestry` | O(depth) | Walks up parent chain |
| `GetBranchDescendants` | O(children) | BFS traversal |
| `GetBranchSiblings` | O(parent.children) | Linear scan |
| `IsAncestorOf` | O(depth) | Walks up parent chain |
| `GetBranchHierarchy` | O(descendants) | Recursive tree building |
| `CollectBranchFamily` | O(descendants) | Combines ancestry + descendants |
| `CascadeStatusUpdate` | O(descendants) | Updates all descendants |

## Thread Safety

All query functions are safe to call concurrently:
- `GetBranchAncestry` - acquires read lock
- `GetBranchDescendants` - acquires read lock
- `GetBranchSiblings` - acquires read lock
- `IsAncestorOf` - acquires read lock
- `GetBranchHierarchy` - acquires read lock
- `CollectBranchFamily` - acquires read lock

Cascade operations are safe but may cause blocking:
- `CascadeStatusUpdate` - acquires lock briefly for each level

## Testing

Comprehensive test coverage includes:

1. **Hierarchy Creation Tests:**
   - `TestCreateSpeculativeBranches_CreatesCorrectHierarchy` - Verifies correct ParentID/ChildrenIDs
   - `TestCreateSpeculativeBranches_AvoidsDuplicates` - Prevents re-creation

2. **Ancestry Query Tests:**
   - `TestGetBranchAncestry_CreatesCorrectChain` - Full ancestry chain
   - `TestGetBranchAncestry_NonExistent` - Handles missing branches

3. **Descendant Query Tests:**
   - `TestGetBranchDescendants_ReturnsAllDescendants` - BFS traversal
   - `TestGetBranchDescendants_NonExistent` - Handles missing branches

4. **Sibling Query Tests:**
   - `TestGetBranchSiblings_ReturnsCorrectSiblings` - Sibling identification
   - `TestGetBranchSiblings_NonExistent` - Handles missing branches

5. **Ancestor Relationship Tests:**
   - `TestIsAncestorOf_CorrectlyIdentifiesAncestry` - Relationship checking
   - Covers all relationship types (ancestor, descendant, unrelated)

6. **Cascade Operation Tests:**
   - `TestCascadeStatusUpdate_UpdatesAllDescendants` - Status propagation
   - `TestCascadeStatusUpdate_NonExistentBranch` - Error handling

7. **Family Collection Tests:**
   - `TestCollectBranchFamily_ReturnsRootAndDescendants` - Full family
   - `TestCollectBranchFamily_NonExistent` - Handles missing branches

8. **Hierarchy Tree Tests:**
   - `TestGetBranchHierarchy_BuildsCorrectTree` - Tree structure
   - `TestGetBranchHierarchy_NonExistent` - Handles missing branches

9. **Integration Tests:**
   - `TestBranchHierarchyIntegration` - Multiple operations together

## See Also

- `KILLSWITCH.md` - Kill switch architecture and cascade failure handling
- `speculative_execution.go` - Branch creation and execution
- `coordinator.go` - Main coordinator implementation
- `types.go` - Data structure definitions
