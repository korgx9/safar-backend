# Safar Backend

MVP backend for Safar ride-sharing platform.

## Run locally

1. Configure environment variables (`HTTP_PORT`, `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `DB_SSLMODE`, `JWT_SECRET`).
2. Start PostgreSQL (for example via Docker):
   ```bash
   make docker-up
   ```
3. Run the API:
   ```bash
   make run
   ```

## Swagger / OpenAPI

Install swagger generator (optional if you use `make swagger`):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Generate docs:

```bash
make swagger
```

This command generates OpenAPI files in `docs/`.

Open Swagger UI locally:

```text
http://localhost:${HTTP_PORT}/swagger/index.html
```
