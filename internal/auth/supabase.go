package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKeyUserID struct{}

type SupabaseVerifier struct {
	PublicKeyPEMOrJWKS string
	JWKSURL            string
	Audience           string
	Issuer             string
	parsedKey          *rsa.PublicKey
	cache              jwksCache
}

func (v *SupabaseVerifier) lazyParse() error {
	if v.parsedKey != nil {
		return nil
	}
	str := strings.TrimSpace(v.PublicKeyPEMOrJWKS)
	if str == "" {
		return nil
	}
	// If JSON (JWKS), we parse and pick first as fallback static key
	if strings.HasPrefix(str, "{") {
		var set jwks
		if err := json.Unmarshal([]byte(str), &set); err != nil {
			return err
		}
		if len(set.Keys) == 0 {
			return errors.New("jwks empty")
		}
		k, err := decodeJWKToRSA(set.Keys[0])
		if err != nil {
			return err
		}
		v.parsedKey = k
		return nil
	}
	// Assume PEM
	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(str))
	if err != nil {
		return err
	}
	v.parsedKey = key
	return nil
}

func (v *SupabaseVerifier) keyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
	}
	// Prefer static parsed key if present
	if err := v.lazyParse(); err == nil && v.parsedKey != nil {
		return v.parsedKey, nil
	}
	// Try JWKS URL if provided
	if v.JWKSURL != "" {
		kid, _ := token.Header["kid"].(string)
		if kid != "" {
			if k, ok := v.cache.get(kid); ok {
				return k, nil
			}
			set, err := fetchJWKS(v.JWKSURL)
			if err != nil {
				return nil, err
			}
			for _, j := range set.Keys {
				if j.Kid == kid {
					k, err := decodeJWKToRSA(j)
					if err != nil {
						return nil, err
					}
					v.cache.set(kid, k)
					return k, nil
				}
			}
		}
	}
	return nil, errors.New("no verification key")
}

func (v *SupabaseVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tok string

		// Try Authorization header first
		authz := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			tok = strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		} else {
			// Try cookie as fallback for browser requests
			if cookie, err := r.Cookie("access_token"); err == nil {
				tok = cookie.Value
			}
		}

		if tok == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		parsed, err := jwt.Parse(tok, v.keyFunc, jwt.WithAudience(v.Audience), jwt.WithIssuer(v.Issuer))
		if err != nil || !parsed.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(string); ok && sub != "" {
				r = r.WithContext(context.WithValue(r.Context(), ctxKeyUserID{}, sub))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyUserID{}).(string); ok {
		return v
	}
	return ""
}
