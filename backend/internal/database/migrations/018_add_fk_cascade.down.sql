-- Down migration: Revert to original FK constraints (ON DELETE NO ACTION)

DO $$
BEGIN
    -- resume_id FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'applications' 
        AND constraint_name = 'applications_resume_id_fkey'
    ) THEN
        ALTER TABLE applications DROP CONSTRAINT applications_resume_id_fkey;
    END IF;
    
    ALTER TABLE applications 
    ADD CONSTRAINT applications_resume_id_fkey 
    FOREIGN KEY (resume_id) REFERENCES resumes(id);
    
    -- cover_letter_id FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'applications' 
        AND constraint_name = 'applications_cover_letter_id_fkey'
    ) THEN
        ALTER TABLE applications DROP CONSTRAINT applications_cover_letter_id_fkey;
    END IF;
    
    ALTER TABLE applications 
    ADD CONSTRAINT applications_cover_letter_id_fkey 
    FOREIGN KEY (cover_letter_id) REFERENCES cover_letters(id);
END $$;