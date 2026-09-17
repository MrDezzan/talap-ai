# Grant Importer

A background worker, running inside the main server process, that periodically fetches a
configurable list of source URLs, uses an LLM (via OpenRouter) to extract structured grant
candidates from the raw page text, and inserts them into the database as not-yet-public rows
(`Status: "pending_review"`) for a human admin to review.

## Environment variables

- `ENABLE_GRANT_IMPORTER` — must be exactly `true` to enable the background loop at all. Any
  other value (or unset) disables it and it does nothing.
- `GRANT_IMPORT_SOURCES` — comma-separated list of URLs to fetch and scan, e.g.
  `https://example.gov.kz/grants,https://example.edu.kz/scholarships`
  (replace with real source URLs before enabling). Empty/unset means there is nothing to import.
- `OPENAI_API_KEY` / `OPENROUTER_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL` — reused as-is from the
  existing AI setup (see `backend/handlers/ai.go`); no separate LLM configuration is needed.

## Review flow

New imports land with `Status = "pending_review"` and are not shown publicly. An admin must
approve them via the (separately-built) `/api/admin/grants/pending` review endpoint before they
become visible to students.

## Wiring into main.go

This package intentionally does not touch `main.go` (to avoid conflicting with concurrent work).
Add the following manually:

```go
import (
    // ...existing imports...
    "talkd-backend/importer"
)
```

```go
db := config.InitDB()
importer.StartScheduled(db, 6*time.Hour)
```

Place the `StartScheduled` call right after `db := config.InitDB()`.
