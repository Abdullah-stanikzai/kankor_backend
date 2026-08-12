-- Migration 007: Add class column to exams table
-- Purpose: Support class/grade selection for exams (10, 11, 12)

ALTER TABLE exams ADD COLUMN IF NOT EXISTS class VARCHAR(5) DEFAULT '' NOT NULL;

-- Add index for class-based queries
CREATE INDEX IF NOT EXISTS idx_exams_class ON exams(class);

-- Update existing exams: set default class to empty string (already done by DEFAULT)
-- The admin can update existing exams to assign their correct class
