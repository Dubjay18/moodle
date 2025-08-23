package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/moodle/internal/ai"
	"github.com/yourname/moodle/internal/auth"
	"github.com/yourname/moodle/internal/handlers"
	httpserver "github.com/yourname/moodle/internal/http"
	"github.com/yourname/moodle/internal/store"
	"github.com/yourname/moodle/internal/tmdb"
)

type Config struct {
	Port                 string `envconfig:"PORT" default:"8080"`
	DatabaseURL          string `envconfig:"DATABASE_URL" required:"true"`
	SupabaseJWTPublicKey string `envconfig:"SUPABASE_JWT_PUBLIC_KEY"`
	SupabaseJWKSURL      string `envconfig:"SUPABASE_JWKS_URL"`
	SupabaseJWTAudience  string `envconfig:"SUPABASE_JWT_AUDIENCE" default:"authenticated"`
	SupabaseJWTIssuer    string `envconfig:"SUPABASE_JWT_ISSUER" required:"true"`
	TMDBAPIKey           string `envconfig:"TMDB_API_KEY" required:"true"`
	TMDBBaseURL          string `envconfig:"TMDB_BASE_URL" default:"https://api.themoviedb.org/3"`
	GeminiAPIKey         string `envconfig:"GEMINI_API_KEY" required:"true"`
	GeminiModel          string `envconfig:"GEMINI_MODEL" default:"gemini-1.5-flash"`
}

func mustLoadEnv() Config {
	_ = godotenv.Load()
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		log.Fatalf("env error: %v", err)
	}
	return c
}

func mustDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sqlDB, _ := db.DB()
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("db ping error: %v", err)
	}
	return db
}

func main() {
	cfg := mustLoadEnv()
	db := mustDB(cfg.DatabaseURL)
	st := store.New(db)
	tmdbClient := tmdb.New(cfg.TMDBAPIKey, cfg.TMDBBaseURL)
	aiClient := ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)

	// Handlers
	wlHandler := handlers.NewWatchlistHandler(st, tmdbClient)
	aiHandler := handlers.NewAIHandler(aiClient)
	userHandler := handlers.NewUserHandler(st)

	// Auth middleware
	verifier := &auth.SupabaseVerifier{PublicKeyPEMOrJWKS: cfg.SupabaseJWTPublicKey, JWKSURL: cfg.SupabaseJWKSURL, Audience: cfg.SupabaseJWTAudience, Issuer: cfg.SupabaseJWTIssuer}

	mounter := func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Get("/search/movies", wlHandler.SearchMovies)
			r.Get("/feed", wlHandler.Feed)
			r.Post("/ai/ask", aiHandler.Ask)
		})
		// Authed routes
		r.Group(func(r chi.Router) {
			r.Use(verifier.Middleware)
			r.Get("/me", userHandler.Me)
			r.Route("/watchlists", wlHandler.Routes)
			// trending can be public but keep here for now or move above
			r.Get("/trending", wlHandler.Trending)
		})
	}

	srv := httpserver.NewServer(mounter)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
