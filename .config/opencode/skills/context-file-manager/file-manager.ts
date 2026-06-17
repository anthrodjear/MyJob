/**
 * File Manager
 * 
 * Handles file operations for context files
 */

import * as fs from 'fs/promises';
import * as path from 'path';

export class FileManager {
  private baseDir: string;

  constructor(baseDir: string) {
    this.baseDir = baseDir;
  }

  /**
   * Check if file exists
   */
  async exists(filePath: string): Promise<boolean> {
    try {
      const fullPath = path.resolve(this.baseDir, filePath);
      await fs.access(fullPath);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Read file content
   */
  async read(filePath: string): Promise<string> {
    const fullPath = path.resolve(this.baseDir, filePath);
    return await fs.readFile(fullPath, 'utf-8');
  }

  /**
   * Write file content
   */
  async write(filePath: string, content: string): Promise<void> {
    const fullPath = path.resolve(this.baseDir, filePath);
    const dir = path.dirname(fullPath);
    
    // Ensure directory exists
    await fs.mkdir(dir, { recursive: true });
    
    await fs.writeFile(fullPath, content, 'utf-8');
  }

  /**
   * List files in directory
   */
  async list(dirPath: string = '.'): Promise<string[]> {
    const fullPath = path.resolve(this.baseDir, dirPath);
    const files = await fs.readdir(fullPath, { withFileTypes: true });
    
    return files
      .filter(f => f.isFile())
      .map(f => f.name);
  }

  /**
   * Get file stats
   */
  async stat(filePath: string): Promise<fs.Stats | null> {
    try {
      const fullPath = path.resolve(this.baseDir, filePath);
      return await fs.stat(fullPath);
    } catch {
      return null;
    }
  }

  /**
   * Delete file
   */
  async delete(filePath: string): Promise<void> {
    const fullPath = path.resolve(this.baseDir, filePath);
    await fs.unlink(fullPath);
  }

  /**
   * Copy file
   */
  async copy(sourcePath: string, destPath: string): Promise<void> {
    const source = path.resolve(this.baseDir, sourcePath);
    const dest = path.resolve(this.baseDir, destPath);
    
    const destDir = path.dirname(dest);
    await fs.mkdir(destDir, { recursive: true });
    
    await fs.copyFile(source, dest);
  }
}

export default FileManager;