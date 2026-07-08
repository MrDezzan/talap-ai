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
	db.AutoMigrate(&models.User{}, &models.Portfolio{})
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
