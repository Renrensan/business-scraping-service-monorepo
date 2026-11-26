-- Migration 006: API keys used to be copied into scrape_jobs.metadata in plain text. Remove them.
UPDATE scrape_jobs SET metadata = metadata - 'api_key' WHERE metadata ? 'api_key';
