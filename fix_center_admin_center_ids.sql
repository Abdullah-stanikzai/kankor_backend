-- Fix center_id for existing center_admin users
-- This script sets the center_id for center_admins based on their assignment in the educational_centers table.

UPDATE users
SET center_id = ec.id
FROM educational_centers ec
WHERE users.role = 'center_admin'
  AND users.center_id IS NULL
  AND ec.admin_user_id = users.id;
