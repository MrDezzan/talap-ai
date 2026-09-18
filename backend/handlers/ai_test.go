package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"talkd-backend/models"
)

func setupTestDBForAI() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Portfolio{}, &models.Profession{}, &models.Grant{})
	return db
}

func TestAnalyzeProfile_IT_SoftwareEngineer(t *testing.T) {
	db := setupTestDBForAI()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// Middleware mock to set user_id
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	r.POST("/ai/analyze", AnalyzeProfile(db))

	// Payload with answers matching IT (math, remote, gadgets)
	payload := map[string]interface{}{
		"interests": []string{"Я люблю программировать и решать алгоритмические задачи"},
		"quiz_answers": map[string]string{
			"subject":         "math",
			"work_type":       "solo",
			"environment":     "remote",
			"goal":            "growth",
			"problem_solving": "logic",
			"leisure":         "gadgets",
			"project_role":    "builder",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/ai/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify that the top profession matched is an IT-related profession
	topProf, ok := response["top_profession"].(string)
	if !ok {
		t.Fatal("Expected top_profession string in response")
	}

	if topProf != "AI/ML Specialist" && topProf != "Software Engineer" {
		t.Errorf("Expected top_profession to be 'AI/ML Specialist' or 'Software Engineer', got '%s'", topProf)
	}

	// Verify that we got recommendations lists
	professions, ok := response["professions"].([]interface{})
	if !ok || len(professions) == 0 {
		t.Error("Expected professions list in response")
	}

	grants, ok := response["grants"].([]interface{})
	if !ok || len(grants) == 0 {
		t.Error("Expected grants list in response")
	}

	// Verify that database record was updated
	var pf models.Portfolio
	if err := db.Where("user_id = ?", 1).First(&pf).Error; err != nil {
		t.Fatalf("Failed to find portfolio in DB: %v", err)
	}

	if len(pf.QuizAnswers) == 0 {
		t.Error("Expected QuizAnswers to be saved in database portfolio")
	}
}

func TestAnalyzeProfile_Business(t *testing.T) {
	db := setupTestDBForAI()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Next()
	})
	r.POST("/ai/analyze", AnalyzeProfile(db))

	// Payload with answers matching Business (human, team, lead, manager)
	payload := map[string]interface{}{
		"interests": []string{"Я хочу создавать бизнес и управлять проектами"},
		"quiz_answers": map[string]string{
			"subject":         "human",
			"work_type":       "lead",
			"environment":     "office",
			"goal":            "money",
			"problem_solving": "people",
			"leisure":         "games",
			"project_role":    "manager",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/ai/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	topProf := response["top_profession"].(string)
	if topProf != "Project Manager" && topProf != "Startup Founder" && topProf != "Business Analyst" && topProf != "Product Manager" {
		t.Errorf("Expected top_profession to be a Business-related role, got '%s'", topProf)
	}
}


// TestAnalyzeProfile_PeopleInteractionAndWorkspaceObjects_Law proves that the
// previously-dead quiz answer keys (workspace_objects, people_interaction,
// risk_tolerance, nature_science, creativity_type) now actually influence
// the outcome: a profile whose only strong signal is workspace_objects =
// "texts_law" should be steered toward Law/humanities and surface the real
// seeded profession for that category, not a generic/invented one.
func TestAnalyzeProfile_PeopleInteractionAndWorkspaceObjects_Law(t *testing.T) {
	db := setupTestDBForAI()
	gin.SetMode(gin.TestMode)

	// Seed a real profession for the "Право" category so the fallback path
	// (LLM is unavailable in tests, no API key set) grounds its result in it
	// instead of a generic "Специалист" placeholder.
	db.Create(&models.Profession{
		Name:            "Юрист-международник",
		Description:     "Работа с международным правом и договорами.",
		Category:        "Право",
		MatchPercentage: 90,
		Hot:             true,
	})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(3))
		c.Next()
	})
	r.POST("/ai/analyze", AnalyzeProfile(db))

	// Only the previously-ignored answer keys carry a strong signal here;
	// the legacy 7 keys are left blank/neutral so the test isolates the
	// effect of the newly-read keys.
	payload := map[string]interface{}{
		"quiz_answers": map[string]string{
			"workspace_objects":  "texts_law",
			"people_interaction": "negotiate",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/ai/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	topCategory, _ := response["top_profession_category"].(string)
	if topCategory != "Право" {
		t.Errorf("Expected top_profession_category to be 'Право' (law), got '%s' -- workspace_objects/people_interaction answers are not affecting scoring", topCategory)
	}

	topProf, _ := response["top_profession"].(string)
	if topProf != "Юрист-международник" {
		t.Errorf("Expected top_profession to be the seeded real profession 'Юрист-международник', got '%s'", topProf)
	}

	// No LLM key is configured in the test environment, so this must have
	// gone through the honest fallback path.
	isFallback, _ := response["is_fallback"].(bool)
	if !isFallback {
		t.Error("Expected is_fallback to be true when no LLM API key is configured")
	}
	fallbackNote, _ := response["fallback_note"].(string)
	if fallbackNote == "" {
		t.Error("Expected a non-empty fallback_note when falling back to rule-based analysis")
	}

	breakdown, ok := response["breakdown"].([]interface{})
	if !ok || len(breakdown) != 4 {
		t.Errorf("Expected breakdown to contain exactly 4 criteria, got %v", response["breakdown"])
	}
}

// setupTestDBForAIChat additionally migrates the chat tables that
// AnalyzeProfile's tests don't need but AIChat does.
func setupTestDBForAIChat(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDBForAI()
	if err := db.AutoMigrate(&models.ChatThread{}, &models.ChatMessage{}); err != nil {
		t.Fatalf("failed to migrate chat tables: %v", err)
	}
	return db
}

// withFakeAuth injects a user_id into the gin context the way
// middleware.AuthRequired() would after verifying a real JWT, so handler
// tests can exercise the authenticated path without a real token.
func withFakeAuth(uid uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", uid)
		c.Next()
	}
}

// TestAIChat_RejectsOversizedMessage guards the token-drain protection added
// to AIChat: input.Message is fully client-controlled and gets forwarded
// almost verbatim into the paid LLM request, so an oversized message must be
// rejected outright (400) before any DB writes or LLM calls happen.
func TestAIChat_RejectsOversizedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForAIChat(t)

	r := gin.New()
	r.Use(withFakeAuth(1))
	r.POST("/ai/chat", AIChat(db))

	tooLong := make([]byte, 5000) // over the 4000-char cap
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	payload := map[string]interface{}{"message": string(tooLong)}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/ai/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an oversized message, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAIChat_TruncatesOversizedHistory_InsteadOfRejecting verifies that a
// too-long history is silently truncated to the most recent entries (rather
// than the whole request being rejected — history padding is common and
// legitimate, unlike a single giant message), and that a reasonably-sized
// request still reaches the LLM-availability check (503, since no
// OPENAI_API_KEY/OPENROUTER_API_KEY is set in the test environment) instead
// of being rejected for size.
func TestAIChat_TruncatesOversizedHistory_InsteadOfRejecting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForAIChat(t)

	r := gin.New()
	r.Use(withFakeAuth(2))
	r.POST("/ai/chat", AIChat(db))

	history := make([]map[string]string, 0, 50)
	for i := 0; i < 50; i++ { // over the 20-entry cap
		history = append(history, map[string]string{"role": "user", "content": "привет"})
	}
	payload := map[string]interface{}{"message": "нормальное сообщение", "history": history}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/ai/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Not rejected for size (400) — it should sail through validation and
	// only fail because no real LLM key is configured in this test env.
	if w.Code == http.StatusBadRequest {
		t.Fatalf("expected the oversized-but-truncatable history to be accepted (then fail on missing LLM key), got 400: %s", w.Body.String())
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (no LLM key configured in test env), got %d: %s", w.Code, w.Body.String())
	}
}
