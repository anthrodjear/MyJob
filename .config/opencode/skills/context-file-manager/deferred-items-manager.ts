/**
 * Deferred Items Manager
 * 
 * Manages deferred work items
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import { FileManager } from './file-manager';

export interface DeferredItem {
  id: string;
  description: string;
  priority: 'low' | 'medium' | 'high';
  status: 'PENDING' | 'IN_PROGRESS' | 'RESOLVED' | 'CANCELLED';
  context?: string;
  addedAt: string;
  resolvedAt?: string;
  resolution?: string;
}

export class DeferredItemsManager {
  private fileManager: FileManager;
  private fileName: string;

  constructor(outputDir: string) {
    this.fileManager = new FileManager(outputDir);
    this.fileName = 'deferred-items.md';
  }

  /**
   * Add deferred item
   */
  async add(item: Omit<DeferredItem, 'id' | 'addedAt' | 'status'>): Promise<DeferredItem> {
    const items = await this.read();
    
    const newItem: DeferredItem = {
      ...item,
      id: this.generateId(),
      status: 'PENDING',
      addedAt: new Date().toISOString()
    };

    items.push(newItem);
    await this.write(items);
    
    return newItem;
  }

  /**
   * Resolve deferred item
   */
  async resolve(itemId: string, resolution: string): Promise<void> {
    const items = await this.read();
    const item = items.find(i => i.id === itemId);
    
    if (item) {
      item.status = 'RESOLVED';
      item.resolvedAt = new Date().toISOString();
      item.resolution = resolution;
      await this.write(items);
    }
  }

  /**
   * Update item status
   */
  async updateStatus(itemId: string, status: DeferredItem['status']): Promise<void> {
    const items = await this.read();
    const item = items.find(i => i.id === itemId);
    
    if (item) {
      item.status = status;
      await this.write(items);
    }
  }

  /**
   * Get all deferred items
   */
  async getAll(): Promise<DeferredItem[]> {
    return await this.read();
  }

  /**
   * Get pending items
   */
  async getPending(): Promise<DeferredItem[]> {
    const items = await this.read();
    return items.filter(i => i.status === 'PENDING');
  }

  /**
   * Read deferred items from file
   */
  private async read(): Promise<DeferredItem[]> {
    const exists = await this.fileManager.exists(this.fileName);
    
    if (!exists) {
      return [];
    }

    const content = await this.fileManager.read(this.fileName);
    return this.parseItems(content);
  }

  /**
   * Write deferred items to file
   */
  private async write(items: DeferredItem[]): Promise<void> {
    const content = this.formatItems(items);
    await this.fileManager.write(this.fileName, content);
  }

  /**
   * Format items as markdown table
   */
  private formatItems(items: DeferredItem[]): string {
    let content = `# Deferred Items\n\n`;
    content += `> Track deferred work items. Update status as items are resolved.\n\n`;
    content += `---\n\n`;
    content += `## Items\n\n`;
    content += `| ID | Description | Priority | Status | Added | Context |\n`;
    content += `|----|-------------|----------|--------|-------|---------|\n`;
    
    for (const item of items) {
      const priorityIcon = item.priority === 'high' ? '🔴' : 
                          item.priority === 'medium' ? '🟡' : '🟢';
      const statusIcon = item.status === 'PENDING' ? '⏳' :
                        item.status === 'IN_PROGRESS' ? '🔄' :
                        item.status === 'RESOLVED' ? '✅' : '❌';
      
      content += `| ${item.id.slice(0, 8)} | ${item.description} | ${priorityIcon} ${item.priority} | ${statusIcon} ${item.status} | ${item.addedAt.split('T')[0]} | ${item.context || '-'} |\n`;
    }
    
    content += `\n---\n\n`;
    content += `## Resolved Items\n\n`;
    
    const resolved = items.filter(i => i.status === 'RESOLVED');
    if (resolved.length > 0) {
      content += `| ID | Description | Resolved | Resolution |\n`;
      content += `|----|-------------|----------|------------|\n`;
      
      for (const item of resolved) {
        content += `| ${item.id.slice(0, 8)} | ${item.description} | ${item.resolvedAt?.split('T')[0]} | ${item.resolution} |\n`;
      }
    } else {
      content += `No resolved items yet.\n`;
    }
    
    content += `\n---\n\n`;
    content += `*Auto-updated. Add items with \`context-file-manager add-deferred\`*\n`;
    
    return content;
  }

  /**
   * Parse items from markdown
   */
  private parseItems(content: string): DeferredItem[] {
    // In reality would parse the markdown table
    // For now return empty array
    return [];
  }

  /**
   * Generate unique ID
   */
  private generateId(): string {
    return `deferred-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

export default DeferredItemsManager;