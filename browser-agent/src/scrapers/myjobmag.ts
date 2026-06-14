import { BaseScraper } from './base';

export class MyJobMagScraper extends BaseScraper {
  async scrape(url: string): Promise<any> {
    throw new Error('Not implemented');
  }
}
