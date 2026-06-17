/**
 * Context File Manager CLI
 * 
 * Command-line interface for the context file manager skill
 */

import { Command } from 'commander';
import { ContextFileManager } from './index';
import { Context7Client } from './context7-client';

const program = new Command();

program
  .name('context-file-manager')
  .description('Manage context files with Context7 MCP integration')
  .version('1.0.0');

// Initialize configuration
const config = {
  coreFiles: [
    'project-overview',
    'architecture',
    'code-standards',
    'progress-tracker',
    'ui-registry',
    'ui-rules',
    'ui-tokens',
    'library-docs',
    'user-preferences',
    'Agent'
  ],
  supportingFiles: [
    'deferred-items',
    'decision-log',
    'risk-log',
    'timeline'
  ],
  context7Mappings: {
    'project-overview': '/vercel/next.js',
    'architecture': '/gin-gonic/gin',
    'code-standards': '/microsoft/playwright',
    'progress-tracker': '/vercel/next.js',
    'ui-registry': '/vercel/next.js',
    'ui-rules': '/vercel/next.js',
    'ui-tokens': '/vercel/next.js',
    'library-docs': '/microsoft/playwright',
    'user-preferences': '/vercel/next.js',
    'Agent': '/vercel/next.js'
  },
  templatesDir: '.config/opencode/skills/context-file-manager/templates',
  outputDir: 'context'
};

const context7Client = new Context7Client();
const manager = new ContextFileManager({ config, context7Client });

// Initialize command
program
  .command('init')
  .description('Create all core context files')
  .option('-c, --use-context7', 'Use Context7 MCP for design guidance', true)
  .action(async (options) => {
    console.log('🚀 Initializing context files...');
    await manager.initializeAll();
    console.log('✅ Done!');
  });

// Create specific file
program
  .command('create <file>')
  .description('Create a specific context file')
  .option('-c, --use-context7', 'Use Context7 MCP for design guidance', true)
  .action(async (file, options) => {
    const validFiles = [...config.coreFiles, ...config.supportingFiles];
    
    if (!validFiles.includes(file)) {
      console.error(`❌ Invalid file: ${file}`);
      console.log(`Valid files: ${validFiles.join(', ')}`);
      process.exit(1);
    }
    
    await manager.createFile(file, options.useContext7);
  });

// Create file if not exists
program
  .command('create-if-not-exists <file>')
  .description('Create file only if it does not exist')
  .option('-c, --use-context7', 'Use Context7 MCP for design guidance', true)
  .action(async (file, options) => {
    const validFiles = [...config.coreFiles, ...config.supportingFiles];
    
    if (!validFiles.includes(file)) {
      console.error(`❌ Invalid file: ${file}`);
      process.exit(1);
    }
    
    await manager.createFileIfNotExists(file, options.useContext7);
  });

// Update progress
program
  .command('update-progress')
  .description('Update progress tracker')
  .requiredOption('-d, --domain <domain>', 'Domain name')
  .requiredOption('-s, --status <status>', 'Status (pending|in-progress|complete)')
  .option('--details <details>', 'Additional details')
  .action(async (options) => {
    await manager.updateProgress(
      options.domain,
      options.status as 'pending' | 'in-progress' | 'complete',
      options.details
    );
  });

// Add deferred item
program
  .command('add-deferred <description>')
  .description('Add a deferred item')
  .option('-p, --priority <priority>', 'Priority (low|medium|high)', 'medium')
  .option('--context <context>', 'Additional context')
  .action(async (description, options) => {
    await manager.addDeferredItem(
      description,
      options.priority as 'low' | 'medium' | 'high',
      options.context
    );
  });

// Log decision
program
  .command('log-decision <decision>')
  .description('Log an architectural decision')
  .requiredOption('-r, --rationale <rationale>', 'Decision rationale')
  .option('-a, --alternatives <alternatives...>', 'Alternative options considered')
  .action(async (decision, options) => {
    await manager.logDecision(decision, options.rationale, options.alternatives);
  });

// Log risk
program
  .command('log-risk <risk>')
  .description('Log a project risk')
  .requiredOption('-i, --impact <impact>', 'Impact (low|medium|high)')
  .requiredOption('-l, --likelihood <likelihood>', 'Likelihood (low|medium|high)')
  .option('-m, --mitigation <mitigation>', 'Mitigation strategy')
  .action(async (risk, options) => {
    await manager.logRisk(
      risk,
      options.impact as 'low' | 'medium' | 'high',
      options.likelihood as 'low' | 'medium' | 'high',
      options.mitigation
    );
  });

// Show status
program
  .command('status')
  .description('Show status of all context files')
  .action(async () => {
    await manager.showStatus();
  });

// Validate all files
program
  .command('validate')
  .description('Validate all context files against standards')
  .action(async () => {
    await manager.validateAll();
  });

// Parse arguments
program.parse(process.argv);

// Show help if no command provided
if (!process.argv.slice(2).length) {
  program.outputHelp();
}