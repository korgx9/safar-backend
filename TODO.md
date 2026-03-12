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

## Medium Priority

6. Clean duplicated request definition in Bruno collection.
   - File: `Safar API/auth/02-verify-otp.bru`
   - Note: keep a single `Verify OTP` block with token save logic.
