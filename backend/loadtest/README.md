# Load testing

`run.sh` is a dependency-free (curl + bash only) load/smoke test for the
backend. No k6/locust/JMeter install required.

## Usage

```bash
# against a local dev server
BASE_URL=http://localhost:8080 ./run.sh

# against a deployed instance, with more load
BASE_URL=https://your-app.up.railway.app CONCURRENCY=20 REQUESTS=200 ./run.sh
```

## What it checks

1. `GET /api/health` under concurrent load — basic "does the server fall
   over" check with latency percentiles.
2. Registers a throwaway test user, logs in, then hits `POST /api/ai/chat`
   concurrently with that user's token to confirm the rate limiting
   (per-user 2s cooldown in the handler, per-IP 12/minute in
   `middleware.RateLimit`) actually engages under load — you should see a
   mix of `429`s and `503`s (the latter if no `OPENAI_API_KEY`/
   `OPENROUTER_API_KEY` is configured), not hangs or a `5xx` storm.

This intentionally does not hammer the real LLM API in a tight loop — that
would cost real money for no benefit. It proves the *protection* engages
(rate limits, body size caps), not the LLM's own throughput.

## Before a real public launch

This script is a smoke test, not a substitute for proper load testing with
a tool like k6 or Locust against a staging environment sized like
production, run by someone who can also watch the database's CPU/connection
count during the run. Treat `run.sh` as "did I obviously break something",
not "is this production-ready at scale".
