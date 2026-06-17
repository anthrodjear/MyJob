/**
 * Context7 MCP Client
 * 
 * Handles communication with Context7 MCP server for design guidance
 */

export interface Context7Library {
  id: string;
  name: string;
  description: string;
  codeSnippets: number;
  sourceReputation: 'High' | 'Medium' | 'Low' | 'Unknown';
  benchmarkScore: number;
  versions: string[];
}

export interface Context7QueryResult {
  content: string;
  source: string;
  confidence: number;
}

export class Context7Client {
  private baseUrl: string;
  private apiKey?: string;

  constructor(baseUrl: string = 'http://localhost:3001', apiKey?: string) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
  }

  /**
   * Resolve library ID from name
   */
  async resolveLibraryId(query: string, libraryName: string): Promise<Context7Library[]> {
    try {
      const response = await fetch(`${this.baseUrl}/resolve-library-id`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.apiKey && { 'Authorization': `Bearer ${this.apiKey}` })
        },
        body: JSON.stringify({ query, libraryName })
      });

      if (!response.ok) {
        throw new Error(`Failed to resolve library: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.warn(`Context7 resolve failed: ${error}`);
      return [];
    }
  }

  /**
   * Query documentation from Context7
   */
  async queryDocs(libraryId: string, query: string): Promise<string> {
    try {
      const response = await fetch(`${this.baseUrl}/query-docs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.apiKey && { 'Authorization': `Bearer ${this.apiKey}` })
        },
        body: JSON.stringify({ libraryId, query })
      });

      if (!response.ok) {
        throw new Error(`Failed to query docs: ${response.statusText}`);
      }

      const result = await response.json();
      return result.content || '';
    } catch (error) {
      console.warn(`Context7 query failed: ${error}`);
      return '';
    }
  }

  /**
   * Get best matching library for a query
   */
  async getBestLibrary(query: string, libraryName: string): Promise<string | null> {
    const libraries = await this.resolveLibraryId(query, libraryName);
    
    if (libraries.length === 0) {
      return null;
    }

    // Sort by benchmark score and source reputation
    const sorted = libraries.sort((a, b) => {
      const reputationOrder = { 'High': 3, 'Medium': 2, 'Low': 1, 'Unknown': 0 };
      const repDiff = reputationOrder[b.sourceReputation] - reputationOrder[a.sourceReputation];
      if (repDiff !== 0) return repDiff;
      return b.benchmarkScore - a.benchmarkScore;
    });

    return sorted[0].id;
  }
}

export default Context7Client;