#!/bin/bash

# Kankor Backend Demo Script

echo "🚀 Starting Kankor Backend Demo..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker to run this demo."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install docker-compose to run this demo."
    exit 1
fi

echo "✅ Docker environment ready"

# Start the services
echo "🐳 Starting services with docker-compose..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 10

# Check if the backend is running
echo "🔍 Checking backend health..."
curl -f http://localhost:8080/health || {
    echo "❌ Backend failed to start"
    docker-compose logs backend
    exit 1
}

echo "✅ Backend is running!"

# Test authentication endpoint
echo "🔐 Testing authentication endpoint..."
curl -X POST http://localhost:8080/api/v1/auth/send-otp \
    -H "Content-Type: application/json" \
    -d '{"phone_number": "93700123456", "purpose": "login"}'

echo ""
echo "🎉 Demo completed successfully!"
echo ""
echo "Available endpoints:"
echo "  Health: http://localhost:8080/health"
echo "  Auth: http://localhost:8080/api/v1/auth/send-otp"
echo ""
echo "To stop the services: docker-compose down"
echo "To view logs: docker-compose logs -f"