-- ============================================
-- OPTIONAL: Create Students Table
-- ============================================
-- Run this ONLY if you want a dedicated students table
-- Otherwise, the exam engine uses the users table for student data

-- Check if students table exists
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'students') THEN
        -- Create students table
        CREATE TABLE students (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
            center_id UUID NOT NULL REFERENCES centers(id) ON DELETE CASCADE,
            student_number VARCHAR(50) UNIQUE,
            enrollment_year INTEGER,
            is_active BOOLEAN DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
        
        -- Create indexes
        CREATE INDEX idx_students_user_id ON students(user_id);
        CREATE INDEX idx_students_center_id ON students(center_id);
        CREATE INDEX idx_students_student_number ON students(student_number);
        
        RAISE NOTICE 'Students table created successfully';
    ELSE
        RAISE NOTICE 'Students table already exists, skipping creation';
    END IF;
END $$;

-- Add foreign key constraints to answers and exam_results if students table exists
DO $$ 
BEGIN 
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'students') THEN
        -- Add foreign key to answers table
        ALTER TABLE answers 
        ADD CONSTRAINT answers_student_id_fkey 
        FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
        
        -- Add foreign key to exam_results table
        ALTER TABLE exam_results 
        ADD CONSTRAINT exam_results_student_id_fkey 
        FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Foreign key constraints added to answers and exam_results tables';
    END IF;
END $$;
