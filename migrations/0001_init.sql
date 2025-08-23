-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,

    email text UNIQUE,
    username text UNIQUE,
    avatar text
);

CREATE TABLE IF NOT EXISTS watchlists (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,

    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title text NOT NULL,
    description text,
    is_public boolean NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_watchlists_owner ON watchlists(owner_id);

CREATE TABLE IF NOT EXISTS watchlist_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,

    watchlist_id uuid NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    tmdb_id bigint NOT NULL,
    title text NOT NULL,
    poster_path text,
    release_date text,
    notes text,
    position int NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_items_watchlist ON watchlist_items(watchlist_id);
CREATE INDEX IF NOT EXISTS idx_items_tmdb ON watchlist_items(tmdb_id);

CREATE TABLE IF NOT EXISTS likes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),

    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    watchlist_id uuid NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    UNIQUE(user_id, watchlist_id)
);

CREATE TABLE IF NOT EXISTS shares (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),

    from_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    watchlist_id uuid NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    message text
);

-- +goose Down
DROP TABLE IF EXISTS shares;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS watchlist_items;
DROP INDEX IF EXISTS idx_items_tmdb;
DROP INDEX IF EXISTS idx_items_watchlist;
DROP TABLE IF EXISTS watchlists;
DROP INDEX IF EXISTS idx_watchlists_owner;
DROP TABLE IF EXISTS users;
