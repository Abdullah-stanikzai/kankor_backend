-- ============================================================
-- Migration 007: Student auth fields
-- Adds avatar column and phone unique constraint to users table
-- ============================================================

-- Add avatar column for student profile pictures
ALTER TABLE users
ADD COLUMN IF NOT EXISTS avatar VARCHAR(500);

-- Add unique constraint on phone_number (skip if already exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_users_phone_number'
          AND conrelid = 'users'::regclass
    ) THEN
        -- First, remove any duplicate phone numbers (set to NULL)
        UPDATE users
        SET phone_number = NULL
        WHERE phone_number IN (
            SELECT phone_number
            FROM users
            WHERE phone_number IS NOT NULL
            GROUP BY phone_number
            HAVING COUNT(*) > 1
        );
        
        -- Then add unique constraint
        ALTER TABLE users
        ADD CONSTRAINT uq_users_phone_number UNIQUE (phone_number);
    END IF;
END $$;

-- Add index for faster phone lookups
CREATE INDEX IF NOT EXISTS idx_users_phone_number ON users(phone_number);
