/**
 * Context File Manager Skill
 * 
 * Manages creation, updating, and maintenance of context files
 * Uses Context7 MCP for design guidance
 */

import { Context7Client } from './context7-client';
import { FileManager } from './file-manager';
import { TemplateManager } from './template-manager';
import { ProgressTracker } from './progress-tracker';
import { DeferredItemsManager } from './deferred-items-manager';
import { DecisionLogger } from './decision-logger';
import { RiskLogger } from './risk-logger';

export interface ContextFileConfig {
  coreFiles: string[];
  supportingFiles: string[];
  context7Mappings: Record<string, string>;
  templatesDir: string;
  outputDir: string;
}

export interface ContextFileManagerOptions {
  config: ContextFileConfig;
  context7Client: Context7Client;
}

export class ContextFileManager {
  private config: ContextFileConfig;
  private context7Client: Context7Client;
  private fileManager: FileManager;
  private templateManager: TemplateManager;
  private progressTracker: ProgressTracker;
  private deferredItemsManager: DeferredItemsManager;
  private decisionLogger: DecisionLogger;
  private riskLogger: RiskLogger;

  constructor(options: ContextFileManagerOptions) {
    this.config = options.config;
    this.context7Client = options.context7Client;
    this.fileManager = new FileManager(options.config.outputDir);
    this.templateManager = new TemplateManager(options.config.templatesDir);
    this.progressTracker = new ProgressTracker(options.config.outputDir);
    this.deferredItemsManager = new DeferredItemsManager(options.config.outputDir);
    this.decisionLogger = new DecisionLogger(options.config.outputDir);
    this.riskLogger = new RiskLogger(options.config.outputDir);
  }

  /**
   * Initialize all core context files
   */
  async initializeAll(): Promise<void> {
    console.log('🚀 Initializing all core context files...');
    
    for (const file of this.config.coreFiles) {
      await this.createFileIfNotExists(file);
    }
    
    console.log('✅ All core context files initialized');
  }

  /**
   * Create a specific context file
   */
  async createFile(fileType: string, useContext7: boolean = true): Promise<boolean> {
    const filePath = this.getFilePath(fileType);
    
    if (await this.fileManager.exists(filePath)) {
      console.log(`⚠️ ${fileType} already exists at ${filePath}`);
      return false;
    }

    console.log(`📝 Creating ${fileType}...`);
    
    let content: string;
    
    if (useContext7 && this.config.context7Mappings[fileType]) {
      const designGuidance = await this.getDesignGuidance(fileType);
      content = await this.templateManager.renderWithGuidance(fileType, designGuidance);
    } else {
      content = await this.templateManager.render(fileType);
    }

    await this.fileManager.write(filePath, content);
    console.log(`✅ Created ${fileType} at ${filePath}`);
    
    return true;
  }

  /**
   * Create file only if it doesn't exist
   */
  async createFileIfNotExists(fileType: string, useContext7: boolean = true): Promise<boolean> {
    const filePath = this.getFilePath(fileType);
    
    if (await this.fileManager.exists(filePath)) {
      console.log(`✅ ${fileType} already exists at ${filePath}`);
      return false;
    }

    return await this.createFile(fileType, useContext7);
  }

  /**
   * Update progress tracker
   */
  async updateProgress(
    domain: string,
    status: 'pending' | 'in-progress' | 'complete',
    details?: string
  ): Promise<void> {
    await this.progressTracker.update(domain, status, details);
    console.log(`📊 Updated progress: ${domain} -> ${status}`);
  }

  /**
   * Add deferred item
   */
  async addDeferredItem(
    description: string,
    priority: 'low' | 'medium' | 'high' = 'medium',
    context?: string
  ): Promise<void> {
    await this.deferredItemsManager.add({
      description,
      priority,
      context,
      status: 'PENDING',
      addedAt: new Date().toISOString()
    });
    console.log(`📝 Added deferred item: ${description}`);
  }

  /**
   * Log architectural decision
   */
  async logDecision(
    decision: string,
    rationale: string,
    alternatives?: string[]
  ): Promise<void> {
    await this.decisionLogger.log({
      decision,
      rationale,
      alternatives,
      date: new Date().toISOString()
    });
    console.log(`📋 Logged decision: ${decision}`);
  }

  /**
   * Log project risk
   */
  async logRisk(
    risk: string,
    impact: 'low' | 'medium' | 'high',
    likelihood: 'low' | 'medium' | 'high',
    mitigation?: string
  ): Promise<void> {
    await this.riskLogger.log({
      risk,
      impact,
      likelihood,
      mitigation,
      date: new Date().toISOString()
    });
    console.log(`⚠️ Logged risk: ${risk}`);
  }

  /**
   * Get design guidance from Context7
   */
  private async getDesignGuidance(fileType: string): Promise<string> {
    const libraryId = this.config.context7Mappings[fileType];
    
    if (!libraryId) {
      return '';
    }

    try {
      const query = this.getFileQuery(fileType);
      const guidance = await this.context7Client.queryDocs(libraryId, query);
      return guidance;
    } catch (error) {
      console.warn(`⚠️ Failed to get Context7 guidance for ${fileType}: ${error}`);
      return '';
    }
  }

  /**
   * Get query for Context7 based on file type
   */
  private getFileQuery(fileType: string): string {
    const queries: Record<string, string> = {
      'project-overview': 'Next.js project structure best practices',
      'architecture': 'Gin web framework architecture patterns',
      'code-standards': 'Playwright TypeScript best practices',
      'progress-tracker': 'Next.js project management patterns',
      'ui-registry': 'Next.js component architecture patterns',
      'ui-rules': 'Next.js App Router component patterns',
      'ui-tokens': 'Tailwind CSS design tokens patterns',
      'library-docs': 'Playwright library usage patterns',
      'user-preferences': 'Next.js development workflow patterns',
      'Agent': 'Next.js AI agent development patterns'
    };

    return queries[fileType] || 'best practices';
  }

  /**
   * Get file path for context file
   */
  private getFilePath(fileType: string): string {
    const extension = '.md';
    return `${fileType}${extension}`;
  }

  /**
   * Show status of all context files
   */
  async showStatus(): Promise<void> {
    console.log('\n📋 Context Files Status:');
    console.log('========================\n');
    
    for (const file of [...this.config.coreFiles, ...this.config.supportingFiles]) {
      const filePath = this.getFilePath(file);
      const exists = await this.fileManager.exists(filePath);
      const status = exists ? '✅ EXISTS' : '❌ MISSING';
      console.log(`  ${file}: ${status}`);
    }
  }

  /**
   * Validate all context files against standards
   */
  async validateAll(): Promise<void> {
    console.log('\n🔍 Validating context files...\n');
    
    for (const file of [...this.config.coreFiles, ...this.config.supportingFiles]) {
      const filePath = this.getFilePath(file);
      const exists = await this.fileManager.exists(filePath);
      
      if (!exists) {
        console.log(`❌ ${file}: MISSING`);
        continue;
      }
      
      const content = await this.fileManager.read(filePath);
      const isValid = this.validateContent(file, content);
      
      console.log(`${isValid ? '✅' : '⚠️'} ${file}: ${isValid ? 'VALID' : 'NEEDS REVIEW'}`);
    }
  }

  /**
   * Validate content against basic standards
   */
  private validateContent(fileType: string, content: string): boolean {
    // Basic validation - check for required sections
    const requiredSections: Record<string, string[]> = {
      'project-overview': ['## What It Is', '## Key Features', '## Architecture'],
      'architecture': ['## System Overview', '## Component Diagram', '## Data Flow'],
      'code-standards': ['## Go Backend', '## TypeScript', '## Next.js'],
      'progress-tracker': ['## Current Status', '## Milestones', '## Timeline'],
      'ui-registry': ['## Directory Structure', '## Component Inventory'],
      'ui-rules': ['## Component Architecture', '## Styling Rules', '## Accessibility'],
      'ui-tokens': ['## CSS Custom Properties', '## Color Palette', '## Typography'],
      'user-preferences': ['## Development Workflow', '## Technical Preferences'],
      'Agent': ['## Purpose', '## Guidelines', '## Workflows']
    };

    const required = requiredSections[fileType] || [];
    
    for (const section of required) {
      if (!content.includes(section)) {
        console.warn(`  Missing section: ${section}`);
        return false;
      }
    }

    return true;
  }
}

export default ContextFileManager;