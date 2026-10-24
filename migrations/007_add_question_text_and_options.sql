-- ============================================================
-- 007: Production schema sync for normalized question reading
-- ============================================================
-- The backend commits 668bf69 / b1dbf5e / eabac08 read
-- `questions.question_text` and the `question_options` table.
-- Older production databases were created before these existed,
-- causing: ERROR: column "question_text" does not exist (SQLSTATE 42703)
--
-- This script is idempotent — safe to run more than once.
-- ============================================================

-- 1) questions.question_text (nullable; the code falls back to the
--    legacy 'text' column whenever question_text is NULL or empty)
ALTER TABLE questions ADD COLUMN IF NOT EXISTS question_text TEXT;

-- 2) Normalized options table (the admin panel writes here and the
--    exam engine / review endpoints read from here)
CREATE TABLE IF NOT EXISTS question_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    option_letter CHAR(1) NOT NULL CHECK (option_letter IN ('A', 'B', 'C', 'D')),
    is_correct BOOLEAN DEFAULT FALSE,
    option_order INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_question_options_question_id
    ON question_options(question_id);

-- 3) No data backfill needed:
--    - old questions keep question_text = NULL and the code reads 'text'
--    - options fall back to the flat option_a..d columns when
--      question_options has no rows for a question
