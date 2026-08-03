-- Migration: Replace UUID center_id with auto-incrementing integer ID
-- This migration will:
-- 1. Create a new id column with auto-increment
-- 2. Migrate data from UUID to integer IDs
-- 3. Update foreign keys
-- 4. Drop the old UUID id column

-- Step 1: Add a new serial id column to educational_centers
ALTER TABLE educational_centers ADD COLUMN id_new SERIAL;

-- Step 2: Create a mapping table to track old UUID to new integer ID
CREATE TEMP TABLE center_id_mapping AS
SELECT id as old_id, id_new as new_id FROM educational_centers;

-- Step 3: Update users table to use new integer center_id
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_center_id_fkey;
ALTER TABLE users ALTER COLUMN center_id TYPE INTEGER;

-- Step 4: Update student_center_enrollment to use new integer center_id
ALTER TABLE student_center_enrollment DROP CONSTRAINT IF EXISTS student_center_enrollment_center_id_fkey;
ALTER TABLE student_center_enrollment ALTER COLUMN center_id TYPE INTEGER;

-- Step 5: Update exams to use new integer center_id
ALTER TABLE exams DROP CONSTRAINT IF EXISTS exams_center_id_fkey;
ALTER TABLE exams ALTER COLUMN center_id TYPE INTEGER;

-- Step 6: Update educational_centers table
-- Drop old UUID id and rename new id
ALTER TABLE educational_centers DROP CONSTRAINT educational_centers_pkey;
ALTER TABLE educational_centers DROP COLUMN id;
ALTER TABLE educational_centers RENAME COLUMN id_new TO id;
ALTER TABLE educational_centers ADD PRIMARY KEY (id);

-- Step 7: Re-add foreign key constraints
ALTER TABLE users ADD CONSTRAINT users_center_id_fkey 
  FOREIGN KEY (center_id) REFERENCES educational_centers(id) ON DELETE SET NULL;

ALTER TABLE student_center_enrollment ADD CONSTRAINT student_center_enrollment_center_id_fkey 
  FOREIGN KEY (center_id) REFERENCES educational_centers(id) ON DELETE CASCADE;

ALTER TABLE exams ADD CONSTRAINT exams_center_id_fkey 
  FOREIGN KEY (center_id) REFERENCES educational_centers(id) ON DELETE CASCADE;

-- Step 8: Update admin_user_id foreign key
ALTER TABLE educational_centers DROP CONSTRAINT IF EXISTS educational_centers_admin_user_id_fkey;
ALTER TABLE educational_centers ADD CONSTRAINT educational_centers_admin_user_id_fkey 
  FOREIGN KEY (admin_user_id) REFERENCES users(id) ON DELETE SET NULL;

-- Step 9: Recreate indexes
DROP INDEX IF EXISTS idx_centers_admin_user;
DROP INDEX IF EXISTS idx_centers_is_active;
DROP INDEX IF EXISTS idx_student_center_center;
DROP INDEX IF EXISTS idx_exams_center;

CREATE INDEX idx_centers_admin_user ON educational_centers(admin_user_id);
CREATE INDEX idx_centers_is_active ON educational_centers(is_active);
CREATE INDEX idx_student_center_center ON student_center_enrollment(center_id);
CREATE INDEX idx_exams_center ON exams(center_id);
