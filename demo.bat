@echo off
setlocal

echo 🚀 Starting Kankor Backend Demo...

REM Check if Docker is available
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker not found. Please install Docker to run this demo.
    exit /b 1
)

REM Check if docker-compose is available
docker-compose --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ docker-compose not found. Please install docker-compose to run this demo.
    exit /b 1
)

echo ✅ Docker environment ready

REM Start the services
echo 🐳 Starting services with docker-compose...
docker-compose up -d

REM Wait for services to be ready
echo ⏳ Waiting for services to start...
timeout /t 10 /nobreak >nul

REM Check if the backend is running
echo 🔍 Checking backend health...
curl -f http://localhost:8080/health >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Backend failed to start
    docker-compose logs backend
    exit /b 1
)

echo ✅ Backend is running!

REM Test authentication endpoint
echo 🔐 Testing authentication endpoint...
curl -X POST http://localhost:8080/api/v1/auth/send-otp ^
    -H "Content-Type: application/json" ^
    -d "{\"phone_number\": \"93700123456\", \"purpose\": \"login\"}"

echo.
echo 🎉 Demo completed successfully!
echo.
echo Available endpoints:
echo   Health: http://localhost:8080/health
echo   Auth: http://localhost:8080/api/v1/auth/send-otp
echo.
echo To stop the services: docker-compose down
echo To view logs: docker-compose logs -f

pause