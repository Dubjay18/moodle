package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/moodle/internal/auth"
	"github.com/yourname/moodle/internal/models"
	"github.com/yourname/moodle/internal/store"
)

type AuthHandler struct {
	Store           *store.Store
	SupabaseURL     string
	SupabaseAnonKey string
	ClientURL       string
}

func NewAuthHandler(store *store.Store, supabaseURL, supabaseAnonKey, clientURL string) *AuthHandler {
	return &AuthHandler{
		Store:           store,
		SupabaseURL:     supabaseURL,
		SupabaseAnonKey: supabaseAnonKey,
		ClientURL:       clientURL,
	}
}

// Routes sets up auth-related routes
func (h *AuthHandler) Routes(r chi.Router) {
	r.Get("/google", h.googleLogin)
	r.Get("/callback", h.authCallback)
	r.Post("/callback", h.authCallbackPost)
	r.Post("/logout", h.logout)
	r.Get("/user", h.getUser)
}

// GoogleLogin initiates Google OAuth flow via Supabase
func (h *AuthHandler) googleLogin(w http.ResponseWriter, r *http.Request) {
	redirectTo := r.URL.Query().Get("redirect_to")
	fmt.Println("Raw redirect_to:", redirectTo)

	// URL decode the redirect_to parameter
	if redirectTo != "" {
		decoded, err := url.QueryUnescape(redirectTo)
		if err == nil {
			redirectTo = decoded
			fmt.Println("Decoded redirect_to:", redirectTo)
		}
	}

	if redirectTo == "" {
		redirectTo = h.ClientURL
	}

	fmt.Println("Final redirect_to:", redirectTo)

	// Build callback URL with the original redirect_to as a parameter
	callbackURL := fmt.Sprintf("%s/v1/auth/callback", getBaseURL(r))
	// Always include redirect_to parameter to preserve the original destination
	callbackURL += "?redirect_to=" + url.QueryEscape(redirectTo)

	// Build Supabase OAuth URL - redirect to OUR callback, not the final destination
	authURL := fmt.Sprintf("%s/auth/v1/authorize", h.SupabaseURL)
	params := url.Values{
		"provider":    []string{"google"},
		"redirect_to": []string{callbackURL},
	}

	finalURL := authURL + "?" + params.Encode()
	fmt.Println("Supabase OAuth URL:", finalURL)

	// Redirect to Supabase Google OAuth
	http.Redirect(w, r, finalURL, http.StatusTemporaryRedirect)
}

// getBaseURL extracts the base URL from the request
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// AuthCallback handles the OAuth callback from Supabase
func (h *AuthHandler) authCallback(w http.ResponseWriter, r *http.Request) {
	// Check for redirect_to parameter from the callback URL
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = h.ClientURL
	}
	fmt.Println("Callback redirect_to:", redirectTo)
	fmt.Println("h.ClientURL:", h.ClientURL)

	// Supabase returns tokens in URL fragments, so we need JavaScript to extract them
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>Authentication</title></head>
<body>
<script>
console.log('Full URL:', window.location.href);
console.log('Hash:', window.location.hash);
console.log('Search:', window.location.search);

const params = new URLSearchParams(window.location.hash.substring(1));
const accessToken = params.get('access_token');
const refreshToken = params.get('refresh_token');
const error = params.get('error');
const errorDescription = params.get('error_description');

const redirectTo = '%s';

console.log('Access Token:', accessToken ? 'Found' : 'Not found');
console.log('Redirect To:', redirectTo);

if (error) {
    document.body.innerHTML = '<h1>Authentication Error</h1><p>' + error + ': ' + errorDescription + '</p>';
} else if (accessToken) {
    document.body.innerHTML = '<h1>Authentication Successful</h1><p>Processing...</p>';
    
    // Send tokens to backend
    fetch('/v1/auth/callback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            access_token: accessToken, 
            refresh_token: refreshToken,
            redirect_to: redirectTo
        })
    }).then(response => {
        console.log('Backend response:', response.status);
        if (response.ok) {
            document.body.innerHTML = '<h1>Redirecting...</h1><p>Taking you to the app...</p>';
            // Redirect to the original redirect_to URL
            setTimeout(() => {
                window.location.href = redirectTo;
            }, 1000);
        } else {
            document.body.innerHTML = '<h1>Error</h1><p>Failed to authenticate with backend</p>';
        }
    }).catch(err => {
        console.error('Fetch error:', err);
        document.body.innerHTML = '<h1>Error</h1><p>Network error: ' + err.message + '</p>';
    });
} else {
    document.body.innerHTML = '<h1>Authentication</h1><p>No access token found. Please try again.</p>';
    console.log('No access token found in URL fragments');
}
</script>
</body>
</html>`, redirectTo)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// authCallbackPost handles the POST request from the JavaScript callback
func (h *AuthHandler) authCallbackPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		RedirectTo   string `json:"redirect_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	fmt.Println("POST callback - RedirectTo:", req.RedirectTo)

	// Get user info from Supabase
	user, err := h.getUserFromSupabase(req.AccessToken)
	if err != nil {
		fmt.Println("Error getting user from Supabase:", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Upsert user in our database
	if err := h.Store.UpsertUser(r.Context(), user); err != nil {
		fmt.Println("Error upserting user:", err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// Set secure cookies with tokens
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    req.AccessToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour), // 1 hour
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    req.RefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour * 30), // 30 days
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// Logout clears authentication cookies
func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	// Clear cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		HttpOnly: true,
		Expires:  time.Now().Add(-time.Hour),
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Expires:  time.Now().Add(-time.Hour),
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// GetUser returns the current authenticated user
func (h *AuthHandler) getUser(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := h.Store.GetUser(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// getUserFromSupabase fetches user profile from Supabase using access token
func (h *AuthHandler) getUserFromSupabase(accessToken string) (*models.User, error) {
	req, err := http.NewRequest("GET", h.SupabaseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", h.SupabaseAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase API error: %d", resp.StatusCode)
	}

	var supabaseUser struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		UserMeta struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
			Username  string `json:"preferred_username"`
		} `json:"user_metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&supabaseUser); err != nil {
		return nil, err
	}

	// Create our user model
	user := &models.User{
		ID:       supabaseUser.ID,
		Email:    supabaseUser.Email,
		Username: supabaseUser.UserMeta.Username,
		Avatar:   supabaseUser.UserMeta.AvatarURL,
	}

	// If no username from Google, use email prefix
	if user.Username == "" && user.Email != "" {
		user.Username = user.Email[:len(user.Email)-len("@gmail.com")]
	}

	// If no username still, use name
	if user.Username == "" {
		user.Username = supabaseUser.UserMeta.Name
	}

	return user, nil
}
