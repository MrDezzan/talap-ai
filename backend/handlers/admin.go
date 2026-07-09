package handlers

import (
	"encoding/json"
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
