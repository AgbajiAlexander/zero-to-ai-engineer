# Database migrations

This service uses golang-migrate to manage PostgreSQL schema changes.

## Local setup

1. Make sure PostgreSQL is running locally.
2. Export the connection string in the same way as the rest of the app uses `DATABASE_URL`.
3. Install the migration CLI:

```bash
# from the apps/api directory
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.2
```

4. Run migrations against the local database:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/zero_to_ai_engineer?sslmode=disable"
"$HOME/go/bin/migrate" -database "$DATABASE_URL" -path ./migrations up
```

## Notes

- The migration directory is intentionally kept empty until the first real schema change is ready.
- No migrations are run automatically when the API starts.
- Do not commit production credentials or local secrets.
- Keep the `DATABASE_URL` environment variable in local developer shells or `.env` files that are excluded from source control.

## Useful commands

```bash
# apply all pending migrations
"$HOME/go/bin/migrate" -database "$DATABASE_URL" -path ./migrations up

# roll back the most recent migration
"$HOME/go/bin/migrate" -database "$DATABASE_URL" -path ./migrations down 1

# show current migration status
"$HOME/go/bin/migrate" -database "$DATABASE_URL" -path ./migrations version
```
