package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"talkd-backend/auth"
	"talkd-backend/models"
)

var (
	grantsCache           []models.Grant
	grantsCacheMutex      sync.RWMutex
	professionsCache      []models.Profession
	professionsCacheMutex sync.RWMutex
)

func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Grade    string `json:"grade"`
			City     string `json:"city"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		input.Email = strings.TrimSpace(strings.ToLower(input.Email))
		input.Name = strings.TrimSpace(input.Name)

		log.Printf("[DEBUG] Register attempt: email=%s name=%s", input.Email, input.Name)
		if input.Name == "" || input.Email == "" || input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All fields are required"})
			return
		}
		emailRe := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRe.MatchString(input.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
			return
		}
		if len(input.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		var existing models.User
		err = db.Where("email = ?", input.Email).First(&existing).Error
		if err == nil {
			log.Printf("[DEBUG] User already exists: %s (ID: %d)", input.Email, existing.ID)
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Email %s уже зарегистрирован (ID: %d)", input.Email, existing.ID)})
			return
		} else {
			log.Printf("[DEBUG] User not found or error: %v", err)
		}

		user := models.User{
			Name:         input.Name,
			Email:        input.Email,
			PasswordHash: string(hash),
			Grade:        input.Grade,
			City:         input.City,
		}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
	}
}

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		input.Email = strings.TrimSpace(strings.ToLower(input.Email))
		var user models.User
		if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		token, err := auth.GenerateToken(user.ID, user.Email, user.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
	}
}

func GetMe(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
		var user models.User
		if err := db.First(&user, uid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func GetDashboardData(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		grantsCacheMutex.RLock()
		cached := grantsCache
		grantsCacheMutex.RUnlock()

		if len(cached) == 0 {
			var dbGrants []models.Grant
			// Only ever populate the shared public cache with published grants —
			// this cache is also read directly by GetGrants, so an unfiltered
			// query here would leak pending_review (unmoderated, auto-imported)
			// grants to end users if this handler happens to run first.
			db.Where("status = ? OR status = ?", "published", "").Find(&dbGrants)
			grantsCacheMutex.Lock()
			grantsCache = dbGrants
			cached = dbGrants
			grantsCacheMutex.Unlock()
		}

		limit := 4
		if len(cached) < limit {
			limit = len(cached)
		}
		grantsSlice := cached[:limit]

		stats := []gin.H{
			{"label": "Подходящих грантов", "value": "24", "delta": "+3 за неделю", "icon": "award"},
			{"label": "Профессий в фокусе", "value": "6", "delta": "обновлено вчера", "icon": "compass"},
			{"label": "Готовность портфолио", "value": "78%", "delta": "+12% за месяц", "icon": "trophy"},
		}
		c.JSON(http.StatusOK, gin.H{"grants": grantsSlice, "stats": stats})
	}
}

func GetGrants(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := strings.TrimSpace(strings.ToLower(c.Query("q")))
		country := strings.TrimSpace(c.Query("country"))
		tagsParam := strings.TrimSpace(c.Query("tags"))

		grantsCacheMutex.RLock()
		cached := grantsCache
		grantsCacheMutex.RUnlock()

		if len(cached) == 0 {
			var dbGrants []models.Grant
			db.Where("status = ? OR status = ?", "published", "").Find(&dbGrants)
			grantsCacheMutex.Lock()
			grantsCache = dbGrants
			cached = dbGrants
			grantsCacheMutex.Unlock()
		}

		if q == "" && country == "" && tagsParam == "" {
			c.JSON(http.StatusOK, cached)
			return
		}

		result := cached

		if q != "" {
			var filtered []models.Grant
			for _, g := range result {
				name := strings.ToLower(g.Name)
				sub := strings.ToLower(g.Subtitle)
				desc := strings.ToLower(g.Description)
				gCountry := strings.ToLower(g.Country)

				match := strings.Contains(name, q) ||
					strings.Contains(sub, q) ||
					strings.Contains(desc, q) ||
					strings.Contains(gCountry, q)

				if !match {
					for _, tag := range g.Tags {
						if strings.Contains(strings.ToLower(tag), q) {
							match = true
							break
						}
					}
				}

				if match {
					filtered = append(filtered, g)
				}
			}
			result = filtered
		}

		if country != "" {
			var filtered []models.Grant
			for _, g := range result {
				if strings.EqualFold(g.Country, country) {
					filtered = append(filtered, g)
				}
			}
			result = filtered
		}

		if tagsParam != "" {
			wantedTags := strings.Split(tagsParam, ",")
			for i := range wantedTags {
				wantedTags[i] = strings.TrimSpace(wantedTags[i])
			}
			var filtered []models.Grant
			for _, g := range result {
				matched := false
				for _, tag := range g.Tags {
					for _, want := range wantedTags {
						if want != "" && strings.EqualFold(tag, want) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
				if matched {
					filtered = append(filtered, g)
				}
			}
			result = filtered
		}

		c.JSON(http.StatusOK, result)
	}
}

// InvalidateGrantsCache clears the in-memory grants cache so the next
// GetGrants call re-reads fresh data from the database. Call this after
// any create/update/delete of a Grant row.
func InvalidateGrantsCache() {
	grantsCacheMutex.Lock()
	defer grantsCacheMutex.Unlock()
	grantsCache = nil
}

func GetProfessions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := strings.TrimSpace(strings.ToLower(c.Query("q")))
		category := strings.TrimSpace(c.Query("category"))
		tagsParam := strings.TrimSpace(c.Query("tags"))

		professionsCacheMutex.RLock()
		cached := professionsCache
		professionsCacheMutex.RUnlock()

		if len(cached) == 0 {
			var dbProfs []models.Profession
			db.Find(&dbProfs)
			professionsCacheMutex.Lock()
			professionsCache = dbProfs
			cached = dbProfs
			professionsCacheMutex.Unlock()
		}

		if q == "" && category == "" && tagsParam == "" {
			c.JSON(http.StatusOK, cached)
			return
		}

		result := cached

		if q != "" {
			var filtered []models.Profession
			for _, p := range result {
				name := strings.ToLower(p.Name)
				desc := strings.ToLower(p.Description)
				cat := strings.ToLower(p.Category)

				match := strings.Contains(name, q) ||
					strings.Contains(desc, q) ||
					strings.Contains(cat, q)

				if !match {
					for _, tag := range p.Tags {
						if strings.Contains(strings.ToLower(tag), q) {
							match = true
							break
						}
					}
				}

				if match {
					filtered = append(filtered, p)
				}
			}
			result = filtered
		}

		if category != "" {
			var filtered []models.Profession
			for _, p := range result {
				if strings.EqualFold(p.Category, category) {
					filtered = append(filtered, p)
				}
			}
			result = filtered
		}

		if tagsParam != "" {
			wantedTags := strings.Split(tagsParam, ",")
			for i := range wantedTags {
				wantedTags[i] = strings.TrimSpace(wantedTags[i])
			}
			var filtered []models.Profession
			for _, p := range result {
				matched := false
				for _, tag := range p.Tags {
					for _, want := range wantedTags {
						if want != "" && strings.EqualFold(tag, want) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
				if matched {
					filtered = append(filtered, p)
				}
			}
			result = filtered
		}

		c.JSON(http.StatusOK, result)
	}
}

// InvalidateProfessionsCache clears the in-memory professions cache so the
// next GetProfessions call re-reads fresh data from the database. Call this
// after any create/update/delete of a Profession row.
func InvalidateProfessionsCache() {
	professionsCacheMutex.Lock()
	defer professionsCacheMutex.Unlock()
	professionsCache = nil
}

func GlobalSearch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			c.JSON(http.StatusOK, gin.H{"grants": []any{}, "professions": []any{}})
			return
		}

		term := "%" + strings.ToLower(q) + "%"
		var grants []models.Grant
		var professions []models.Profession

		// Use LOWER() for maximum compatibility and robustness
		db.Where("LOWER(name) LIKE ? OR LOWER(subtitle) LIKE ? OR LOWER(country) LIKE ? OR LOWER(array_to_string(tags, ',')) LIKE ?", term, term, term, term).Limit(10).Find(&grants)
		db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(category) LIKE ? OR LOWER(array_to_string(tags, ',')) LIKE ?", term, term, term, term).Limit(10).Find(&professions)

		c.JSON(http.StatusOK, gin.H{
			"grants":      grants,
			"professions": professions,
		})
	}
}

func GetPortfolio(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
		var portfolio models.Portfolio
		if err := db.Where("user_id = ?", uid).First(&portfolio).Error; err != nil {
			c.JSON(http.StatusOK, models.Portfolio{UserID: uid})
			return
		}
		c.JSON(http.StatusOK, portfolio)
	}
}

func UpdatePortfolio(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}

		var input struct {
			Bio          string   `json:"bio"`
			ENT          string   `json:"ent"`
			English      string   `json:"english"`
			Skills       []string `json:"skills"`
			Achievements any      `json:"achievements"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var portfolio models.Portfolio
		db.Where("user_id = ?", uid).FirstOrInit(&portfolio)
		portfolio.UserID = uid
		portfolio.Bio = input.Bio
		portfolio.ENT = input.ENT
		portfolio.English = input.English
		portfolio.Skills = input.Skills
		
		achJSON, _ := json.Marshal(input.Achievements)
		portfolio.Achievements = achJSON
		
		db.Save(&portfolio)
		c.JSON(http.StatusOK, portfolio)
	}
}
func UpdateMe(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}

		var input struct {
			Name   string `json:"name"`
			Grade  string `json:"grade"`
			City   string `json:"city"`
			School string `json:"school"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		if err := db.First(&user, uid).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.Name = input.Name
		user.Grade = input.Grade
		user.City = input.City
		user.School = input.School

		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
