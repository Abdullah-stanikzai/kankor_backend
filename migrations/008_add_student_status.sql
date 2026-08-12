-- ============================================================
-- Migration 008: Add student activation status
-- Adds status column to users table with CHECK constraint
-- ============================================================

-- Add status column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'status'
    ) THEN
        ALTER TABLE users
        ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';

        -- Add CHECK constraint
        ALTER TABLE users
        ADD CONSTRAINT chk_users_status CHECK (status IN ('pending', 'active', 'inactive'));
    END IF;
END $$;

-- Create index for faster status lookups
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
