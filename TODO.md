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

4. Make JWT claims parsing in middleware type-safe.
   - File: `internal/middleware/auth_middleware.go`
   - Note: avoid unsafe cast `user_id.(float64)` to prevent panic.

5. Introduce versioned SQL migrations and reduce startup `AutoMigrate` dependence.
   - Files: `migrations/`, `internal/config/database.go`
   - Note: use explicit migration tooling for reproducible schema changes.

6. Store cities in a dedicated table and reference them from trips.
   - Files: `internal/models/`, `internal/handlers/trip_handler.go`, `migrations/`
   - Note: normalize city data (e.g., `cities` table + foreign keys) to avoid duplicates and simplify validation/search.

7. Standardize API error payload with machine-readable code/key for client localization.
   - Files: all handlers in `internal/handlers/`
   - Note: include stable `error_code` (or `error_key`) in every error response so mobile/web clients can map localized messages.

## Medium Priority

8. Clean duplicated request definition in Bruno collection.
   - File: `Safar API/auth/02-verify-otp.bru`
   - Note: keep a single `Verify OTP` block with token save logic.
