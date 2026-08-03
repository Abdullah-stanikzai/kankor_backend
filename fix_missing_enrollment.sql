-- Quick Fix: Create Missing Student Enrollment
-- Run this in your PostgreSQL database to ensure student has enrollment

-- IMPORTANT: Replace the UUIDs below with your actual student and center IDs if different
-- From your JWT token:
-- Student ID: 8d72c1b6-675a-4a64-bbce-bbe433a81f3e
-- Center ID: c2cbdf4d-54c5-4dc3-85c1-a1e04b686723

-- Check if enrollment exists first
SELECT 
    sce.id,
    sce.student_id,
    sce.center_id,
    sce.is_active
FROM student_center_enrollment sce
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e';

-- If the above returns empty, run this INSERT:
INSERT INTO student_center_enrollment 
(student_id, center_id, enrollment_date, is_active, created_at)
SELECT '8d72c1b6-675a-4a64-bbce-bbe433a81f3e', 'c2cbdf4d-54c5-4dc3-85c1-a1e04b686723', NOW(), true, NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM student_center_enrollment 
    WHERE student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e'
);

-- Verify it was created
SELECT 
    sce.id,
    sce.student_id,
    sce.center_id,
    sce.is_active,
    c.name as center_name
FROM student_center_enrollment sce
LEFT JOIN centers c ON sce.center_id = c.id
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e';

-- Test that the student can now see exams
SELECT e.id, e.title, e.center_id
FROM exams e 
JOIN student_center_enrollment sce ON e.center_id = sce.center_id 
WHERE sce.student_id = '8d72c1b6-675a-4a64-bbce-bbe433a81f3e' 
  AND sce.is_active = true;

-- This should return your exam "دنیات"
