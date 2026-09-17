package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"

	"talkd-backend/logger"
	"talkd-backend/models"
	"talkd-backend/monitoring"
)

var (
	tagStripRe   = regexp.MustCompile(`<[^>]*>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

const maxTextChars = 15000

// openaiClient builds an OpenRouter/OpenAI client using the same env vars
// as backend/handlers/ai.go (OPENAI_API_KEY / OPENROUTER_API_KEY / LLM_BASE_URL).
// This duplicates that tiny bit of setup logic on purpose so this package
// stays independent from files being edited concurrently elsewhere.
func openaiClient() (*openai.Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		key = os.Getenv("OPENROUTER_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("neither OPENAI_API_KEY nor OPENROUTER_API_KEY is set")
	}
	url := os.Getenv("LLM_BASE_URL")
	config := openai.DefaultConfig(key)
	if url != "" {
		config.BaseURL = url
	} else {
		config.BaseURL = "https://openrouter.ai/api/v1"
	}
	return openai.NewClientWithConfig(config), nil
}

func llmModel() string {
	m := os.Getenv("LLM_MODEL")
	if m == "" {
		return "deepseek/deepseek-v4-pro-0813"
	}
	return m
}

// grantCandidate is the shape the LLM is asked to return for each
// extracted scholarship/grant program.
type grantCandidate struct {
	Name                string   `json:"name"`
	Subtitle            string   `json:"subtitle"`
	Country             string   `json:"country"`
	Amount              string   `json:"amount"`
	DeadlineDescription string   `json:"deadline_description"`
	Description         string   `json:"description"`
	Requirements        string   `json:"requirements"`
	Link                string   `json:"link"`
	Tags                []string `json:"tags"`
}

type extractionResult struct {
	Grants []grantCandidate `json:"grants"`
}

// fetchAndCleanText fetches a URL and returns its plain, visible text,
// truncated to a safe budget for LLM context. Returns an error if the
// fetch fails, the status is not 200, or the body is empty.
func fetchAndCleanText(url string) (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty body")
	}

	text := tagStripRe.ReplaceAllString(string(body), " ")
	text = whitespaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("empty text after stripping tags")
	}

	if len(text) > maxTextChars {
		text = text[:maxTextChars]
	}
	return text, nil
}

// extractGrants sends the given page text to the LLM and asks it to
// extract zero or more grant/scholarship candidates as strict JSON.
func extractGrants(sourceText string) ([]grantCandidate, error) {
	client, err := openaiClient()
	if err != nil {
		return nil, err
	}

	systemPrompt := `You are a careful data-extraction assistant helping a career-guidance platform for Kazakhstani students find scholarship and grant programs.

You will be given raw text scraped from a web page. Extract ZERO OR MORE scholarship/grant program candidates that are clearly and confidently described in the given text.

Rules:
- Only extract programs you are confident are real and clearly described in the text.
- NEVER invent or guess details that are not present in the source text. If a field is not present, leave it as an empty string (or empty array for tags).
- If nothing relevant is found, return an empty "grants" array. Prefer returning fewer or no results over fabricating information.

Respond with STRICT JSON only, matching exactly this shape:
{"grants": [{"name": "...", "subtitle": "...", "country": "...", "amount": "...", "deadline_description": "...", "description": "...", "requirements": "...", "link": "...", "tags": ["..."]}]}`

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llmModel(),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: sourceText},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})
	if err != nil {
		return nil, fmt.Errorf("llm call failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("llm returned no choices")
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result extractionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("llm response not valid JSON: %w", err)
	}
	return result.Grants, nil
}

// RunOnce performs a single import pass over all configured sources:
// fetch each URL, extract candidate grants via LLM, dedupe against
// existing DB rows, and insert new ones with Status = "pending_review".
// Returns the number of new candidates inserted and any error.
func RunOnce(db *gorm.DB) (int, error) {
	rawSources := os.Getenv("GRANT_IMPORT_SOURCES")
	var sources []string
	for _, s := range strings.Split(rawSources, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			sources = append(sources, s)
		}
	}
	if len(sources) == 0 {
		logger.Info("importer: no GRANT_IMPORT_SOURCES configured, skipping")
		return 0, nil
	}

	inserted := 0
	var lastErr error

	for _, sourceURL := range sources {
		text, err := fetchAndCleanText(sourceURL)
		if err != nil {
			logger.Warn("importer: skipping source", "source", sourceURL, "error", err)
			lastErr = err
			continue
		}

		candidates, err := extractGrants(text)
		if err != nil {
			logger.Error("importer: extraction failed for source", "source", sourceURL, "error", err)
			lastErr = err
			continue
		}

		fetchedAt := time.Now()
		for _, cand := range candidates {
			name := strings.TrimSpace(cand.Name)
			if name == "" {
				continue
			}
			country := strings.TrimSpace(cand.Country)

			var count int64
			if err := db.Model(&models.Grant{}).
				Where("LOWER(name) = LOWER(?) AND LOWER(country) = LOWER(?)", name, country).
				Count(&count).Error; err != nil {
				logger.Error("importer: dedupe check failed", "name", name, "error", err)
				lastErr = err
				continue
			}
			if count > 0 {
				continue
			}

			grant := models.Grant{
				Name:         name,
				Subtitle:     cand.Subtitle,
				Country:      country,
				Amount:       cand.Amount,
				Description:  cand.Description,
				Requirements: cand.Requirements,
				Link:         cand.Link,
				Tags:         cand.Tags,
				Status:       "pending_review",
				SourceURL:    sourceURL,
				FetchedAt:    &fetchedAt,
			}
			if err := db.Create(&grant).Error; err != nil {
				logger.Error("importer: insert failed", "name", name, "error", err)
				lastErr = err
				continue
			}
			inserted++
		}
	}

	return inserted, lastErr
}

// StartScheduled launches RunOnce on a ticker in a background goroutine
// and returns immediately. Intended to be called once from main.go at
// startup, e.g.: importer.StartScheduled(db, 6*time.Hour)
// It recovers from panics inside the loop so a bad run never crashes
// the server, and logs errors instead of returning them (nothing is
// listening on the other end of a background goroutine).
func StartScheduled(db *gorm.DB, interval time.Duration) {
	if os.Getenv("ENABLE_GRANT_IMPORTER") != "true" {
		logger.Info("importer: disabled (set ENABLE_GRANT_IMPORTER=true to enable)")
		return
	}

	go func() {
		runSafely := func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("importer: recovered from panic", "panic", r)
					monitoring.Alert("panic", map[string]any{"source": "importer", "panic": fmt.Sprint(r)})
				}
			}()
			n, err := RunOnce(db)
			if err != nil {
				logger.Error("importer: run finished with error", "inserted", n, "error", err)
				monitoring.Alert("importer_run_error", map[string]any{"inserted": n, "error": err.Error()})
			} else {
				logger.Info("importer: inserted new pending grants", "count", n)
			}
		}

		runSafely()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			runSafely()
		}
	}()
}
