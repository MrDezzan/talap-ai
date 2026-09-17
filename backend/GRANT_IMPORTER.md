# Grant Importer

A background worker, running inside the main server process, that periodically fetches a
configurable list of source URLs, uses an LLM (via OpenRouter) to extract structured grant
candidates from the raw page text, and inserts them into the database as not-yet-public rows
(`Status: "pending_review"`) for a human admin to review.

## Environment variables

- `ENABLE_GRANT_IMPORTER` — must be exactly `true` to enable the background loop at all. Any
  other value (or unset) disables it and it does nothing.
- `GRANT_IMPORT_SOURCES` — comma-separated list of URLs to fetch and scan. Currently configured
  (in `backend/.env`, not committed) to a curated set of text-rich, server-rendered Kazakhstani
  scholarship/grant pages that update over time:
  - https://get-edu.kz/publication/stipendii-i-granty-uchimsya-za-graniczej-iz-kazahstana/ — overview
    of major international scholarships (Fulbright, Chevening, DAAD, Erasmus+, etc.)
  - https://bolashak.gov.kz/ru/o-stipendii — official Bolashak scholarship page
  - https://globalstudents.kz/ru/turkiyeburslari/ — Türkiye Bursları program details
  - https://studyinkazakhstan.edu.kz/ru/edu/scholarship — scholarships for foreign students
    studying in Kazakhstan
  - https://vipusknik.kz/articles/opublikovany-gosudarstvennye-granty-na-2026-2027-uchebnyy-god-dlya-programm-bakalavriata —
    annual state education grant quotas by specialty (this one is a dated news article; replace
    with the equivalent page for the next admissions cycle each year)

  Sources were checked with a plain HTTP fetch before being added — pages that only render via
  client-side JS (e.g. egov.kz) or that are PDF/Excel downloads (e.g. the Bolashak specialties
  page) return no usable text and were excluded. When adding new sources, verify they return
  substantive text on a simple fetch, not just an SPA shell.
- `OPENAI_API_KEY` / `OPENROUTER_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL` — reused as-is from the
  existing AI setup (see `backend/handlers/ai.go`); no separate LLM configuration is needed.

## Review flow

New imports land with `Status = "pending_review"` and are not shown publicly. An admin must
approve them via `POST /api/admin/grants/:id/approve` (or reject via `/reject`) — see
`GetPendingGrants` / `ApproveGrant` / `RejectGrant` in `backend/handlers/admin.go` — before they
become visible to students.

## Wiring in main.go

Already wired: `importer.StartScheduled(db, 6*time.Hour)` is called in `main.go` **after**
`db.AutoMigrate(...)` and `config.SeedDB(db)`, since `StartScheduled` triggers an immediate first
run that queries/inserts into the `grants` table — calling it before migrations/seeding exist
risks hitting a table or columns (`status`, `source_url`, `fetched_at`) that don't exist yet on a
fresh database.

## Deploying to Railway (or any other environment)

`GRANT_IMPORT_SOURCES` and `ENABLE_GRANT_IMPORTER` must also be set as environment variables in
the deployment target (Railway project settings, etc.) — `.env` on a developer's machine is not
read in production. Use the same source list above, or an updated one.
