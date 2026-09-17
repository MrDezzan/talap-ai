package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"talkd-backend/models"
)

// AdminRequired must run AFTER AuthRequired() in the chain (it reads
// "user_id" set by AuthRequired). It aborts with 403 if the requesting
// user's Role is not "admin".
func AdminRequired(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid, ok := userID.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
		var user models.User
		if err := db.First(&user, uid).Error; err != nil || user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ разрешен только администраторам"})
			return
		}
		c.Next()
	}
}
