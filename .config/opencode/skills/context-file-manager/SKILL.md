# Context File Manager Skill

## Description

A comprehensive skill for creating, managing, and updating context files in the project. Uses Context7 MCP for design guidance and follows established best practices for documentation structure.

## When to Use

- When starting a new project that needs context files
- When context files are missing and need to be created
- When updating existing context files as the project evolves
- When creating Agent.md and other documentation files
- When managing deferred items, decisions, and risk logs

## Triggers

- "create context files"
- "initialize project documentation"
- "update progress tracker"
- "create Agent.md"
- "manage deferred items"
- "log decisions"
- "track risks"

## Workflow

### 1. Initialization Phase
```bash
# Create all core context files
context-file-manager init

# Or create specific files
context-file-manager create project-overview
context-file-manager create architecture
context-file-manager create code-standards
context-file-manager create progress-tracker
context-file-manager create ui-registry
context-file-manager create ui-rules
context-file-manager create ui-tokens
context-file-manager create library-docs
context-file-manager create user-preferences
```

### 2. Progressive Creation
```bash
# Create files as needed
context-file-manager create-if-not-exists project-overview
context-file-manager create-if-not-exists architecture
```

### 3. Updates and Maintenance
```bash
# Update progress
context-file-manager update-progress --domain=jobs --status=complete --details="All 5 files implemented"

# Add deferred item
context-file-manager add-deferred "Add pagination to jobs API" --priority=high

# Log decision
context-file-manager log-decision "Use pgvector over dedicated vector DB" --rationale="Operational simplicity"

# Log risk
context-file-manager log-risk "ATS anti-scraping" --impact=high --likelihood=medium
```

## Features

### Core Context Files
1. **project-overview.md** - Project description, purpose, features, status
2. **architecture.md** - System design, component diagrams, data flows
3. **code-standards.md** - Coding conventions, patterns, best practices
4. **progress-tracker.md** - Milestone tracking, domain completion status
5. **ui-registry.md** - Component hierarchy, file structure
4. **ui-rules.md** - Component patterns, accessibility, styling
5. **ui-tokens.md** - Design tokens, colors, typography, spacing
6. **library-docs.md** - Library usage patterns and gotchas
7. **user-preferences.md** - Development workflow preferences
8. **Agent.md** - Development agent guidelines and workflows

### Supporting Files
- **deferred-items.md** - Track deferred work items
- **decision-log.md** - Document architectural decisions
- **risk-log.md** - Track project risks
- **timeline.md** - Project timeline and milestones

### Context7 MCP Integration
- Fetches design guidance from official documentation
- Validates against best practices
- Provides up-to-date patterns for each technology

## Configuration

```yaml
contextFileManager:
  # Core files that should always exist
  coreFiles:
    - project-overview
    - architecture
    - code-standards
    - progress-tracker
    - ui-registry
    - ui-rules
    - ui-tokens
    - library-docs
    - user-preferences
    - Agent
  
  # Supporting files
  supportingFiles:
    - deferred-items
    - decision-log
    - risk-log
    - timeline
  
  # Context7 library mappings for design guidance
  context7Mappings:
    project-overview: "/vercel/next.js"
    architecture: "/gin-gonic/gin"
    code-standards: "/microsoft/playwright"
    progress-tracker: "/vercel/next.js"
    ui-registry: "/vercel/next.js"
    ui-rules: "/vercel/next.js"
    ui-tokens: "/vercel/next.js"
    library-docs: "/microsoft/playwright"
    user-preferences: "/vercel/next.js"
    Agent: "/vercel/next.js"
  
  # File templates
  templatesDir: ".config/opencode/skills/context-file-manager/templates"
  
  # Output directory
  outputDir: "context"
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create all core context files |
| `create <file>` | Create specific context file |
| `create-if-not-exists <file>` | Create file only if missing |
| `update-progress` | Update progress tracker |
| `add-deferred` | Add deferred item |
| `log-decision` | Log architectural decision |
| `log-risk` | Log project risk |
| `status` | Show context file status |
| `validate` | Validate files against standards |

## Integration Points

- **Context7 MCP** - For design guidance and best practices
- **Git** - For version control of context files
- **Progress Tracker** - For milestone tracking
- **Code Standards** - For validation against conventions

## Examples

### Create All Context Files
```bash
context-file-manager init
```

### Create Specific File with Context7 Guidance
```bash
context-file-manager create architecture --use-context7=true
```

### Update Progress After Completing Domain
```bash
context-file-manager update-progress \
  --domain=scoring \
  --status=complete \
  --details="LLM scoring pipeline implemented with Ollama integration"
```

### Add Deferred Item
```bash
context-file-manager add-deferred \
  "Implement email classifier for recruiter emails" \
  --priority=medium \
  --context="Waiting for Microsoft Graph API integration"
```

## Best Practices

1. **Always check if file exists** before creating
2. **Use Context7 MCP** for design guidance on new files
3. **Follow existing patterns** in the codebase
4. **Update progress tracker** after completing work
5. **Document decisions** with rationale
6. **Track risks** with impact and likelihood
7. **Maintain deferred items** list for future work
8. **Validate against code standards** before committing

## Error Handling

- Gracefully handles missing files
- Provides clear error messages for missing dependencies
- Falls back to default templates if Context7 unavailable
- Validates file structure before writing

## Testing

- Unit tests for file creation logic
- Integration tests for Context7 MCP calls
- Validation tests for template rendering
- End-to-end tests for complete workflows