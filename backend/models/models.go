package models

import (
	"time"
	"gorm.io/datatypes"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Email        string    `gorm:"unique;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Grade        string    `json:"grade"`
	City         string    `json:"city"`
	School       string    `json:"school"`
	Role         string    `gorm:"default:'student'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Grant struct {
	ID              uint     `gorm:"primaryKey" json:"id"`
	Name            string   `json:"name"`
	Subtitle        string   `json:"subtitle"`
	MatchPercentage int      `json:"match_percentage"`
	DeadlineDays    int      `json:"deadline_days"`
	Tone            string   `json:"tone"`
	Country         string   `json:"country"`
	Amount          string   `json:"amount"`
	Description     string   `json:"description"`
	Requirements    string   `json:"requirements"`
	Documents       string   `json:"documents"`
	Link            string   `json:"link"`
	Tags            []string `gorm:"serializer:json;type:text" json:"tags"`
}

type Profession struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Category        string         `json:"category"`
	Tags            []string       `gorm:"serializer:json;type:text" json:"tags"`
	Salary          string         `json:"salary"`
	Growth          string         `json:"growth"`
	MatchPercentage int            `json:"match_percentage"`
	Hot             bool           `json:"hot"`
}

type Portfolio struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"uniqueIndex" json:"user_id"`
	Bio          string         `json:"bio"`
	ENT          string         `json:"ent"`
	English      string         `json:"english"`
	Skills       []string       `gorm:"serializer:json;type:text" json:"skills"`
	Interests    []string       `gorm:"serializer:json;type:text" json:"interests"`
	QuizAnswers  datatypes.JSON `gorm:"type:json" json:"quiz_answers"`
	Achievements datatypes.JSON `gorm:"type:json" json:"achievements"`
	AIResult     datatypes.JSON `gorm:"type:json" json:"ai_result"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ChatThread struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_user_updated" json:"user_id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `gorm:"index:idx_user_updated,priority:2" json:"updated_at"`
}

type ChatMessage struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ThreadID  uint           `gorm:"index:idx_thread_created" json:"thread_id"`
	Role      string         `json:"role"` // "user" or "assistant"
	Content   string         `json:"content"`
	Roadmap   datatypes.JSON `gorm:"type:json" json:"roadmap"`
	CVData    datatypes.JSON `gorm:"type:json" json:"cv_data"`
	CreatedAt time.Time      `gorm:"index:idx_thread_created,priority:2" json:"created_at"`
}
