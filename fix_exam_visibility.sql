-- Fix Exam Visibility Issues for Students
-- Run this to ensure exams are properly configured for student access

-- 1. Check current exam status
SELECT 
    id, 
    title, 
    center_id, 
    creator_id,
    is_published, 
    is_active, 
    start_time, 
    end_time,
    created_at
FROM exams
ORDER BY created_at DESC;

-- 2. Update all exams to be published (if they should be visible to students)
-- Uncomment the next line if you want to publish all existing exams
-- UPDATE exams SET is_published = true WHERE is_published = false;

-- 3. Ensure all exams are active
-- Uncomment the next line if you want to activate all existing exams
-- UPDATE exams SET is_active = true WHERE is_active = false;

-- 4. Check student center enrollments
SELECT 
    sce.id,
    sce.student_id,
    sce.center_id,
    sce.is_active,
    s.email as student_email,
    c.name as center_name
FROM student_center_enrollment sce
JOIN students s ON sce.student_id = s.id
JOIN centers c ON sce.center_id = c.id
WHERE sce.is_active = true
ORDER BY sce.created_at DESC;

-- 5. Verify exam and question relationship
SELECT 
    e.id as exam_id,
    e.title as exam_title,
    COUNT(q.id) as question_count
FROM exams e
LEFT JOIN questions q ON e.id = q.exam_id
GROUP BY e.id, e.title
ORDER BY e.created_at DESC;

-- 6. DEBUG: Create a test enrollment if none exists
-- Replace 'student-uuid-here' with actual student ID and 'center-uuid-here' with actual center ID
-- INSERT INTO student_center_enrollment (student_id, center_id, enrollment_date, is_active, created_at)
-- VALUES ('student-uuid-here', 'center-uuid-here', NOW(), true, NOW());

-- 7. Verify that the student has access to exams through their center
SELECT 
    s.id as student_id,
    s.email as student_email,
    e.id as exam_id,
    e.title as exam_title,
    e.is_published,
    e.is_active,
    e.start_time
FROM students s
JOIN student_center_enrollment sce ON s.id = sce.student_id
JOIN exams e ON sce.center_id = e.center_id
WHERE sce.is_active = true
ORDER BY e.start_time DESC;
