-- Migration to add scoring to questions
-- Add score column to questions table

ALTER TABLE questions 
ADD COLUMN score DECIMAL(5,2) DEFAULT 1.00 NOT NULL;

-- Add comment to explain the purpose
COMMENT ON COLUMN questions.score IS 'Points awarded for correctly answering this question';