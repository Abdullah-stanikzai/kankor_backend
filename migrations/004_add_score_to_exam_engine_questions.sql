-- Migration to add score column to questions table for exam engine
-- This adds scoring support to the exam engine questions table

-- Add score column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'questions' AND column_name = 'score'
    ) THEN
        ALTER TABLE questions 
        ADD COLUMN score DECIMAL(5,2) DEFAULT 2.50 NOT NULL;
        
        RAISE NOTICE 'Added score column to questions table';
    ELSE
        RAISE NOTICE 'Score column already exists in questions table';
    END IF;
END $$;

-- Update existing questions to have a default score of 2.5 if they have NULL
UPDATE questions 
SET score = 2.50 
WHERE score IS NULL;

-- Add comment to explain the purpose
COMMENT ON COLUMN questions.score IS 'Points awarded for correctly answering this question (default: 2.50)';

-- Verify the column was added
SELECT column_name, data_type, column_default, is_nullable
FROM information_schema.columns
WHERE table_name = 'questions' AND column_name = 'score';
