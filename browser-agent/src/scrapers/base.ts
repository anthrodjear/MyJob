export abstract class BaseScraper {
  abstract scrape(url: string): Promise<any>;
}
