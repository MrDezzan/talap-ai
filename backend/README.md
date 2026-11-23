# Talap.ai Backend

REST API for Talap.ai, written in Go with Gin and GORM.

## What it does

- Email and password auth with JWT, plus Google OAuth sign-in
- AI endpoints: career consultant chat, roadmap generation, CV drafting (OpenAI-compatible API)
- Public catalogs of grants and professions, with search
- Admin API: platform stats, create/edit/delete grants and professions, approve or reject imported grants
- Grant importer: a background worker that pulls grant listings from configured URLs and saves them as `pending_review` for an admin to approve (see [GRANT_IMPORTER.md](GRANT_IMPORTER.md))
- Rate limiting on auth routes, structured logging, optional webhook alerts on panics
- Tests for handlers, auth and middleware, plus a small load test in `loadtest/`

## Run locally

Requires Go (version in `go.mod`) and a MySQL-compatible database.

```bash
cp .env.example .env     # fill in your own keys
go mod download
go run main.go           # listens on :8080 by default
```

Health check: `GET /api/health`.

Database settings are read from `DATABASE_URL`, or from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS` and `DB_NAME`.

## Docker

```bash
docker build -t talap-backend .
docker run -p 8080:8080 --env-file .env talap-backend
```

## Tests

```bash
go test ./...
```

## Legacy schema tools

`schema.sql` and `setup_db.py` are an older PostgreSQL setup. The script reads the connection from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` and `DB_NAME`, and it wipes and re-creates the schema, so use it only on an empty database.
