# Zipway URL Shortener API

High-performance URL shortener API built with Go, featuring authentication via Better Auth session validation and Redis caching for optimal performance.

**Version:** 1.0.0

## Overview

Zipway is a fast, scalable URL shortener service that allows authenticated users to create shortened links with optional custom slugs. The API validates sessions through Better Auth (Next.js) and uses Redis for caching to achieve sub-5ms response times.

## Architecture

The project follows Clean Architecture principles with clear separation of concerns:

```
backend go/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── adapters/                # External adapters
│   │   ├── handlers/            # HTTP handlers
│   │   ├── middleware/          # HTTP middleware (auth)
│   │   └── repositories/        # Database & cache repositories
│   └── core/                    # Business logic
│       ├── auth/                # Authentication logic
│       ├── domain/              # Domain models
│       ├── ports/               # Interfaces
│       └── services/            # Business services
├── docs/                        # Swagger documentation
└── docker-compose.yml           # Docker configuration
```

## Technology Stack

- **Language:** Go 1.25.4
- **Web Framework:** Fiber v3
- **Database:** PostgreSQL (Supabase)
- **Cache:** Redis
- **Authentication:** Better Auth (session validation)
- **Documentation:** Swagger/OpenAPI

## Features

- ✅ **Authentication Required:** All link creation requires valid Better Auth session
- ✅ **Custom Slugs:** Users can specify custom slugs for their links
- ✅ **Reserved Slugs:** System protects reserved routes (api, swagger, admin, etc.)
- ✅ **Redis Caching:** Sub-5ms redirect performance with cache
- ✅ **User Association:** All links are associated with authenticated users
- ✅ **Click Tracking:** Automatic click counting and statistics
- ✅ **Public Resolution:** Public endpoint for link resolution (used by frontend)

## Performance

- **Session Validation (cache hit):** <5µs (local in-memory cache)
- **Session Validation (cache miss):** ~20-50ms (Redis/PostgreSQL lookup)
- **Link Creation:** 50-55ms total (optimized with local session cache)
- **Redirect (cache hit):** Instant (<1ms)
- **Redirect (cache miss):** Fast (includes database lookup and cache update)

### Performance Optimizations

- **Multi-tier caching:** Local in-memory cache → Redis → PostgreSQL
- **Local session cache:** Frequently accessed sessions cached in memory (<5µs access time)
- **Optimized Redis pool:** 50 connections, reduced timeouts for faster responses
- **Prepared statements:** Database queries use prepared statements for better performance

## Project Structure

### Core Domain (`internal/core/`)

- **domain/**: Domain models (Link, LinkStatus)
- **ports/**: Interfaces for repositories and services
- **services/**: Business logic (link creation, resolution)
- **auth/**: Session validation from Better Auth cookies

### Adapters (`internal/adapters/`)

- **handlers/**: HTTP request handlers
- **middleware/**: Authentication middleware
- **repositories/**: PostgreSQL and Redis implementations

### Entry Point (`cmd/api/`)

- **main.go**: Application initialization, route setup, dependency injection

## How It Works

### Authentication Flow

1. User logs in via Next.js frontend (Better Auth)
2. Better Auth creates session in PostgreSQL `session` table
3. Frontend sends requests with `__Secure-better-auth.session_token` cookie
4. Backend extracts session token from cookie
5. Backend validates session using multi-tier caching:
   - **First:** Checks local in-memory cache (<5µs for frequent sessions)
   - **Second:** Checks Redis cache with key `session:{sessionID}`
   - **Third:** Checks Redis with full token `{sessionID}` and parses JSON
   - **Fallback:** Queries PostgreSQL `session` table
   - Valid sessions are cached locally (5 min) and in Redis (5 min)
6. `userId` is stored in request context for use in handlers

### Link Creation Flow

1. Client sends POST to `/api/shorten` with cookie
2. Auth middleware validates session and extracts `userId`
3. Handler validates input and reserved slugs
4. Service generates slug (or uses custom) and creates link
5. Link saved to PostgreSQL with `userId`
6. URL cached in Redis for fast retrieval
7. Response returns short URL

### Link Resolution Flow

1. Client requests `/api/resolve/:slug` (public endpoint)
2. Service checks Redis cache first
3. If cache miss, queries PostgreSQL
4. Updates cache and returns target URL
5. Frontend (Next.js) handles 301 redirect

## Setup

### Prerequisites

- Go 1.25.4+
- Docker & Docker Compose
- Supabase account (for PostgreSQL)
- Redis (local or Supabase)

### Environment Variables

Create a `.env` file:

```bash
# Database - Supabase PostgreSQL (use Connection Pooling)
DATABASE_URL=postgresql://postgres.xxxxx:YOUR_PASSWORD@aws-0-us-east-1.pooler.supabase.com:6543/postgres

# Redis - Local or Supabase
REDIS_URL=redis://redis:6379

# API Configuration
BASE_URL=http://localhost:8080
ALLOWED_ORIGIN=http://localhost:3000
SHORT_URL_DOMAIN=http://localhost:8080  # Optional: Custom domain for short URLs (defaults to BASE_URL)
```

### Running with Docker

```bash
# Build and start
docker-compose up --build

# View logs
docker-compose logs -f api

# Stop
docker-compose down
```

### Running Locally

```bash
# Install dependencies
go mod download

# Run
go run cmd/api/main.go
```

## API Endpoints

### Public Endpoints

#### `GET /`

Health check endpoint.

**Response:**

```json
{
  "message": "Zipway URL Shortener API",
  "version": "1.0.0",
  "status": "ok",
  "timestamp": "2025-01-26T21:00:00Z",
  "uptime": "1h30m",
  "swagger": "http://localhost:8080/swagger"
}
```

#### `GET /api/resolve/:slug`

Resolve a shortened link (public, no auth required).

**Response (200):**

```json
{
  "target_url": "https://example.com"
}
```

**Response (404):**

```json
{
  "error": "Link not found"
}
```

### Protected Endpoints

#### `POST /api/shorten`

Create a shortened link (requires authentication).

**Headers:**

```
Content-Type: application/json
Cookie: __Secure-better-auth.session_token=YOUR_TOKEN
```

**Request Body:**

```json
{
  "target_url": "https://example.com",
  "custom_slug": "my-link" // optional
}
```

**Response (200):**

```json
{
  "short_url": "http://localhost:8080/my-link",
  "details": {
    "id": "uuid",
    "short_id": "my-link",
    "target_url": "https://example.com",
    "user_id": "userIdFromSession",
    "status": "ACTIVE",
    "clicks": 0,
    "created_at": "2025-01-26T21:00:00Z"
  }
}
```

**Error Responses:**

- `400`: Invalid input or reserved slug
- `401`: Unauthorized (invalid/expired session)
- `409`: Custom slug already exists
- `500`: Internal server error

## Reserved Slugs

The following slugs cannot be used as custom slugs:

- `api`
- `swagger`
- `shorten`
- `admin`
- `health`
- `metrics`
- `docs`
- `static`
- `assets`
- `favicon.ico`

## Database Schema

### Session Table (Better Auth)

```sql
CREATE TABLE session (
    id TEXT PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    "userId" TEXT NOT NULL,
    "expiresAt" TIMESTAMP NOT NULL,
    "createdAt" TIMESTAMP DEFAULT NOW(),
    "updatedAt" TIMESTAMP DEFAULT NOW(),
    ...
);
```

### URLs Table

```sql
CREATE TABLE urls (
    id VARCHAR(36) PRIMARY KEY,
    "shortId" VARCHAR(255) UNIQUE NOT NULL,
    target_url TEXT NOT NULL,
    "userId" VARCHAR(255),
    status VARCHAR(20) DEFAULT 'ACTIVE',
    "createdAt" TIMESTAMP DEFAULT NOW(),
    clicks INTEGER DEFAULT 0
);
```

## Development

### Generate Swagger Documentation

```bash
swag init -g cmd/api/main.go -o docs
```

### Running Tests

```bash
go test ./...
```

### Code Structure Principles

- **Clean Architecture:** Separation of business logic from infrastructure
- **Dependency Injection:** Interfaces for testability
- **Single Responsibility:** Each package has a clear purpose
- **Performance First:** Optimized queries, Redis caching, async operations

## Production Considerations

1. **Environment Variables:** Use secure secret management
2. **Database Connection Pooling:** Already configured for Supabase
3. **Redis Persistence:** Configure Redis persistence for production
4. **Rate Limiting:** Consider adding rate limiting middleware
5. **Monitoring:** Add logging and metrics collection
6. **HTTPS:** Ensure all connections use HTTPS in production
7. **Session Validation:** Optimized with multi-tier caching (local memory → Redis → PostgreSQL)

## License

[Your License Here]

## Author

Zipway Team
