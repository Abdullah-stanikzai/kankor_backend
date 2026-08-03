-- Migration to add exam scheduling and status control fields
-- Adds status column and performance indexes

-- Add status column if not exists
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exams' AND column_name = 'status'
    ) THEN
        ALTER TABLE exams 
        ADD COLUMN status VARCHAR(20) DEFAULT 'pending' NOT NULL;
        
        -- Update existing exams to 'published' if is_published is true
        UPDATE exams SET status = 'published' WHERE is_published = true;
        
        RAISE NOTICE 'Added status column to exams table';
        RAISE NOTICE 'Migrated existing exams based on is_published flag';
    ELSE
        RAISE NOTICE 'Status column already exists';
    END IF;
END $$;

-- Add performance indexes
CREATE INDEX IF NOT EXISTS idx_exams_status ON exams(status);
CREATE INDEX IF NOT EXISTS idx_exams_start_time ON exams(start_time);
CREATE INDEX IF NOT EXISTS idx_exams_center_id ON exams(center_id);
CREATE INDEX IF NOT EXISTS idx_exams_status_start_time ON exams(status, start_time);

-- Verify the migration
SELECT 
    column_name, 
    data_type, 
    character_maximum_length,
    column_default, 
    is_nullable
FROM information_schema.columns
WHERE table_name = 'exams' 
  AND column_name IN ('status', 'start_time', 'center_id')
ORDER BY column_name;

-- Show indexes
SELECT 
    indexname, 
    indexdef
FROM pg_indexes
WHERE tablename = 'exams'
  AND indexname LIKE 'idx_exams_%'
ORDER BY indexname;
