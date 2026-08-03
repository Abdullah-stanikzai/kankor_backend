-- Check and Fix Student Center Enrollment
-- This will verify and create the missing enrollment record

-- 1. Check if student exists
SELECT id, email FROM students WHERE id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e';

-- 2. Check current enrollments for this student
SELECT 
    sce.id,
    sce.student_id,
    sce.center_id,
    sce.is_active,
    sce.enrollment_date,
    c.name as center_name
FROM student_center_enrollment sce
LEFT JOIN centers c ON sce.center_id = c.id
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e';

-- 3. Check what exams exist in the database for this center
SELECT 
    e.id,
    e.title,
    e.description,
    e.center_id,
    e.is_published,
    e.is_active,
    e.start_time,
    e.created_at
FROM exams e
WHERE e.center_id = 'c2cbdf4d-54c5-4dc3-85c1-a1e04b686723'
ORDER BY e.created_at DESC;

-- 4. Create the missing enrollment (RUN THIS IF STEP 2 RETURNS EMPTY)
INSERT INTO student_center_enrollment 
(student_id, center_id, enrollment_date, is_active, created_at)
VALUES 
('8d72c1b6-675a-4a64-bbce-bbe433a81f3e', 'c2cbdf4d-54c5-4dc3-85c1-a1e04b686723', NOW(), true, NOW());

-- 5. Verify the enrollment was created
SELECT 
    sce.id,
    sce.student_id,
    sce.center_id,
    sce.is_active,
    c.name as center_name
FROM student_center_enrollment sce
LEFT JOIN centers c ON sce.center_id = c.id
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e';

-- 6. Test the exact query that the backend uses
SELECT e.id, e.title, e.description, e.center_id, e.is_published, e.is_active
FROM exams e 
JOIN student_center_enrollment sce ON e.center_id = sce.center_id 
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e' 
  AND sce.is_active = true;

-- This should return the exams if enrollment exists
