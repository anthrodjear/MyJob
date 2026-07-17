/**
 * Tests for config module (config/config.ts).
 *
 * Covers: ConfigError, Zod schema validation, loadConfig, clearConfigCache.
 * Mocks: fs.readFileSync for file system access.
 */

import * as fs from 'fs';
import { ConfigError, loadConfig, clearConfigCache } from '../config';

// Mock fs.readFileSync
jest.mock('fs');
const mockReadFileSync = fs.readFileSync as jest.MockedFunction<typeof fs.readFileSync>;

// Minimal valid config for testing
const VALID_CONFIG = `
application:
  approval_tiers:
    auto_apply:
      min_score: 95
      action: auto_submit
      notify: true
    review:
      min_score: 80
      max_score: 94
      action: queue_for_review
    reject:
      max_score: 79
      action: auto_reject
      log: true
  auto_generate:
    resume: true
    cover_letter: true
  resume:
    engine: latex
    template_dir: templates/resume
  cover_letter:
    engine: latex
    template_dir: templates/cover_letter
    max_length: 1000
queue:
  redis_url: redis://localhost:6379
  concurrency: 5
  retryAttempts: 3
llm:
  primary:
    provider: openai
    model: gpt-4
  local:
    provider: ollama
    model: llama2
    baseUrl: http://localhost:11434
voice:
  provider: openai
  model: gpt-4
  livekit:
    url: ws://localhost:7880
    api_key: devkey
    api_secret: devsecret
interview:
  memory: {}
  retriever: {}
  responder: {}
  planner: {}
email:
  provider: microsoft
  check_interval: 30m
  folders:
    - Inbox
prompts:
  scoring:
    system: Score this job
    user: Rate 0-100
  email_classifier:
    system: Classify email
    user: What type?
  cover_letter:
    system: Write cover letter
    user: Write for this job
  resume_tailor:
    system: Tailor resume
    user: Tailor for this job
  interview_prep:
    system: Prepare for interview
    user: Prep questions
  job_extraction:
    system: Extract job info
    user: Extract from page
  form_understanding:
    system: Understand form
    user: Map fields
  resume_generation:
    system: Generate resume
    user: Generate for this job
`;

describe('ConfigError', () => {
  it('stores message and cause', () => {
    const cause = new Error('underlying');
    const err = new ConfigError('config failed', cause);
    expect(err.message).toBe('config failed');
    expect(err.cause).toBe(cause);
    expect(err.name).toBe('ConfigError');
  });

  it('is an instance of Error', () => {
    const err = new ConfigError('test');
    expect(err).toBeInstanceOf(Error);
  });

  it('cause is optional', () => {
    const err = new ConfigError('test');
    expect(err.cause).toBeUndefined();
  });
});

describe('loadConfig', () => {
  beforeEach(() => {
    clearConfigCache();
    mockReadFileSync.mockReset();
  });

  it('loads and validates a valid config', () => {
    mockReadFileSync.mockReturnValue(VALID_CONFIG);
    const config = loadConfig('/fake/path/application.yaml');
    expect(config.application).toBeDefined();
    expect(config.queue).toBeDefined();
    expect(config.llm).toBeDefined();
    expect(config.voice).toBeDefined();
    expect(config.prompts).toBeDefined();
  });

  it('caches config after first load', () => {
    mockReadFileSync.mockReturnValue(VALID_CONFIG);
    const config1 = loadConfig('/fake/path/application.yaml');
    const config2 = loadConfig('/fake/path/application.yaml');
    expect(config1).toBe(config2); // Same reference = cached
    expect(mockReadFileSync).toHaveBeenCalledTimes(1); // Only read once
  });

  it('throws ConfigError for missing file', () => {
    const enoent = Object.assign(new Error('ENOENT'), { code: 'ENOENT' });
    mockReadFileSync.mockImplementation(() => { throw enoent; });
    
    expect(() => loadConfig('/nonexistent.yaml')).toThrow(ConfigError);
    expect(() => loadConfig('/nonexistent.yaml')).toThrow('Config file not found');
  });

  it('throws ConfigError for file read error', () => {
    mockReadFileSync.mockImplementation(() => { throw new Error('permission denied'); });
    
    expect(() => loadConfig('/fake/path.yaml')).toThrow(ConfigError);
    expect(() => loadConfig('/fake/path.yaml')).toThrow('Failed to read config file');
  });

  it('throws ConfigError for invalid YAML', () => {
    mockReadFileSync.mockReturnValue('{{invalid yaml::');
    
    expect(() => loadConfig('/fake/path.yaml')).toThrow(ConfigError);
    expect(() => loadConfig('/fake/path.yaml')).toThrow('Invalid YAML');
  });

  it('throws ConfigError for schema validation failure', () => {
    const invalidConfig = `
application:
  approval_tiers:
    auto_apply:
      min_score: not-a-number
      action: auto_submit
      notify: true
queue:
  redis_url: redis://localhost:6379
  concurrency: 5
  retryAttempts: 3
`;
    mockReadFileSync.mockReturnValue(invalidConfig);
    
    expect(() => loadConfig('/fake/path.yaml')).toThrow(ConfigError);
    expect(() => loadConfig('/fake/path.yaml')).toThrow('Config validation failed');
  });

  it('clears cache when clearConfigCache is called', () => {
    mockReadFileSync.mockReturnValue(VALID_CONFIG);
    loadConfig('/fake/path/application.yaml');
    clearConfigCache();
    loadConfig('/fake/path/application.yaml');
    expect(mockReadFileSync).toHaveBeenCalledTimes(2);
  });
});
