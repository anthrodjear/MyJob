-- Drop tables in reverse order
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS embeddings;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS emails;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS cover_letters;
DROP TABLE IF EXISTS resumes;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS job_sources;
DROP TABLE IF EXISTS profiles;

-- Drop extensions
DROP EXTENSION IF EXISTS vector;
DROP EXTENSION IF EXISTS "uuid-ossp";
