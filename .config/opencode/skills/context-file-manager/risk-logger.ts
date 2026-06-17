/**
 * Risk Logger
 * 
 * Logs and tracks project risks
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import { FileManager } from './file-manager';

export interface Risk {
  risk: string;
  impact: 'low' | 'medium' | 'high';
  likelihood: 'low' | 'medium' | 'high';
  mitigation?: string;
  date: string;
  status: 'OPEN' | 'MITIGATED' | 'ACCEPTED' | 'CLOSED';
}

export class RiskLogger {
  private fileManager: FileManager;
  private fileName: string;

  constructor(outputDir: string) {
    this.fileManager = new FileManager(outputDir);
    this.fileName = 'risk-log.md';
  }

  /**
   * Log a risk
   */
  async log(risk: Omit<Risk, 'date' | 'status'>): Promise<void> {
    const risks = await this.read();
    
    const newRisk: Risk = {
      ...risk,
      date: new Date().toISOString(),
      status: 'OPEN'
    };

    risks.push(newRisk);
    await this.write(risks);
  }

  /**
   * Update risk status
   */
  async updateStatus(riskDescription: string, status: Risk['status']): Promise<void> {
    const risks = await this.read();
    const risk = risks.find(r => r.risk === riskDescription);
    
    if (risk) {
      risk.status = status;
      await this.write(risks);
    }
  }

  /**
   * Get all risks
   */
  async getAll(): Promise<Risk[]> {
    return await this.read();
  }

  /**
   * Get open risks
   */
  async getOpen(): Promise<Risk[]> {
    const risks = await this.read();
    return risks.filter(r => r.status === 'OPEN');
  }

  /**
   * Read risks from file
   */
  private async read(): Promise<Risk[]> {
    const exists = await this.fileManager.exists(this.fileName);
    
    if (!exists) {
      return [];
    }

    const content = await this.fileManager.read(this.fileName);
    return this.parseRisks(content);
  }

  /**
   * Write risks to file
   */
  private async write(risks: Risk[]): Promise<void> {
    const content = this.formatRisks(risks);
    await this.fileManager.write(this.fileName, content);
  }

  /**
   * Format risks as markdown
   */
  private formatRisks(risks: Risk[]): string {
    let content = `# Risk Log\n\n`;
    content += `> Project risks with impact, likelihood, and mitigation strategies.\n\n`;
    content += `---\n\n`;
    content += `## Open Risks\n\n`;
    content += `| Risk | Impact | Likelihood | Mitigation | Status |\n`;
    content += `|------|--------|------------|------------|--------|\n`;
    
    const openRisks = risks.filter(r => r.status === 'OPEN');
    
    for (const risk of openRisks) {
      const impactIcon = risk.impact === 'high' ? '🔴' : 
                        risk.impact === 'medium' ? '🟡' : '🟢';
      const likelihoodIcon = risk.likelihood === 'high' ? '🔴' : 
                            risk.likelihood === 'medium' ? '🟡' : '🟢';
      
      content += `| ${risk.risk} | ${impactIcon} ${risk.impact} | ${likelihoodIcon} ${risk.likelihood} | ${risk.mitigation || 'TBD'} | ${risk.status} |\n`;
    }
    
    if (openRisks.length === 0) {
      content += `| No open risks | | | | |\n`;
    }
    
    content += `\n---\n\n`;
    content += `## Closed/Accepted Risks\n\n`;
    content += `| Risk | Impact | Likelihood | Mitigation | Status | Date |\n`;
    content += `|------|--------|------------|------------|--------|------|\n`;
    
    const closedRisks = risks.filter(r => r.status !== 'OPEN');
    
    for (const risk of closedRisks) {
      const impactIcon = risk.impact === 'high' ? '🔴' : 
                        risk.impact === 'medium' ? '🟡' : '🟢';
      
      content += `| ${risk.risk} | ${impactIcon} ${risk.impact} | ${risk.likelihood} | ${risk.mitigation || 'TBD'} | ${risk.status} | ${risk.date.split('T')[0]} |\n`;
    }
    
    if (closedRisks.length === 0) {
      content += `| No closed risks | | | | | |\n`;
    }
    
    content += `\n---\n\n`;
    content += `## Risk Matrix\n\n`;
    content += `\`\`\`\n`;
    content += `Impact\\Likelihood  Low    Medium  High\n`;
    content += `Low                 🟢     🟡     🟡\n`;
    content += `Medium              🟡     🟡     🔴\n`;
    content += `High                🟡     🔴     🔴\n`;
    content += `\`\`\`\n\n`;
    content += `---\n\n`;
    content += `*Auto-updated. Log risks with \`context-file-manager log-risk\`*\n`;
    
    return content;
  }

  /**
   * Parse risks from markdown
   */
  private parseRisks(content: string): Risk[] {
    // In reality would parse the markdown table
    return [];
  }
}

export default RiskLogger;