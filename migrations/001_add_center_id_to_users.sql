-- Migration: Add center_id column to users table
-- This allows center admins to have a direct reference to their assigned center

ALTER TABLE users ADD COLUMN center_id UUID REFERENCES educational_centers(id) ON DELETE SET NULL;

-- Create index for faster lookups
CREATE INDEX idx_users_center_id ON users(center_id);

-- Update existing center admins to have their center_id set
-- This assumes that center admins are already assigned to centers via educational_centers.admin_user_id
UPDATE users u
SET center_id = ec.id
FROM educational_centers ec
WHERE ec.admin_user_id = u.id AND u.role = 'center_admin';
