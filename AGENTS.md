# onerep-api

Core gym domain service for OneRep — exercises, templates, workouts, progress tracking.

> **Follow [`CONVENTIONS.md`](https://github.com/vladgrskkh/gym/blob/main/CONVENTIONS.md) — the shared engineering conventions for all OneRep services. This file lists the service-specific parts.**

## Stack
- Go 1.26+, PostgreSQL 16, Redis 7, S3/MinIO

## Commands
```bash
make run          # Start server on :8081
make test         # Run all tests
make lint         # golangci-lint run (golden config)
make tools        # Install pinned tools
make generate     # mockery + swagger
make mock         # mockery only
make swagger      # swag init only
make check-generate  # regenerate + fail on diff
make migrate-up   # Apply migrations
make migrate-down # Rollback migrations
```

## Architecture
Domain-based layered structure (see CONVENTIONS.md):

```
cmd/server/              # Entry point (thin)
internal/
  domain/
    exercises/           # exercise entities, value objects, errors
    templates/           # template entities
    workouts/            # workout entities
    progress/            # progress read models
  service/
    exercises/           # use cases + consumer-defined interfaces (ISP)
    templates/
    workouts/            # per-set PR check, finish → Redis job
    progress/
  handler/
    exercises/           # handlers + dto/ + error_mapper.go + mapper.go
    templates/
    workouts/
    progress/
  infrastructure/
    exercises/           # postgres/, s3/ per domain
    templates/
    workouts/
    progress/
  application/           # App wiring (options pattern), route registration
  handler/               # shared HTTP utils: response, context, middleware
  config/                # caarlos0/env
```

Key domain notes:
- `workouts/` — per-set PR check (Epley 1RM) is synchronous; post-workout
  volume recalculation is enqueued to Redis and processed by a worker.
- Progress data is materialized (`progress_1rm`, `progress_volume`), not computed on read.
- S3 media (exercise photos/videos, template photos) via presigned URLs.

## Service dependency
Validates JWT tokens against the Auth service's JWKS endpoint
(`/.well-known/jwks.json`). Public key cached on startup, no per-request calls.
