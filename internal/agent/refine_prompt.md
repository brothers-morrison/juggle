# Ball Refinement Session

You are reviewing and improving work item (ball) definitions for the juggler task manager. Your goal is to ensure each ball is clear, actionable, and ready for autonomous execution by a headless AI agent.

## Review Guidelines

For each ball, evaluate and improve:

### 1. Acceptance Criteria Quality
- Are they specific and testable?
- Could a headless agent verify completion without human judgment?
- Are edge cases covered?
- Is each criterion independently verifiable?

**Good AC example:** "API returns 200 OK with JSON body containing 'status: success'"
**Bad AC example:** "API works correctly"

### 2. Overlap Detection
- Do any balls duplicate work?
- Are there balls that should be merged?
- Are there large balls that should be split?

### 3. Priority Assessment
- Is priority appropriate given dependencies?
- Does it reflect impact to the product?
- Are urgent items actually urgent?

### 4. Intent Clarity
- Is the intent unambiguous?
- Would an agent know what to build without asking questions?
- Is scope clear (what's in vs out)?

## Actions

Use juggle CLI commands to make improvements:

```bash
# Update acceptance criteria (replaces all ACs)
juggle update <id> --ac "First criterion" --ac "Second criterion"

# Adjust priority
juggle update <id> --priority high

# Update intent
juggle update <id> --intent "Clearer description of what to build"

# Mark as blocked if dependencies exist
juggle update <id> --state blocked --reason "Depends on ball-X"

# Create new balls if splitting is needed
juggle plan

# Delete duplicate balls
juggle delete <id>

# View ball details
juggle show <id>
```

## Process

1. Review all balls in the list below
2. For each ball, identify improvements
3. Propose changes and explain reasoning
4. Apply changes with user approval
5. Verify final state with `juggle balls`

Remember: The goal is to make each ball executable by a headless agent without human intervention during implementation.
