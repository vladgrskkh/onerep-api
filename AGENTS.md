# onerep-api

Core gym domain service for OneRep — exercises, templates, workouts, progress tracking.

## Stack
- Go 1.23+, PostgreSQL 16, Redis 7, S3/MinIO

## Commands
```bash
make run          # Start server on :8081
make test         # Run all tests
make lint         # go vet ./...
make migrate-up   # Apply migrations
make migrate-down # Rollback migrations
```

## Architecture
DDD with domain-based vertical slicing:

```
cmd/server/          # Entry point
internal/
  domain/
    exercises/       # exercise domain (entities, service, handler)
    templates/       # template domain
    workouts/        # workout domain
    progress/        # progress domain
  application/       # App wire-up, route registration
  handler/           # Shared HTTP utilities (response, middleware, DTOs)
  infrastructure/    # Postgres, Redis, S3, JWT validator
  config/            # Environment configuration
```

## CI
- `golangci-lint` v2 with golden config (maratori)
- Docker build pushes to `ghcr.io/vladgrskkh/onerep-api`

## Local dev
```bash
# Start dependencies
docker compose -f ../docker-compose.yml up -d

# Run
make run
```

## ISP rule
Interfaces are declared where they are consumed (application/), not in infrastructure/.
No central interfaces.go file. Each service declares only the methods it needs.

## Service dependency
Validates JWT tokens against the Auth service's JWKS endpoint (`/.well-known/jwks.json`).
No direct HTTP call to Auth service per request — public key is cached on startup.
