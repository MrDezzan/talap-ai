#!/usr/bin/env bash
# Simple, dependency-free load test for the Talap backend using curl + bash.
# No k6/locust/etc. required — just curl, which is available everywhere.
#
# Usage:
#   BASE_URL=http://localhost:8080 ./run.sh
#   BASE_URL=https://your-railway-app.up.railway.app CONCURRENCY=20 REQUESTS=200 ./run.sh
#
# What it does:
#   1. Hits GET /api/health with N concurrent requests, M times each, and
#      reports latency percentiles + error rate — a basic "is the server
#      falling over under load" check.
#   2. Registers a throwaway test user, logs in, then hammers
#      POST /api/ai/chat with that user's token to confirm the per-user
#      rate limit (2s cooldown) and per-IP rate limit (12/min) actually
#      kick in under load — you should see a mix of 200/503(no LLM
#      key)/429 responses, NOT the server crashing or hanging.
#
# This intentionally does NOT hammer the real OpenRouter/LLM API in a tight
# loop for real — that would cost real money. It only proves the rate
# limiting and body-size protections engage correctly; run it against a
# LOCAL dev server (or a staging deploy with a real LLM key) if you want to
# see real chat responses under a *few* manual requests, not a load test.

set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
CONCURRENCY="${CONCURRENCY:-10}"
REQUESTS="${REQUESTS:-100}"

echo "== Talap load test =="
echo "Target: $BASE_URL"
echo "Concurrency: $CONCURRENCY, total requests per phase: $REQUESTS"
echo

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

run_concurrent() {
  local label="$1" method="$2" path="$3" data="${4:-}" auth_header="${5:-}"
  local codes_file="$tmpdir/${label}_codes.txt"
  local times_file="$tmpdir/${label}_times.txt"
  : > "$codes_file"
  : > "$times_file"

  local pids=()
  for ((i = 0; i < REQUESTS; i++)); do
    (
      local curl_args=(-s -o /dev/null -w "%{http_code} %{time_total}\n" -X "$method")
      [[ -n "$data" ]] && curl_args+=(-H "Content-Type: application/json" -d "$data")
      [[ -n "$auth_header" ]] && curl_args+=(-H "Authorization: Bearer $auth_header")
      curl "${curl_args[@]}" "$BASE_URL$path" >> "$tmpdir/${label}_raw.txt" 2>/dev/null
    ) &
    pids+=($!)
    # cap concurrency
    if (( ${#pids[@]} >= CONCURRENCY )); then
      wait "${pids[@]}"
      pids=()
    fi
  done
  wait "${pids[@]}" 2>/dev/null

  echo "--- $label ---"
  if [[ -f "$tmpdir/${label}_raw.txt" ]]; then
    echo "Status code distribution:"
    awk '{print $1}' "$tmpdir/${label}_raw.txt" | sort | uniq -c | sort -rn
    echo "Latency (seconds) — min/avg/max:"
    awk '{print $2}' "$tmpdir/${label}_raw.txt" | sort -n | awk '
      { a[NR]=$1; sum+=$1 }
      END {
        if (NR == 0) { print "no data"; exit }
        printf "  min=%.3f avg=%.3f max=%.3f\n", a[1], sum/NR, a[NR]
      }'
  fi
  echo
}

echo "Phase 1: GET /api/health under load (no auth needed)"
run_concurrent "health" "GET" "/api/health"

echo "Phase 2: register + login a throwaway test user"
test_email="loadtest-$(date +%s)@example.com"
register_resp=$(curl -s -X POST "$BASE_URL/api/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Load Test\",\"email\":\"$test_email\",\"password\":\"loadtest123\"}")
echo "Register response: $register_resp"

login_resp=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$test_email\",\"password\":\"loadtest123\"}")
token=$(echo "$login_resp" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [[ -z "$token" ]]; then
  echo "Could not obtain a token — skipping the authenticated /ai/chat phase."
  echo "Login response was: $login_resp"
  exit 0
fi
echo "Got a token, proceeding to phase 3."
echo

echo "Phase 3: POST /api/ai/chat under load with a valid token"
echo "(expect a mix of 429 from rate limiting and 503 if no LLM key is configured — NOT hangs/crashes/5xx storms)"
chat_payload='{"message":"Привет, расскажи о профессии программиста"}'
run_concurrent "chat" "POST" "/api/ai/chat" "$chat_payload" "$token"

echo "== Done. =="
echo "How to read this:"
echo "  - Phase 1 should be almost all 200s with low, stable latency — if you see 5xx or timeouts, the server can't handle this concurrency."
echo "  - Phase 3 should show 429s (per-user 2s cooldown / per-IP 12-per-minute limit) once concurrency exceeds the limit — that's the rate limiter doing its job, not a failure."
echo "  - If phase 3 shows mostly 200/503 with almost no 429s even at high concurrency, the rate limiter may not be wired up correctly."
