# Go Backend for Smart Cafeteria

This is the Go backend for the Smart Cafeteria Management System, using PostgreSQL as the database.

## Prerequisites

- Docker and Docker Compose installed
- (Optional) Go 1.21+ for local development without Docker

## Quick Start with Docker

From the project root directory:

```bash
# Start all services (PostgreSQL + Backend)
docker-compose up --build

# First time? The database will be seeded automatically
# Access the API at http://localhost:5000
```

## Local Development (without Docker)

1. **Start PostgreSQL** (using Docker):
   ```bash
   docker-compose up postgres
   ```

2. **Run the backend**:
   ```bash
   cd backend
   cp .env.example .env
   go mod download
   SEED_DB=true go run cmd/server/main.go
   ```

## API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/health` | Health check | No |
| POST | `/api/auth/login` | User login | No |
| POST | `/api/auth/register` | User registration | No |
| GET | `/api/menu` | Get menu items | No |
| GET | `/api/users/me` | Get current user | Yes |
| PUT | `/api/users/me` | Update profile | Yes |
| GET | `/api/slots/today` | Get today's slots | Yes |
| GET | `/api/bookings` | Get all bookings | Yes |
| POST | `/api/bookings` | Create booking | Yes |
| GET | `/api/queue/status` | Queue status | Yes |
| GET | `/api/analytics/dashboard` | Dashboard data | Yes |

## Demo Credentials

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@cafeteria.com | admin123 |
| Student | john.keller@university.edu | john123 |

## Project Structure

```
backend/
├── cmd/server/main.go     # Entry point
├── internal/
│   ├── config/            # Configuration
│   ├── database/          # DB connection & seeding
│   ├── handlers/          # HTTP handlers
│   ├── middleware/        # Auth, CORS
│   └── models/            # GORM models
├── Dockerfile
├── go.mod
└── .env.example
```
