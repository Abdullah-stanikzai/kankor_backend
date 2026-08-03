#!/bin/bash
# Run this script to add the score column to the questions table

echo "=========================================="
echo "Adding score column to questions table"
echo "=========================================="
echo ""

# Database connection - update these values if needed
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="kankor"
DB_USER="postgres"

echo "Running migration..."
echo ""

# Run the migration
PGPASSWORD=${DB_PASSWORD} psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f ../migrations/004_add_score_to_exam_engine_questions.sql

echo ""
echo "=========================================="
echo "Migration complete!"
echo "=========================================="
echo ""
echo "Verifying the column was added..."
echo ""

# Verify
PGPASSWORD=${DB_PASSWORD} psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\d questions"
