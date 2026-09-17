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
