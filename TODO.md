# TODO

## High Priority

1. Remove OTP code from `POST /auth/send-otp` response for non-local environments.
   - File: `internal/handlers/auth_handler.go`
   - Note: keep OTP visible only in local/dev diagnostics if needed.

2. Replace `math/rand` with cryptographically secure generation for OTP.
   - File: `internal/services/otp_service.go`
   - Note: use `crypto/rand` and keep fixed-length 6-digit format.

3. Add rate limiting and attempt limits for OTP flow.
   - Files: `internal/handlers/auth_handler.go`, `internal/services/otp_service.go`
   - Note: limit `send-otp` frequency and failed `verify-otp` attempts.

4. Add refresh token flow for session renewal.
   - Files: `internal/handlers/auth_handler.go`, `internal/services/jwt_service.go`, `internal/models/auth.go`
   - Note: issue access + refresh tokens, add refresh endpoint, and support refresh token rotation/invalidation strategy.

5. Make JWT claims parsing in middleware type-safe.
   - File: `internal/middleware/auth_middleware.go`
   - Note: avoid unsafe cast `user_id.(float64)` to prevent panic.

6. Introduce versioned SQL migrations and reduce startup `AutoMigrate` dependence.
   - Files: `migrations/`, `internal/config/database.go`
   - Note: use explicit migration tooling for reproducible schema changes.

7. Store cities in a dedicated table and reference them from trips.
   - Files: `internal/models/`, `internal/handlers/trip_handler.go`, `migrations/`
   - Note: normalize city data (e.g., `cities` table + foreign keys) to avoid duplicates and simplify validation/search.

8. Add trip confirmation flow: new trip starts in pending status and only driver can confirm it.
   - Files: `internal/models/trip_offer.go`, `internal/handlers/trip_handler.go`, `internal/routes/routes.go`
   - Note: introduce statuses like `pending_confirmation` -> `active`, and block passenger-visible/searchable trips until driver confirmation.

9. Standardize API error payload with machine-readable code/key for client localization.
   - Files: all handlers in `internal/handlers/`
   - Note: include stable `error_code` (or `error_key`) in every error response so mobile/web clients can map localized messages.

10. Restrict admin/debug endpoints by role-based access control.
   - Files: `internal/routes/routes.go`, `internal/middleware/`, `internal/models/user.go`
   - Note: `/admin/*` routes should be accessible only for authorized admin roles, not all authenticated users.

## Medium Priority

11. Clean duplicated request definition in Bruno collection.
   - File: `Safar API/auth/02-verify-otp.bru`
   - Note: keep a single `Verify OTP` block with token save logic.
