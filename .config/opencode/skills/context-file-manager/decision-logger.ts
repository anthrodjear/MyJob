/**
 * Decision Logger
 * 
 * Logs architectural and technical decisions
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import { FileManager } from './file-manager';

export interface Decision {
  date: string;
  decision: string;
  rationale: string;
  alternatives?: string[];
}

export class DecisionLogger {
  private fileManager: FileManager;
  private fileName: string;

  constructor(outputDir: string) {
    this.fileManager = new FileManager(outputDir);
    this.fileName = 'decision-log.md';
  }

  /**
   * Log a decision
   */
  async log(decision: Omit<Decision, 'date'>): Promise<void> {
    const decisions = await this.read();
    
    const newDecision: Decision = {
      ...decision,
      date: new Date().toISOString()
    };

    decisions.push(newDecision);
    await this.write(decisions);
  }

  /**
   * Get all decisions
   */
  async getAll(): Promise<Decision[]> {
    return await this.read();
  }

  /**
   * Read decisions from file
   */
  private async read(): Promise<Decision[]> {
    const exists = await this.fileManager.exists(this.fileName);
    
    if (!exists) {
      return [];
    }

    const content = await this.fileManager.read(this.fileName);
    return this.parseDecisions(content);
  }

  /**
   * Write decisions to file
   */
  private async write(decisions: Decision[]): Promise<void> {
    const content = this.formatDecisions(decisions);
    await this.fileManager.write(this.fileName, content);
  }

  /**
   * Format decisions as markdown
   */
  private formatDecisions(decisions: Decision[]): string {
    let content = `# Decision Log\n\n`;
    content += `> Architectural and technical decisions with rationale.\n\n`;
    content += `---\n\n`;
    content += `## Decisions\n\n`;
    content += `| Date | Decision | Rationale | Alternatives |\n`;
    content += `|------|----------|-----------|--------------|\n`;
    
    // Sort by date descending (newest first)
    const sorted = [...decisions].sort((a, b) => b.date.localeCompare(a.date));
    
    for (const decision of sorted) {
      const alternatives = decision.alternatives?.join(', ') || 'N/A';
      content += `| ${decision.date.split('T')[0]} | ${decision.decision} | ${decision.rationale} | ${alternatives} |\n`;
    }
    
    content += `\n---\n\n`;
    content += `*Auto-updated. Log decisions with \`context-file-manager log-decision\`*\n`;
    
    return content;
  }

  /**
   * Parse decisions from markdown
   */
  private parseDecisions(content: string): Decision[] {
    // In reality would parse the markdown table
    return [];
  }
}

export default DecisionLogger;