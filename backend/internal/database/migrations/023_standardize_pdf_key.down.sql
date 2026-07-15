-- Down migration: Revert pdf_key -> pdf_path

ALTER TABLE resumes RENAME COLUMN pdf_key TO pdf_path;
