import { BaseScraper } from './base';

export class GreenhouseScraper extends BaseScraper {
  async scrape(url: string): Promise<any> {
    throw new Error('Not implemented');
  }
}
