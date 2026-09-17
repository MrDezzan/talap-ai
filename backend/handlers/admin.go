package handlers

import (
	"encoding/json"
	"strconv"
	"net/http"
	"talkd-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminDashboardStats struct {
	TotalUsers       int64            `json:"total_users"`
	TotalGrants      int64            `json:"total_grants"`
	TotalProfessions int64            `json:"total_professions"`
	TotalChats       int64            `json:"total_chats"`
	TotalMessages    int64            `json:"total_messages"`
	Users            []UserStatDetail `json:"users"`
	CityBreakdown    map[string]int64 `json:"city_breakdown"`
	GradeBreakdown   map[string]int64 `json:"grade_breakdown"`
	ProfBreakdown    map[string]int64 `json:"profession_breakdown"`
}

type UserStatDetail struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	Grade         string    `json:"grade"`
	City          string    `json:"city"`
	School        string    `json:"school"`
	TopProfession string    `json:"top_profession"`
	CreatedAt     time.Time `json:"created_at"`
}

func GetAdminStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify requester is admin
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

		var requester models.User
		if err := db.First(&requester, uid).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		if requester.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Доступ разрешен только администраторам"})
			return
		}

		var stats AdminDashboardStats
		db.Model(&models.User{}).Count(&stats.TotalUsers)
		db.Model(&models.Grant{}).Count(&stats.TotalGrants)
		db.Model(&models.Profession{}).Count(&stats.TotalProfessions)
		db.Model(&models.ChatThread{}).Count(&stats.TotalChats)
		db.Model(&models.ChatMessage{}).Count(&stats.TotalMessages)

		var dbUsers []struct {
			ID           uint
			Name         string
			Email        string
			Role         string
			Grade        string
			City         string
			School       string
			AIResult     []byte
			CreatedAt    time.Time
		}

		db.Table("users").
			Select("users.id, users.name, users.email, users.role, users.grade, users.city, users.school, users.created_at, portfolios.ai_result").
			Joins("LEFT JOIN portfolios ON portfolios.user_id = users.id").
			Order("users.created_at DESC").
			Scan(&dbUsers)

		stats.CityBreakdown = make(map[string]int64)
		stats.GradeBreakdown = make(map[string]int64)
		stats.ProfBreakdown = make(map[string]int64)

		for _, du := range dbUsers {
			topProf := "Не пройден"
			if len(du.AIResult) > 0 {
				var parsed struct {
					TopProf string `json:"top_profession"`
				}
				if err := json.Unmarshal(du.AIResult, &parsed); err == nil && parsed.TopProf != "" {
					topProf = parsed.TopProf
					stats.ProfBreakdown[topProf]++
				}
			}

			cCity := du.City
			if cCity == "" {
				cCity = "Не указан"
			}
			stats.CityBreakdown[cCity]++

			cGrade := du.Grade
			if cGrade == "" {
				cGrade = "Не указан"
			}
			stats.GradeBreakdown[cGrade]++

			stats.Users = append(stats.Users, UserStatDetail{
				ID:            du.ID,
				Name:          du.Name,
				Email:         du.Email,
				Role:          du.Role,
				Grade:         du.Grade,
				City:          du.City,
				School:        du.School,
				TopProfession: topProf,
				CreatedAt:     du.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, stats)
	}
}

// ---- Admin CRUD: Grants ----

func parseAdminIDParam(c *gin.Context) (uint, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор"})
		return 0, false
	}
	return uint(id), true
}

func CreateGrant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var grant models.Grant
		if err := c.ShouldBindJSON(&grant); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные гранта"})
			return
		}
		grant.ID = 0 // ignore any client-supplied id

		if err := db.Create(&grant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать грант"})
			return
		}

		InvalidateGrantsCache()

		c.JSON(http.StatusCreated, grant)
	}
}

func UpdateGrant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		var grant models.Grant
		if err := db.First(&grant, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Грант не найден"})
			return
		}

		var input models.Grant
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные гранта"})
			return
		}
		input.ID = 0 // never let the body change the primary key

		// Select("*") forces GORM to write every column from input, including
		// zero values (empty string, 0, false) — plain .Updates(struct) silently
		// skips zero-value fields, so without this an admin could never clear a
		// text field back to empty.
		if err := db.Model(&grant).Select("*").Updates(input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить грант"})
			return
		}

		InvalidateGrantsCache()

		c.JSON(http.StatusOK, grant)
	}
}

func DeleteGrant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		if err := db.Delete(&models.Grant{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить грант"})
			return
		}

		InvalidateGrantsCache()

		c.Status(http.StatusNoContent)
	}
}

// ---- Admin CRUD: Professions ----

func CreateProfession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var profession models.Profession
		if err := c.ShouldBindJSON(&profession); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные профессии"})
			return
		}
		profession.ID = 0 // ignore any client-supplied id

		if err := db.Create(&profession).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать профессию"})
			return
		}

		InvalidateProfessionsCache()

		c.JSON(http.StatusCreated, profession)
	}
}

func UpdateProfession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		var profession models.Profession
		if err := db.First(&profession, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Профессия не найдена"})
			return
		}

		var input models.Profession
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные профессии"})
			return
		}
		input.ID = 0 // never let the body change the primary key

		// Select("*") forces GORM to write every column from input, including
		// zero values (empty string, 0, false) — plain .Updates(struct) silently
		// skips zero-value fields, so without this an admin could never
		// uncheck Hot or clear a text field back to empty.
		if err := db.Model(&profession).Select("*").Updates(input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить профессию"})
			return
		}

		InvalidateProfessionsCache()

		c.JSON(http.StatusOK, profession)
	}
}

func DeleteProfession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		if err := db.Delete(&models.Profession{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить профессию"})
			return
		}

		InvalidateProfessionsCache()

		c.Status(http.StatusNoContent)
	}
}

// ---- Admin Moderation: Grants ----

func GetPendingGrants(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var grants []models.Grant
		if err := db.Where("status = ?", "pending_review").
			Order("fetched_at DESC NULLS LAST").
			Find(&grants).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить гранты на модерации"})
			return
		}

		c.JSON(http.StatusOK, grants)
	}
}

func ApproveGrant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		if err := db.Model(&models.Grant{}).Where("id = ?", id).Update("status", "published").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось одобрить грант"})
			return
		}

		InvalidateGrantsCache()

		var grant models.Grant
		if err := db.First(&grant, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Грант не найден"})
			return
		}

		c.JSON(http.StatusOK, grant)
	}
}

func RejectGrant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseAdminIDParam(c)
		if !ok {
			return
		}

		if err := db.Delete(&models.Grant{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось отклонить грант"})
			return
		}

		InvalidateGrantsCache()

		c.Status(http.StatusNoContent)
	}
}
