-- Fix for questions table ID generation issue
-- This ensures the id column has DEFAULT gen_random_uuid()

-- Drop the existing table if it exists
DROP TABLE IF EXISTS questions CASCADE;

-- Recreate with proper structure
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    option_a TEXT NOT NULL,
    option_b TEXT NOT NULL,
    option_c TEXT NOT NULL,
    option_d TEXT NOT NULL,
    correct_option CHAR(1) NOT NULL CHECK (correct_option IN ('A', 'B', 'C', 'D')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for fast question lookup by exam
CREATE INDEX idx_questions_exam_id ON questions(exam_id);

-- Verify the table structure
\d questions

-- Test insert to verify id generation works
-- Uncomment to test:
/*
INSERT INTO questions (exam_id, text, option_a, option_b, option_c, option_d, correct_option)
VALUES ('00000000-0000-0000-0000-000000000000', 'Test question', 'Option A', 'Option B', 'Option C', 'Option D', 'B');

SELECT * FROM questions LIMIT 1;
*/
