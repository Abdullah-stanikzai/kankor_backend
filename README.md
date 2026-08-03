# Kankor Backend API

Production-ready backend for the Kankor online exam system built with Go, Fiber, PostgreSQL, and Redis.

## Features

- **Role-based access control** (Super Admin, Center Admin, Student)
- **RESTful API** with proper error handling and validation
- **JWT authentication** with refresh tokens
- **PostgreSQL database** with optimized schema
- **Redis caching** for improved performance
- **OTP-based authentication** for secure login
- **Docker support** for easy deployment
- **Production-ready logging** and monitoring

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Fiber v2
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Authentication**: JWT
- **Containerization**: Docker & Docker Compose

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose (recommended)
- PostgreSQL (if running locally without Docker)

### Quick Start with Docker

```bash
# Clone the repository
git clone <repository-url>
cd kankor/backend

# Start all services
make docker-up

# Check health
make health

# View logs
make docker-logs
```

### Local Development

```bash
# Install dependencies
make deps

# Run database migrations
make db-migrate

# Start the application
make run

# Or run with hot reload (requires reflex)
make dev
```

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

All protected endpoints require a Bearer token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Endpoints

#### Public Endpoints
- `POST /auth/send-otp` - Send OTP to phone number
- `POST /auth/verify-otp` - Verify OTP and login
- `POST /auth/refresh-token` - Refresh JWT token
- `POST /auth/logout` - Logout user

#### Super Admin Endpoints
- `GET /super-admin/centers` - Get all educational centers
- `POST /super-admin/centers` - Create new center
- `GET /super-admin/centers/:id` - Get center details
- `PUT /super-admin/centers/:id` - Update center
- `DELETE /super-admin/centers/:id` - Delete center
- `PATCH /super-admin/centers/:id/activate` - Activate center
- `PATCH /super-admin/centers/:id/deactivate` - Deactivate center

#### Center Admin Endpoints
- `GET /center-admin/exams` - Get all exams
- `POST /center-admin/exams` - Create new exam
- `GET /center-admin/exams/:id` - Get exam details
- `PUT /center-admin/exams/:id` - Update exam
- `DELETE /center-admin/exams/:id` - Delete exam
- `GET /center-admin/students` - Get enrolled students

#### Student Endpoints
- `GET /student/profile` - Get student profile
- `GET /student/exams/upcoming` - Get upcoming exams
- `GET /student/exams/available` - Get available exams
- `POST /student/exams/:id/start` - Start exam
- `GET /student/exams/:id/questions` - Get exam questions
- `POST /student/exams/:id/submit` - Submit exam

## Environment Variables

Create a `.env` file or set these environment variables:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=kankor_db
JWT_SECRET=your-super-secret-jwt-key
REDIS_ADDR=localhost:6379
APP_ENV=development
```

## Database Schema

The database schema is defined in `db_schema.sql` and includes:

- **Users**: Students, center admins, and super admins
- **Educational Centers**: Managed by super admins
- **Exams**: Created by center admins
- **Questions**: MCQ questions with 4 options each
- **Exam Attempts**: Student exam attempts and results
- **Student Answers**: Individual question answers
- **OTP Codes**: For authentication
- **Refresh Tokens**: For session management

## Project Structure

```
backend/
├── config/          # Configuration and database setup
├── controllers/     # HTTP request handlers
├── middleware/      # Authentication and authorization middleware
├── models/          # Data models and structs
├── routes/          # API route definitions
├── services/        # Business logic layer
├── utils/           # Utility functions
├── main.go          # Application entry point
├── Dockerfile       # Docker configuration
├── docker-compose.yml # Multi-container setup
└── Makefile         # Development commands
```

## Development Commands

```bash
# Build the application
make build

# Run tests
make test

# Clean build artifacts
make clean

# Reset database
make db-reset

# View Docker logs
make docker-logs
```

## Testing

```bash
# Run all tests
make test

# Test authentication endpoint
make test-auth

# Test health endpoint
make test-health
```

## Deployment

### Docker Deployment

```bash
# Build and deploy
docker-compose up -d --build

# Scale the backend service
docker-compose up -d --scale backend=3
```

### Manual Deployment

1. Build the binary:
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
```

2. Set environment variables
3. Run the binary:
```bash
./main
```

## Security Considerations

- Passwords are hashed using bcrypt
- JWT tokens with 24-hour expiration
- Role-based access control
- Input validation and sanitization
- Rate limiting (to be implemented)
- SQL injection prevention through parameterized queries

## Performance Optimization

- Database connection pooling
- Redis caching for frequently accessed data
- Proper database indexing
- Efficient query design
- Pagination for large datasets

## Monitoring

- Structured logging with Fiber logger middleware
- Health check endpoint at `/health`
- Error tracking and metrics (to be implemented)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For support, email support@kankor.af or join our Slack channel.