# Safar Backend

MVP backend for Safar ride-sharing platform.  
The project implements authentication, driver trip offers, passenger trip search queue, booking flow, and operational cleanup tools.

## MVP Features

- Health check endpoint (`GET /health`)
- OTP auth flow (`POST /auth/send-otp`, `POST /auth/verify-otp`)
- JWT-protected profile endpoint (`GET /me`)
- Vehicle management for authenticated users
- Driver trip offer creation and listing
- Passenger trip search queue with "next in queue"
- Passenger booking create/list/cancel flow
- Manual cleanup endpoint for expiring old trips
- Swagger/OpenAPI docs + Swagger UI

## Tech Stack

- Go
- Gin
- GORM
- PostgreSQL
- Swagger (`swaggo/swag`, `gin-swagger`)
- Integration tests with `testing`, `httptest`, SQLite (in-memory) for test runs

## Project Structure

```text
cmd/api/                  # app entrypoint
internal/config/          # env/config and database setup
internal/handlers/        # HTTP handlers
internal/middleware/      # JWT auth middleware
internal/models/          # GORM models + request/response DTOs
internal/routes/          # route wiring
internal/services/        # business services (OTP/JWT/cleanup)
tests/                    # integration tests + test helpers
docs/                     # generated Swagger/OpenAPI files
Safar API/                # Bruno collection
```

## Prerequisites

- Go (matching `go.mod`)
- Docker + Docker Compose (for local PostgreSQL)

## Environment Configuration

1. Copy env template:
   ```bash
   cp .env.example .env
   ```
2. Adjust values in `.env` if needed.

Required variables:

- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET`

Variables with defaults:

- `HTTP_PORT` (default: `8080`)
- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5432`)
- `DB_SSLMODE` (default: `disable`)

If a required variable is missing, app startup fails with a clear message.

## Run Locally

1. Start PostgreSQL:
   ```bash
   make docker-up
   ```
2. Start API:
   ```bash
   make run
   ```

API starts on `http://localhost:${HTTP_PORT}`.

## Build, Test, and Lint

- Build binary:
  ```bash
  make build
  ```
- Run tests:
  ```bash
  make test
  ```
- Run tests verbose:
  ```bash
  make test-verbose
  ```
- Format:
  ```bash
  make format
  ```
- Lint (`go vet`):
  ```bash
  make lint
  ```

## Swagger / OpenAPI

Install generator (optional if you use `make swagger` only):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Generate docs:

```bash
make swagger
```

Swagger UI:

```text
http://localhost:${HTTP_PORT}/swagger/index.html
```

## Main API Modules

- `Auth`: OTP send/verify and JWT issuance
- `Users`: current user profile
- `Vehicles`: create/list own vehicles
- `Driver Trips`: create/list own trips
- `Passenger Search`: queue-based trip search and next option
- `Bookings`: create/list/cancel own bookings
- `Admin`: cleanup expired trips (JWT-protected debug/admin endpoint)

## OTP Behavior (MVP Note)

- `POST /auth/send-otp` currently returns the OTP code in response.
- This is intentionally convenient for local MVP/testing.
- In production this must be removed and replaced by real SMS delivery.

## Known Limitations (Current MVP)

- No refresh token flow yet.
- No role-based restriction for admin/debug routes yet.
- No machine-readable error code/key in error payloads yet.
- OTP generation and anti-abuse protections need hardening.
- Startup still uses `AutoMigrate` (versioned SQL migrations planned).

## Next Planned Steps

- Add refresh token support.
- Add role-based access control for admin routes.
- Add machine-readable `error_code`/`error_key` in API errors.
- Move city names to a dedicated table and reference by IDs.
- Add trip confirmation flow (`pending` -> driver-confirmed `active`).

## Developer Notes

- Stop local PostgreSQL:
  ```bash
  make docker-down
  ```
- Refresh dependencies:
  ```bash
  make tidy
  ```
- Regenerate Swagger after handler/model changes:
  ```bash
  make swagger
  ```
- Reset local build/test artifacts:
  ```bash
  make clean
  ```

Troubleshooting:

- If startup fails with "missing required environment variable", check `.env`.
- If DB connection fails, verify PostgreSQL is up and env values match `docker-compose.yml`.
