-- Migration: Create question_bank table for AI-powered question extraction
-- This table stores questions extracted from PDFs using Gemini AI

CREATE TABLE question_bank (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    center_id UUID REFERENCES educational_centers(id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL,
    question_text TEXT NOT NULL,
    option_a TEXT NOT NULL,
    option_b TEXT NOT NULL,
    option_c TEXT NOT NULL,
    option_d TEXT NOT NULL,
    correct_option CHAR(1), -- 'A', 'B', 'C', or 'D'
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (correct_option IN ('A', 'B', 'C', 'D', NULL))
);

-- Create indexes for faster lookups
CREATE INDEX idx_question_bank_center_id ON question_bank(center_id);
CREATE INDEX idx_question_bank_subject ON question_bank(subject);
CREATE INDEX idx_question_bank_created_by ON question_bank(created_by);

-- Add comment for documentation
COMMENT ON TABLE question_bank IS 'Stores questions extracted from PDFs using AI for reuse in exams';
