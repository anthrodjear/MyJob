/**
 * Template Manager
 * 
 * Manages templates for context files with Context7 guidance integration
 */

import * as fs from 'fs/promises';
import * as path from 'path';

export class TemplateManager {
  private templatesDir: string;
  private templates: Map<string, string> = new Map();

  constructor(templatesDir: string) {
    this.templatesDir = templatesDir;
  }

  /**
   * Initialize templates from directory
   */
  async initialize(): Promise<void> {
    try {
      const files = await fs.readdir(this.templatesDir);
      
      for (const file of files) {
        if (file.endsWith('.md') || file.endsWith('.template')) {
          const content = await fs.readFile(
            path.join(this.templatesDir, file),
            'utf-8'
          );
          const name = file.replace(/\.(md|template)$/, '');
          this.templates.set(name, content);
        }
      }
    } catch (error) {
      console.warn(`Failed to load templates: ${error}`);
    }
  }

  /**
   * Render template with optional Context7 guidance
   */
  async render(templateName: string, guidance?: string): Promise<string> {
    const template = this.templates.get(templateName) || this.getDefaultTemplate(templateName);
    
    if (guidance) {
      return this.injectGuidance(template, guidance);
    }
    
    return template;
  }

  /**
   * Render template with Context7 guidance
   */
  async renderWithGuidance(templateName: string, guidance: string): Promise<string> {
    return this.render(templateName, guidance);
  }

  /**
   * Inject Context7 guidance into template
   */
  private injectGuidance(template: string, guidance: string): string {
    // Replace placeholder with guidance
    const guidanceSection = `
## Context7 Design Guidance

${guidance}

---
`;
    
    // Insert after first heading or at beginning
    const headingMatch = template.match(/^# .+$/m);
    if (headingMatch) {
      const insertIndex = template.indexOf(headingMatch[0]) + headingMatch[0].length;
      return template.slice(0, insertIndex) + guidanceSection + template.slice(insertIndex);
    }
    
    return guidanceSection + template;
  }

  /**
   * Get default template for file type
   */
  private getDefaultTemplate(fileType: string): string {
    const templates: Record<string, string> = {
      'project-overview': this.getProjectOverviewTemplate(),
      'architecture': this.getArchitectureTemplate(),
      'code-standards': this.getCodeStandardsTemplate(),
      'progress-tracker': this.getProgressTrackerTemplate(),
      'ui-registry': this.getUIRegistryTemplate(),
      'ui-rules': this.getUIRulesTemplate(),
      'ui-tokens': this.getUITokensTemplate(),
      'library-docs': this.getLibraryDocsTemplate(),
      'user-preferences': this.getUserPreferencesTemplate(),
      'Agent': this.getAgentTemplate(),
      'deferred-items': this.getDeferredItemsTemplate(),
      'decision-log': this.getDecisionLogTemplate(),
      'risk-log': this.getRiskLogTemplate(),
      'timeline': this.getTimelineTemplate()
    };

    return templates[fileType] || '# ' + fileType + '\n\nContent to be added.\n';
  }

  // Default templates
  private getProjectOverviewTemplate(): string {
    return `# Project Overview

## What It Is

A brief description of the project.

## Who It's For

Target audience and use cases.

## Key Features

| Capability | Description |
|---|---|
| Feature 1 | Description |
| Feature 2 | Description |

## Architecture

| Layer | Technology | Role |
|---|---|---|
| Layer 1 | Tech 1 | Role 1 |

## Design Principles

- Principle 1
- Principle 2

## Current Status

**Phase:** Current phase description

## What's Built

- [ ] Item 1
- [ ] Item 2

## What's Next

- [ ] Next item 1
- [ ] Next item 2
`;
  }

  private getArchitectureTemplate(): string {
    return `# Architecture

## System Overview

High-level system description.

## Component Diagram

\`\`\`
Component diagram here
\`\`\`

## Data Flow

### Flow 1

\`\`\`
Flow description
\`\`\`

## Domain Model

\`\`\`
Domain model diagram
\`\`\`

## Technology Choices

### Technology 1

| Decision | Choice | Rationale |
|---|---|---|
| Decision | Choice | Reason |

## Deployment Topology

\`\`\`yaml
# Deployment config
\`\`\`

## Task-Based API Pattern

\`\`\`
Async pattern description
\`\`\`
`;
  }

  private getCodeStandardsTemplate(): string {
    return `# Code Standards

## Project Overview

| Component | Stack | Entry Point |
|---|---|---|
| Backend | Language, Framework | Path |
| Frontend | Framework | Path |

## Language-Specific Standards

### Backend Language

\`\`\`
Standards here
\`\`\`

### Frontend Language

\`\`\`
Standards here
\`\`\`

## Shared Patterns

- Pattern 1
- Pattern 2

## Testing

### General

Testing standards

### Language-Specific

Test patterns
`;
  }

  private getProgressTrackerTemplate(): string {
    return `# Project Progress Tracker

## Current Status

| Field | Value |
|---|---|
| **Project** | Project Name |
| **Active Phase** | Phase Name |
| **Phase Progress** | X% |
| **Overall Progress** | X% |
| **Blockers** | None |
| **Next Up** | Next tasks |

## Milestones

### Phase 1: Phase Name

#### 1.1 Milestone Name

| Milestone | Status | Notes |
|---|---|---|
| Item 1 | Done | Notes |

## Upcoming Tasks

1. Task 1
2. Task 2

## Timeline

| Milestone | Target | Actual | Status |
|---|---|---|---|
| Milestone | Date | Date | Status |

## Risk Log

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| Risk 1 | High | Medium | Mitigation |

## Decision Log

| Date | Decision | Rationale |
|---|---|---|
| Date | Decision | Reason |
`;
  }

  private getUIRegistryTemplate(): string {
    return `# UI Component Registry

## Directory Structure

\`\`\`
src/
├── app/
├── components/
└── lib/
\`\`\`

## Component Inventory

### Category

| Component | Type | Purpose |
|---|---|---|
| ComponentName | Server/Client | Description |

## Component Hierarchy

\`\`\`
RootComponent
├── ChildComponent
└── AnotherChild
\`\`\`
`;
  }

  private getUIRulesTemplate(): string {
    return `# UI Design Rules

## Component Architecture

### Server Components (default)

Rules for server components

### Client Components (opt-in)

Rules for client components

## Styling Rules

- Tailwind CSS for all styling
- Design tokens from ui-tokens.md

## Accessibility Requirements

- Semantic HTML
- Focus management
- ARIA labels

## Data Display Patterns

Pattern descriptions

## Interaction Patterns

Interaction rules
`;
  }

  private getUITokensTemplate(): string {
    return `# UI Design Tokens

## CSS Custom Properties

\`\`\`css
:root {
  --color-primary: #value;
  --color-success: #value;
  --color-warning: #value;
  --color-danger: #value;
}
\`\`\`

## Color Palette Reference

| Token | Hex | Usage |
|---|---|---|
| --color-primary | #value | Usage |

## Typography Scale

| Level | Size | Weight | Usage |
|---|---|---|---|
| text-sm | 14px | 400 | Body default |

## Spacing Scale

| Token | Value | Usage |
|---|---|---|
| 4 | 16px | Standard padding |
`;
  }

  private getLibraryDocsTemplate(): string {
    return `# Library Documentation

## Library Name

Usage patterns and gotchas

## Key Patterns

- Pattern 1
- Pattern 2

## Common Gotchas

- Gotcha 1
- Gotcha 2
`;
  }

  private getUserPreferencesTemplate(): string {
    return `# User Preferences

## Development Workflow

### Code Quality

- Preference 1
- Preference 2

## Technical Preferences

### Stack

- Technology choices

## Architecture Preferences

- Architectural decisions

## Communication Preferences

- Communication style
`;
  }

  private getAgentTemplate(): string {
    return `# Agent Documentation

## Purpose

Development agent for the project.

## Guidelines

### File Creation

Guidelines for creating files

### Code Review

Code review process

### Testing

Testing requirements

## Workflows

### Context Management

How to manage context

### Task Execution

Task execution workflow

## References

Links to relevant documentation
`;
  }

  private getDeferredItemsTemplate(): string {
    return `# Deferred Items

## Items

| ID | Description | Priority | Status | Added | Context |
|---|---|---|---|---|---|
| | | | PENDING | | |
`;
  }

  private getDecisionLogTemplate(): string {
    return `# Decision Log

## Decisions

| Date | Decision | Rationale |
|---|---|---|
| | | |
`;
  }

  private getRiskLogTemplate(): string {
    return `# Risk Log

## Risks

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| | | | |
`;
  }

  private getTimelineTemplate(): string {
    return `# Timeline

## Milestones

| Milestone | Target | Actual | Status |
|---|---|---|---|
| | | | |
`;
  }
}

export default TemplateManager;