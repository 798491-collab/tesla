package models

import "time"

type AIAnalysis struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"index" json:"user_id"`
	VIN        string    `gorm:"size:32;index" json:"vin"`
	Type       string    `gorm:"size:32;index" json:"type"`
	RefID      string    `gorm:"size:64;index" json:"ref_id"`
	Prompt     string    `gorm:"type:text" json:"prompt"`
	Result     string    `gorm:"type:longtext" json:"result"`
	Summary    string    `gorm:"type:varchar(500)" json:"summary"`
	Model      string    `gorm:"size:64" json:"model"`
	TokensIn   int       `json:"tokens_in"`
	TokensOut  int       `json:"tokens_out"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AIAnalysis) TableName() string { return "tesla_ai_analyses" }
