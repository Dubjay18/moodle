# Moodle API (Go)

Backend for a mobile app to create, share, like, and discover movie watchlists. Includes TMDb integration for movie metadata and Gemini-powered AI assistant ("Moodle").

## Stack
- Go 1.22+
- chi router
- GORM (with PostgreSQL/Supabase)
- goose for migrations
- TMDb API for movie data
- Google Gemini for AI

## Features
- Auth (Supabase JWT verification)
- Users & Profiles
- Watchlists (create/update/delete)
- Watchlist items (movies from TMDb)
- Likes & Shares
- Trending/top watchlists (weekly/monthly)
- Search via TMDb proxy endpoints
- AI endpoint `/ai/ask` powered by Gemini

## Local setup

1. Prereqs
- Go 1.22+
- PostgreSQL (or Supabase connection string)
- goose installed (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

2. Copy env
```
cp .env.example .env
```

3. Fill env
- DATABASE_URL: Supabase Postgres connection string (no sslmode=verify-full for local)
- SUPABASE_JWT_PUBLIC_KEY: JWKS or public JWK JSON for auth validation
- TMDB_API_KEY: your TMDb API key
- GEMINI_API_KEY: your Google AI API key

4. Run migrations
```
make migrate-up
```

5. Run server
```
make run
```

## Makefile targets
- build, run
- migrate-up, migrate-down, migrate-status, migrate-create name=<name>
- test

## API sketch
- POST /v1/auth/verify (optional helper)
- GET /v1/me
- GET /v1/users/{id}
- POST /v1/watchlists
- GET /v1/watchlists?owner=<id>
- GET /v1/watchlists/{id}
- PATCH /v1/watchlists/{id}
- DELETE /v1/watchlists/{id}
- POST /v1/watchlists/{id}/items
- DELETE /v1/watchlists/{id}/items/{itemId}
- POST /v1/watchlists/{id}/like
- DELETE /v1/watchlists/{id}/like
- POST /v1/watchlists/{id}/share
- GET /v1/trending?window=week|month
- GET /v1/search/movies?q=...
- POST /v1/ai/ask {"query":"..."}

