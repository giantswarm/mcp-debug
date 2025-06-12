---
description: 
globs: 
alwaysApply: false
---
- **Required Rule Structure:**
  ```markdown
  ---
  description: Clear, one-line description of what the rule enforces
  globs: path/to/files/*.ext, other/path/**/*
  alwaysApply: boolean
  ---

  - **Main Points in Bold**
    - Sub-points with details
    - Examples and explanations
  ```

- **File References:**
  - Use `[filename](mdc:path/to/file)` ([filename](mdc:filename)) to reference files
  - Example: [mcp-debugging.md](mdc:.roo/rules/mcp-debugging.md) for rule references
  - Example: [agent.go](mdc:cmd/agent.go) for code references

- **Code Examples:**
  - Use language-specific code blocks
  ```go
  // ✅ DO: Show good examples
  goodExample := true
  
  // ❌ DON'T: Show anti-patterns
  badExample := false
  ```

- **Rule Content Guidelines:**
  - Start with high-level overview
  - Include specific, actionable requirements
  - Show examples of correct implementation
  - Reference existing code when possible
  - Keep rules DRY by referencing other rules

- **Rule Maintenance:**
  - Update rules when new patterns emerge
  - Add examples from actual codebase
  - Remove outdated patterns
  - Cross-reference related rules

- **Best Practices:**
  - Use bullet points for clarity
  - Keep descriptions concise
  - Include both DO and DON'T examples
  - Reference actual code over theoretical examples
  - Use consistent formatting across rules
