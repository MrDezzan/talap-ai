package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"talkd-backend/models"
)

func setupAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Grant{}, &models.Profession{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestUpdateProfession_CanClearHotFlagAndText guards against a real GORM
// gotcha: db.Model(&x).Updates(structValue) silently skips any field that
// is at its Go zero value (false, "", 0), so a naive Updates() call could
// never let an admin uncheck Hot or blank out a text field. UpdateProfession
// must use Select("*") to force every field to actually be written.
func TestUpdateProfession_CanClearHotFlagAndText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAdminTestDB(t)

	prof := models.Profession{Name: "Ветеринарный врач", Category: "Сельское хозяйство", Hot: true, Salary: "300000"}
	if err := db.Create(&prof).Error; err != nil {
		t.Fatalf("failed to seed profession: %v", err)
	}

	r := gin.New()
	r.PUT("/professions/:id", UpdateProfession(db))

	// Send an update that explicitly turns Hot off and blanks Salary —
	// both are Go zero values, which is exactly the case Updates() alone
	// would silently drop.
	payload := map[string]interface{}{
		"name":     "Ветеринарный врач",
		"category": "Сельское хозяйство",
		"hot":      false,
		"salary":   "",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/professions/%d", prof.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.Profession
	if err := db.First(&updated, prof.ID).Error; err != nil {
		t.Fatalf("failed to reload profession: %v", err)
	}
	if updated.Hot {
		t.Error("expected Hot to be cleared to false, but it is still true")
	}
	if updated.Salary != "" {
		t.Errorf("expected Salary to be cleared to empty, got %q", updated.Salary)
	}
}
