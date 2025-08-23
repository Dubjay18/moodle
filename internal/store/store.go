package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yourname/moodle/internal/models"
)

type Store struct{ DB *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{DB: db} }

// Users
func (s *Store) UpsertUser(ctx context.Context, u *models.User) error {
	if u.ID == "" && u.Email == "" && u.Username == "" {
		return errors.New("missing identifiers")
	}
	return s.DB.WithContext(ctx).Where(models.User{Email: u.Email}).Assign(u).FirstOrCreate(u).Error
}

func (s *Store) GetUser(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	if err := s.DB.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Watchlists
func (s *Store) CreateWatchlist(ctx context.Context, wl *models.Watchlist) error {
	return s.DB.WithContext(ctx).Create(wl).Error
}

func (s *Store) UpdateWatchlist(ctx context.Context, wl *models.Watchlist) error {
	return s.DB.WithContext(ctx).Model(&models.Watchlist{}).Where("id = ? AND owner_id = ?", wl.ID, wl.OwnerID).Updates(map[string]any{
		"title": wl.Title, "description": wl.Description, "is_public": wl.IsPublic,
	}).Error
}

func (s *Store) DeleteWatchlist(ctx context.Context, id, owner string) error {
	return s.DB.WithContext(ctx).Where("id = ? AND owner_id = ?", id, owner).Delete(&models.Watchlist{}).Error
}

func (s *Store) GetWatchlist(ctx context.Context, id string) (*models.Watchlist, error) {
	var wl models.Watchlist
	if err := s.DB.WithContext(ctx).Preload("Items", func(tx *gorm.DB) *gorm.DB { return tx.Order("position ASC") }).First(&wl, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &wl, nil
}

func (s *Store) ListWatchlistsByOwner(ctx context.Context, owner string) ([]models.Watchlist, error) {
	var out []models.Watchlist
	if err := s.DB.WithContext(ctx).Where("owner_id = ?", owner).Order("updated_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListPublicWatchlistsByOwner(ctx context.Context, owner string) ([]models.Watchlist, error) {
	var out []models.Watchlist
	if err := s.DB.WithContext(ctx).Where("owner_id = ? AND is_public = TRUE", owner).Order("updated_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) EnsureWatchlistOwner(ctx context.Context, wlID, owner string) error {
	var count int64
	if err := s.DB.WithContext(ctx).Model(&models.Watchlist{}).Where("id = ? AND owner_id = ?", wlID, owner).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Items
func (s *Store) AddItem(ctx context.Context, it *models.WatchlistItem, owner string) error {
	if err := s.EnsureWatchlistOwner(ctx, it.WatchlistID, owner); err != nil {
		return err
	}
	var pos int
	if err := s.DB.WithContext(ctx).Model(&models.WatchlistItem{}).Where("watchlist_id = ?", it.WatchlistID).Select("COALESCE(MAX(position), -1)+1").Scan(&pos).Error; err != nil {
		return err
	}
	it.Position = pos
	return s.DB.WithContext(ctx).Create(it).Error
}

func (s *Store) RemoveItem(ctx context.Context, wlID, itemID, owner string) error {
	if err := s.EnsureWatchlistOwner(ctx, wlID, owner); err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Where("id = ? AND watchlist_id = ?", itemID, wlID).Delete(&models.WatchlistItem{}).Error
}

// Likes
func (s *Store) Like(ctx context.Context, user, wl string) error {
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "watchlist_id"}}, DoNothing: true}).Create(&models.Like{UserID: user, WatchlistID: wl}).Error
}

func (s *Store) Unlike(ctx context.Context, user, wl string) error {
	return s.DB.WithContext(ctx).Where("user_id = ? AND watchlist_id = ?", user, wl).Delete(&models.Like{}).Error
}

// Trending: like counts in time window (week/month); only public watchlists.
func (s *Store) TopWatchlists(ctx context.Context, window string, limit int) ([]models.Watchlist, error) {
	var out []models.Watchlist
	q := s.DB.WithContext(ctx).Table("watchlists w").Select("w.*").Joins("LEFT JOIN likes l ON l.watchlist_id = w.id").Where("w.is_public = TRUE")
	switch window {
	case "week":
		q = q.Where("l.created_at >= NOW() - interval '7 days'")
	case "month":
		q = q.Where("l.created_at >= NOW() - interval '30 days'")
	default:
		// no window filter
	}
	if err := q.Group("w.id").Order("COUNT(l.id) DESC, w.updated_at DESC").Limit(limit).Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
