# Run this script to add the score column to the questions table
# PowerShell version for Windows

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Adding score column to questions table" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Database connection - update these values if needed
$DB_HOST = "localhost"
$DB_PORT = "5432"
$DB_NAME = "kankor"
$DB_USER = "postgres"
$DB_PASSWORD = "your_password"  # UPDATE THIS

Write-Host "Running migration..." -ForegroundColor Yellow
Write-Host ""

# Set environment variable for password
$env:PGPASSWORD = $DB_PASSWORD

# Run the migration
& psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "..\migrations\004_add_score_to_exam_engine_questions.sql"

Write-Host ""
Write-Host "==========================================" -ForegroundColor Green
Write-Host "Migration complete!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Verifying the column was added..." -ForegroundColor Yellow
Write-Host ""

# Verify
& psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\d questions"

# Clear password from environment
$env:PGPASSWORD = $null
