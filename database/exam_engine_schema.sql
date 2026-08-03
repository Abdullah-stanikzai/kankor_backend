-- Exam Engine Database Schema
-- Run this to create all necessary tables for the exam engine

-- ========================================
-- 1. QUESTIONS TABLE (Already exists, but ensuring structure)
-- ========================================
-- This table already exists from previous work, but let's verify the structure

-- If questions table doesn't exist, create it:
CREATE TABLE IF NOT EXISTS questions (
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

-- Index for fast question lookup by exam
CREATE INDEX IF NOT EXISTS idx_questions_exam_id ON questions(exam_id);

-- ========================================
-- 2. ANSWERS TABLE (NEW - For storing student answers)
-- ========================================
-- Note: Removed foreign key to students table - using users table instead
CREATE TABLE IF NOT EXISTS answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL, -- References users(id) from your auth system
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    selected_option CHAR(1) NOT NULL CHECK (selected_option IN ('A', 'B', 'C', 'D')),
    is_correct BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure one answer per question per student
    UNIQUE(student_id, question_id)
);

-- Indexes for fast lookup
CREATE INDEX IF NOT EXISTS idx_answers_student_id ON answers(student_id);
CREATE INDEX IF NOT EXISTS idx_answers_exam_id ON answers(exam_id);
CREATE INDEX IF NOT EXISTS idx_answers_question_id ON answers(question_id);
CREATE INDEX IF NOT EXISTS idx_answers_student_exam ON answers(student_id, exam_id);

-- ========================================
-- 3. EXAM RESULTS TABLE (NEW - For storing exam results)
-- ========================================
-- Note: Removed foreign key to students table - using users table instead
CREATE TABLE IF NOT EXISTS exam_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL, -- References users(id) from your auth system
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    score DECIMAL(5,2) NOT NULL, -- Percentage score (0-100)
    total_questions INTEGER NOT NULL,
    correct_answers INTEGER NOT NULL,
    wrong_answers INTEGER NOT NULL,
    unanswered_questions INTEGER NOT NULL,
    time_spent_seconds INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure one result per student per exam
    UNIQUE(student_id, exam_id)
);

-- Indexes for fast lookup
CREATE INDEX IF NOT EXISTS idx_exam_results_student_id ON exam_results(student_id);
CREATE INDEX IF NOT EXISTS idx_exam_results_exam_id ON exam_results(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_results_created_at ON exam_results(created_at);

-- ========================================
-- 4. HELPFUL VIEWS
-- ========================================

-- View: Student exam summary with results
CREATE OR REPLACE VIEW student_exam_summary AS
SELECT 
    e.id as exam_id,
    e.title as exam_title,
    e.start_time,
    e.end_time,
    e.is_published,
    e.is_active,
    COUNT(DISTINCT q.id) as total_questions,
    COUNT(DISTINCT a.id) as answered_questions,
    er.score,
    er.correct_answers,
    er.wrong_answers,
    er.created_at as result_date
FROM exams e
LEFT JOIN questions q ON e.id = q.exam_id
LEFT JOIN answers a ON e.id = a.exam_id
LEFT JOIN exam_results er ON e.id = er.exam_id
GROUP BY e.id, er.score, er.correct_answers, er.wrong_answers, er.created_at;

-- ========================================
-- 5. TRIGGERS FOR AUTOMATIC UPDATES
-- ========================================

-- Trigger to update updated_at on answers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_answers_updated_at BEFORE UPDATE ON answers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- 6. SAMPLE DATA FOR TESTING (OPTIONAL)
-- ========================================

-- Uncomment to add test data
/*
INSERT INTO questions (exam_id, text, option_a, option_b, option_c, option_d, correct_option)
VALUES 
('YOUR_EXAM_ID_HERE', 'What is 2 + 2?', '3', '4', '5', '6', 'B'),
('YOUR_EXAM_ID_HERE', 'What is the capital of Afghanistan?', 'Kandahar', 'Herat', 'Kabul', 'Mazar', 'C');
*/

-- ========================================
-- 7. VERIFICATION QUERIES
-- ========================================

-- Check if tables exist
SELECT 
    table_name,
    table_schema
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('questions', 'answers', 'exam_results')
ORDER BY table_name;

-- Show table structures
\d questions
\d answers
\d exam_results
