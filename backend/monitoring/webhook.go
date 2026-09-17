// Package monitoring provides a minimal, dependency-free alerting hook.
// There is no external error-monitoring service (e.g. Sentry) wired into
// this project, since that would require adding a new go.mod dependency.
// Instead, Alert posts a small JSON payload to a generic webhook URL
// (e.g. a Slack or Discord incoming webhook) configured via the
// ERROR_WEBHOOK_URL env var. If that env var is unset, Alert is a no-op.
package monitoring

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"talkd-backend/logger"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

// Alert fires an out-of-band notification about a notable event (e.g. a
// panic or critical error). It is fire-and-forget: delivery happens in a
// background goroutine and never blocks the caller, and any failure to
// deliver is only logged as a warning, never panics or returns an error.
func Alert(event string, details map[string]any) {
	webhookURL := os.Getenv("ERROR_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	payload := map[string]any{
		"event":   event,
		"details": details,
		"time":    time.Now().Format(time.RFC3339),
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			logger.Warn("monitoring: failed to marshal alert payload", "error", err)
			return
		}

		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			logger.Warn("monitoring: failed to build alert request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Warn("monitoring: failed to deliver alert webhook", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			logger.Warn("monitoring: alert webhook returned non-2xx status", "status", resp.StatusCode)
		}
	}()
}
