package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/yourname/moodle/internal/auth"
	"github.com/yourname/moodle/internal/cache"
	"github.com/yourname/moodle/internal/models"
	"github.com/yourname/moodle/internal/store"
	"github.com/yourname/moodle/internal/tmdb"
	"github.com/yourname/moodle/internal/validate"
)

type WatchlistHandler struct {
	Store     *store.Store
	TMDB      *tmdb.Client
	FeedCache *cache.TTLCache[string, []byte]
}

func NewWatchlistHandler(s *store.Store, t *tmdb.Client) *WatchlistHandler {
	return &WatchlistHandler{Store: s, TMDB: t, FeedCache: cache.NewTTL[string, []byte](60 * time.Second)}
}

// Routes is mounted under /watchlists in main.
func (h *WatchlistHandler) Routes(r chi.Router) {
	r.Get("/{id}", h.get)
	r.Get("/", h.listByOwner)
	r.Post("/", h.create)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	// items
	r.Post("/{id}/items", h.addItem)
	r.Delete("/{id}/items/{itemId}", h.removeItem)
	// likes
	r.Post("/{id}/like", h.like)
	r.Delete("/{id}/like", h.unlike)
}

// Public: /v1/search/movies
func (h *WatchlistHandler) SearchMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "q is required"})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	res, err := h.TMDB.SearchMovies(r.Context(), q, page)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

// Public: GET /v1/movies/{id}
// Fetch a single movie from TMDb by its numeric ID.
func (h *WatchlistHandler) Movie(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id must be a positive integer"})
		return
	}

	mv, err := h.TMDB.GetMovie(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(mv)
}

// Public (or semi-public): /v1/trending?window=week|month&limit=20
func (h *WatchlistHandler) Trending(w http.ResponseWriter, r *http.Request) {
	type qT struct {
		Window string `validate:"oneof= week month"`
		Limit  int    `validate:"gte=1,lte=100"`
	}
	q := qT{Window: r.URL.Query().Get("window"), Limit: 20}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if errs := validate.Map(q); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	lists, err := h.Store.TopWatchlists(r.Context(), q.Window, q.Limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(lists)
}

func (h *WatchlistHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wl, err := h.Store.GetWatchlist(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	uid := auth.UserID(r.Context())
	if !wl.IsPublic && wl.OwnerID != uid {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(wl)
}

func (h *WatchlistHandler) listByOwner(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	uid := auth.UserID(r.Context())
	if owner == "" {
		if uid == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "owner required"})
			return
		}
		owner = uid
	}
	var (
		lists []models.Watchlist
		err   error
	)
	if uid != "" && owner == uid {
		lists, err = h.Store.ListWatchlistsByOwner(r.Context(), owner)
	} else {
		lists, err = h.Store.ListPublicWatchlistsByOwner(r.Context(), owner)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(lists)
}

func (h *WatchlistHandler) create(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	type bodyT struct {
		Title       string `validate:"required,min=1,max=200"`
		Description string `validate:"max=1000"`
		IsPublic    bool
	}
	var b bodyT
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if errs := validate.Map(b); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	wl := &models.Watchlist{OwnerID: uid, Title: b.Title, Description: b.Description, IsPublic: b.IsPublic}
	if err := h.Store.CreateWatchlist(r.Context(), wl); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(wl)
}

func (h *WatchlistHandler) update(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	type bodyT struct {
		Title       *string `validate:"omitempty,min=1,max=200"`
		Description *string `validate:"omitempty,max=1000"`
		IsPublic    *bool
	}
	var b bodyT
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if errs := validate.Map(b); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	// fetch existing to merge
	existing, err := h.Store.GetWatchlist(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if existing.OwnerID != uid {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if b.Title != nil {
		existing.Title = *b.Title
	}
	if b.Description != nil {
		existing.Description = *b.Description
	}
	if b.IsPublic != nil {
		existing.IsPublic = *b.IsPublic
	}
	if err := h.Store.UpdateWatchlist(r.Context(), existing); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(existing)
}

func (h *WatchlistHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteWatchlist(r.Context(), id, uid); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WatchlistHandler) addItem(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	wlID := chi.URLParam(r, "id")
	type bodyT struct {
		TMDBID int64  `json:"tmdb_id" validate:"required,gt=0"`
		Notes  string `json:"notes" validate:"max=1000"`
	}
	var b bodyT
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if errs := validate.Map(b); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	mv, err := h.TMDB.GetMovie(r.Context(), b.TMDBID)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	item := &models.WatchlistItem{WatchlistID: wlID, TMDBID: b.TMDBID, Title: mv.Title, PosterPath: mv.PosterPath, ReleaseDate: mv.ReleaseDate, Notes: b.Notes}
	if err := h.Store.AddItem(r.Context(), item, uid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *WatchlistHandler) removeItem(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	wlID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	if err := h.Store.RemoveItem(r.Context(), wlID, itemID, uid); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WatchlistHandler) like(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	wlID := chi.URLParam(r, "id")
	if err := h.Store.Like(r.Context(), uid, wlID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WatchlistHandler) unlike(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	wlID := chi.URLParam(r, "id")
	if err := h.Store.Unlike(r.Context(), uid, wlID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Feed: GET /v1/feed?type=trending|discover&window=day|week&page=1&genre=&year=&region=&sort_by=
// type=trending uses TMDb trending; type=discover uses TMDb discover with filters.
func (h *WatchlistHandler) Feed(w http.ResponseWriter, r *http.Request) {
	type qT struct {
		Type   string `validate:"required,oneof=trending discover"`
		Window string `validate:"omitempty,oneof=day week"`
		Page   int    `validate:"omitempty,gte=1,lte=1000"`
		Genre  string `validate:"omitempty"`
		Year   string `validate:"omitempty"`
		Region string `validate:"omitempty,len=2"`
		SortBy string `validate:"omitempty,oneof=popularity.desc vote_average.desc release_date.desc"`
	}
	q := qT{
		Type:   r.URL.Query().Get("type"),
		Window: r.URL.Query().Get("window"),
		Genre:  r.URL.Query().Get("genre"),
		Year:   r.URL.Query().Get("year"),
		Region: r.URL.Query().Get("region"),
		SortBy: r.URL.Query().Get("sort_by"),
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			q.Page = n
		}
	}
	if errs := validate.Map(q); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	// Build cache key from query
	key := r.URL.RawQuery
	if key == "" {
		key = "_empty"
	}
	if b, ok := h.FeedCache.Get(key); ok {
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
		return
	}

	if q.Type == "trending" {
		res, err := h.TMDB.TrendingMovies(r.Context(), q.Window, q.Page, q.Region)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		b, _ := json.Marshal(res)
		h.FeedCache.Set(key, b)
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
		return
	}
	// discover
	res, err := h.TMDB.DiscoverMovies(r.Context(), q.Page, q.Genre, q.Year, q.Region, q.SortBy)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(res)
	h.FeedCache.Set(key, b)
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// Mount returns a function that adds the routes under the given router
func (h *WatchlistHandler) Mount() func(r chi.Router) {
	return func(r chi.Router) { h.Routes(r) }
}
