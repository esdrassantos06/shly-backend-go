package domain

import "time"

type LinkStatus string

const (
	StatusActive LinkStatus = "ACTIVE"
	StatusPaused LinkStatus = "PAUSED"
)

type Link struct {
	ID        string     `json:"id" db:"id"`
	ShortID   string     `json:"short_id" db:"short_id"`
	Clicks    int        `json:"clicks" db:"clicks"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UserID    *string    `json:"user_id,omitempty" db:"user_id"`
	TargetURL string     `json:"target_url" db:"target_url"`
	Status    LinkStatus `json:"status" db:"status"`
}
