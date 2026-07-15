-- Standardize: resumes.pdf_path -> pdf_key (matching cover_letters naming)

ALTER TABLE resumes RENAME COLUMN pdf_path TO pdf_key;
