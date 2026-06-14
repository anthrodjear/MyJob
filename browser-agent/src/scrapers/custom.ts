import { BaseScraper } from './base';

export class CustomScraper extends BaseScraper {
  async scrape(url: string): Promise<any> {
    throw new Error('Not implemented');
  }
}
