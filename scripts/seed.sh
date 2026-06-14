#!/bin/bash
set -e

echo "Seeding initial data..."

# Insert default job sources
docker compose exec postgres psql -U myjob -d myjob -c "
INSERT INTO job_sources (name, base_url, source_type, config) VALUES
('indeed', 'https://www.indeed.com', 'indeed', '{\"rate_limit\": {\"requests_per_second\": 0.5}}'),
('remoteok', 'https://remoteok.com', 'remoteok', '{\"rate_limit\": {\"requests_per_second\": 0.5}}')
ON CONFLICT DO NOTHING;
"

echo "Seed complete!"
