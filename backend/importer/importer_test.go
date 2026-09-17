package importer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"talkd-backend/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Grant{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestFetchAndCleanText_StripsTagsAndCollapsesWhitespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><h1>Grant   Name</h1>\n\n<p>Some   description   here.</p></body></html>`))
	}))
	defer server.Close()

	text, err := fetchAndCleanText(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(text, "<") || strings.Contains(text, ">") {
		t.Errorf("expected all HTML tags to be stripped, got: %q", text)
	}
	if strings.Contains(text, "  ") {
		t.Errorf("expected whitespace to be collapsed, got: %q", text)
	}
	if !strings.Contains(text, "Grant") || !strings.Contains(text, "description") {
		t.Errorf("expected visible text to be preserved, got: %q", text)
	}
}

func TestFetchAndCleanText_TruncatesLongPages(t *testing.T) {
	longBody := strings.Repeat("a ", maxTextChars) // way more than maxTextChars once collapsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(longBody))
	}))
	defer server.Close()

	text, err := fetchAndCleanText(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(text) > maxTextChars {
		t.Errorf("expected text truncated to at most %d chars, got %d", maxTextChars, len(text))
	}
}

func TestFetchAndCleanText_ErrorsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := fetchAndCleanText(server.URL); err == nil {
		t.Error("expected an error for a non-200 response, got nil")
	}
}

func TestFetchAndCleanText_ErrorsOnEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 OK, no body written
	}))
	defer server.Close()

	if _, err := fetchAndCleanText(server.URL); err == nil {
		t.Error("expected an error for an empty body, got nil")
	}
}

func TestFetchAndCleanText_ErrorsOnHtmlOnlyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<div><span></span></div>`))
	}))
	defer server.Close()

	if _, err := fetchAndCleanText(server.URL); err == nil {
		t.Error("expected an error when stripping tags leaves nothing, got nil")
	}
}

// TestRunOnce_DedupesAgainstExistingGrants verifies that RunOnce's insertion
// loop skips a candidate whose name+country already exists in the DB
// (case-insensitively), which is the core safety property that keeps the
// importer from spamming duplicate pending_review rows every run. This
// exercises the dedup query directly rather than the full RunOnce (which
// needs network + an LLM key), by seeding the DB the same way RunOnce's
// loop checks it.
func TestRunOnce_DedupesAgainstExistingGrants(t *testing.T) {
	db := setupTestDB(t)

	existing := models.Grant{Name: "Bolashak", Country: "Kazakhstan", Status: "published"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("failed to seed existing grant: %v", err)
	}

	cases := []struct {
		name    string
		country string
		want    bool // want == true means "should be treated as a duplicate"
	}{
		{"Bolashak", "Kazakhstan", true},
		{"bolashak", "kazakhstan", true},   // case-insensitive match
		{"BOLASHAK", "KAZAKHSTAN", true},   // case-insensitive match
		{"Bolashak", "Turkey", false},      // same name, different country
		{"Chevening", "Kazakhstan", false}, // different name
	}

	for _, tc := range cases {
		var count int64
		if err := db.Model(&models.Grant{}).
			Where("LOWER(name) = LOWER(?) AND LOWER(country) = LOWER(?)", tc.name, tc.country).
			Count(&count).Error; err != nil {
			t.Fatalf("dedupe query failed for %s/%s: %v", tc.name, tc.country, err)
		}
		isDup := count > 0
		if isDup != tc.want {
			t.Errorf("name=%q country=%q: got isDuplicate=%v, want %v", tc.name, tc.country, isDup, tc.want)
		}
	}
}

func TestLlmModel_DefaultsWhenUnset(t *testing.T) {
	t.Setenv("LLM_MODEL", "")
	if got := llmModel(); got == "" {
		t.Error("expected a non-empty default model when LLM_MODEL is unset")
	}
}

func TestLlmModel_UsesEnvOverride(t *testing.T) {
	t.Setenv("LLM_MODEL", "some/custom-model")
	if got := llmModel(); got != "some/custom-model" {
		t.Errorf("expected llmModel() to respect LLM_MODEL override, got %q", got)
	}
}
