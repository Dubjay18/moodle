package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Email    string `gorm:"uniqueIndex" json:"email"`
	Username string `gorm:"uniqueIndex" json:"username"`
	Avatar   string `json:"avatar"`
}

type Watchlist struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	OwnerID     string `gorm:"type:uuid;index" json:"owner_id"`
	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`
	IsPublic    bool   `gorm:"default:true" json:"is_public"`

	Items []WatchlistItem `json:"items"`
}

type WatchlistItem struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	WatchlistID string `gorm:"type:uuid;index" json:"watchlist_id"`
	TMDBID      int64  `gorm:"index" json:"tmdb_id"`
	Title       string `json:"title"`
	PosterPath  string `json:"poster_path"`
	ReleaseDate string `json:"release_date"`
	Notes       string `json:"notes"`
	Position    int    `gorm:"default:0" json:"position"`
}

type Like struct {
	ID          string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      string    `gorm:"type:uuid;index" json:"user_id"`
	WatchlistID string    `gorm:"type:uuid;index" json:"watchlist_id"`
}

type Share struct {
	ID          string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	FromUserID  string    `gorm:"type:uuid;index" json:"from_user_id"`
	ToUserID    string    `gorm:"type:uuid;index" json:"to_user_id"`
	WatchlistID string    `gorm:"type:uuid;index" json:"watchlist_id"`
	Message     string    `json:"message"`
}
