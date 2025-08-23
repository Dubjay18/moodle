package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

type Movie struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Overview    string `json:"overview"`
	PosterPath  string `json:"poster_path"`
	ReleaseDate string `json:"release_date"`
}

type SearchMoviesResponse struct {
	Page         int     `json:"page"`
	TotalPages   int     `json:"total_pages"`
	TotalResults int     `json:"total_results"`
	Results      []Movie `json:"results"`
}

type TrendingResponse struct {
	Page    int     `json:"page"`
	Results []Movie `json:"results"`
}

type DiscoverResponse struct {
	Page    int     `json:"page"`
	Results []Movie `json:"results"`
}

func New(apiKey, base string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: base,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SearchMovies(ctx context.Context, query string, page int) (*SearchMoviesResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/search/movie")
	q := u.Query()
	q.Set("api_key", c.APIKey)
	q.Set("query", query)
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb status %d", res.StatusCode)
	}
	var out SearchMoviesResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetMovie(ctx context.Context, id int64) (*Movie, error) {
	u := fmt.Sprintf("%s/movie/%d?api_key=%s", c.BaseURL, id, url.QueryEscape(c.APIKey))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb status %d", res.StatusCode)
	}
	var out Movie
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TrendingMovies gets trending movies for a given window (day|week) and page.
func (c *Client) TrendingMovies(ctx context.Context, window string, page int, region string) (*TrendingResponse, error) {
	if window == "" {
		window = "day"
	}
	u, _ := url.Parse(c.BaseURL + "/trending/movie/" + window)
	q := u.Query()
	q.Set("api_key", c.APIKey)
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if region != "" {
		q.Set("region", region)
	}
	u.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb status %d", res.StatusCode)
	}
	var out TrendingResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DiscoverMovies provides a randomized-like feed using discover with sort_by.
// Filters: with_genres, primary_release_year, region, sort_by (popularity.desc|vote_average.desc|release_date.desc)
func (c *Client) DiscoverMovies(ctx context.Context, page int, genre, year, region, sortBy string) (*DiscoverResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/discover/movie")
	q := u.Query()
	q.Set("api_key", c.APIKey)
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if genre != "" {
		q.Set("with_genres", genre)
	}
	if year != "" {
		q.Set("primary_release_year", year)
	}
	if region != "" {
		q.Set("region", region)
	}
	if sortBy == "" {
		sortBy = "popularity.desc"
	}
	q.Set("sort_by", sortBy)
	u.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb status %d", res.StatusCode)
	}
	var out DiscoverResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
