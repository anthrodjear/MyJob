/**
 * Progress Tracker
 * 
 * Manages progress tracking for context files
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import { FileManager } from './file-manager';

export interface DomainProgress {
  domain: string;
  status: 'pending' | 'in-progress' | 'complete';
  details?: string;
  completedAt?: string;
  files: string[];
}

export interface ProgressData {
  project: string;
  activePhase: string;
  phaseProgress: string;
  overallProgress: string;
  blockers: string;
  nextUp: string;
  domains: DomainProgress[];
  lastUpdated: string;
}

export class ProgressTracker {
  private fileManager: FileManager;
  private fileName: string;

  constructor(outputDir: string) {
    this.fileManager = new FileManager(outputDir);
    this.fileName = 'progress-tracker.md';
  }

  /**
   * Read current progress tracker
   */
  async read(): Promise<ProgressData> {
    const exists = await this.fileManager.exists(this.fileName);
    
    if (!exists) {
      return this.getDefaultProgress();
    }

    const content = await this.fileManager.read(this.fileName);
    return this.parseProgress(content);
  }

  /**
   * Update progress for a domain
   */
  async update(
    domain: string, 
    status: 'pending' | 'in-progress' | 'complete',
    details?: string
  ): Promise<void> {
    const progress = await this.read();
    
    // Update or add domain
    const existingIndex = progress.domains.findIndex(d => d.domain === domain);
    
    const domainProgress: DomainProgress = {
      domain,
      status,
      details,
      completedAt: status === 'complete' ? new Date().toISOString() : undefined,
      files: this.getDomainFiles(domain)
    };

    if (existingIndex >= 0) {
      progress.domains[existingIndex] = domainProgress;
    } else {
      progress.domains.push(domainProgress);
    }

    // Update overall progress
    progress.lastUpdated = new Date().toISOString();
    progress.overallProgress = this.calculateOverallProgress(progress.domains);

    await this.write(progress);
  }

  /**
   * Write progress data to file
   */
  private async write(progress: ProgressData): Promise<void> {
    const content = this.formatProgress(progress);
    await this.fileManager.write(this.fileName, content);
  }

  /**
   * Format progress data as markdown
   */
  private formatProgress(progress: ProgressData): string {
    let content = `# Project Progress Tracker\n\n`;
    content += `> Auto-updated as milestones complete. Last updated: ${progress.lastUpdated.split('T')[0]}\n\n`;
    content += `---\n\n`;
    content += `## Current Status\n\n`;
    content += `| Field | Value |\n`;
    content += `|-------|-------|\n`;
    content += `| **Project** | ${progress.project} |\n`;
    content += `| **Active Phase** | ${progress.activePhase} |\n`;
    content += `| **Phase Progress** | ${progress.phaseProgress} |\n`;
    content += `| **Overall Progress** | ${progress.overallProgress} |\n`;
    content += `| **Blockers** | ${progress.blockers} |\n`;
    content += `| **Next Up** | ${progress.nextUp} |\n\n`;
    content += `---\n\n`;
    content += `## Milestones\n\n`;

    // Group by phase
    const phases = this.groupByPhase(progress.domains);
    
    for (const [phaseName, domains] of Object.entries(phases)) {
      content += `### ${phaseName}\n\n`;
      
      for (const domain of domains) {
        content += `#### ${domain.domain}\n\n`;
        content += `| File | Status |\n`;
        content += `|------|--------|\n`;
        
        for (const file of domain.files) {
          const status = domain.status === 'complete' ? '✅' : 
                        domain.status === 'in-progress' ? '🔄' : '⏳';
          content += `| ${file} | ${status} |\n`;
        }
        content += `\n`;
      }
    }

    content += `---\n\n`;
    content += `## Upcoming Tasks\n\n`;
    
    const pending = progress.domains.filter(d => d.status === 'pending');
    for (const domain of pending) {
      content += `1. **${domain.domain}** domain\n`;
    }

    content += `\n---\n\n`;
    content += `## Timeline\n\n`;
    content += `| Milestone | Target | Actual | Status |\n`;
    content += `|-----------|--------|--------|--------|\n`;
    
    for (const domain of progress.domains) {
      if (domain.completedAt) {
        content += `| ${domain.domain} complete | - | ${domain.completedAt.split('T')[0]} | ✅ Done |\n`;
      }
    }

    content += `\n---\n\n`;
    content += `## Risk Log\n\n`;
    content += `| Risk | Impact | Likelihood | Mitigation |\n`;
    content += `|------|--------|------------|------------|\n`;
    content += `| Example risk | Medium | Low | Mitigation strategy |\n\n`;

    content += `---\n\n`;
    content += `## Decision Log\n\n`;
    content += `| Date | Decision | Rationale |\n`;
    content += `|------|----------|-----------|\n`;
    content += `| ${new Date().toISOString().split('T')[0]} | Example decision | Reasoning |\n\n`;

    content += `---\n\n`;
    content += `*This file tracks project state. Update after completing any milestone or making a significant decision.*\n`;

    return content;
  }

  /**
   * Parse progress from markdown content
   */
  private parseProgress(content: string): ProgressData {
    // Basic parsing - in reality would be more sophisticated
    return this.getDefaultProgress();
  }

  /**
   * Get default progress structure
   */
  private getDefaultProgress(): ProgressData {
    return {
      project: 'AI Job Search Agent',
      activePhase: 'Phase 1 — Foundation',
      phaseProgress: 'Scaffolding 100% / Implementation ~60%',
      overallProgress: '~40%',
      blockers: 'None',
      nextUp: 'Worker task handlers + Browser Agent scrapers',
      domains: [
        { domain: 'tasks', status: 'complete', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go', 'dispatcher.go'] },
        { domain: 'auth', status: 'complete', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go', 'middleware/'] },
        { domain: 'jobs', status: 'complete', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'] },
        { domain: 'applications', status: 'complete', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'] },
        { domain: 'resumes', status: 'complete', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'] },
        { domain: 'scoring', status: 'in-progress', files: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'] }
      ],
      lastUpdated: new Date().toISOString()
    };
  }

  /**
   * Calculate overall progress percentage
   */
  private calculateOverallProgress(domains: DomainProgress[]): string {
    const total = domains.length;
    const complete = domains.filter(d => d.status === 'complete').length;
    const inProgress = domains.filter(d => d.status === 'in-progress').length;
    
    const percentage = Math.round(((complete + inProgress * 0.5) / total) * 100);
    return `${percentage}%`;
  }

  /**
   * Group domains by phase
   */
  private groupByPhase(domains: DomainProgress[]): Record<string, DomainProgress[]> {
    const phases: Record<string, DomainProgress[]> = {
      'Phase 1: Foundation': []
    };
    
    for (const domain of domains) {
      phases['Phase 1: Foundation'].push(domain);
    }
    
    return phases;
  }

  /**
   * Get expected files for a domain
   */
  private getDomainFiles(domain: string): string[] {
    const domainFiles: Record<string, string[]> = {
      tasks: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go', 'dispatcher.go'],
      auth: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go', 'middleware/'],
      jobs: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'],
      applications: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'],
      resumes: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go'],
      scoring: ['model.go', 'dto.go', 'repository.go', 'service.go', 'handler.go']
    };
    
    return domainFiles[domain] || [];
  }
}

export default ProgressTracker;