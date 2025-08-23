package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"sync"
)

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwksCache struct {
	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
}

func (c *jwksCache) get(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

func (c *jwksCache) set(kid string, k *rsa.PublicKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys == nil {
		c.keys = map[string]*rsa.PublicKey{}
	}
	c.keys[kid] = k
}

func fetchJWKS(url string) (*jwks, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("jwks fetch failed")
	}
	var set jwks
	if err := json.NewDecoder(res.Body).Decode(&set); err != nil {
		return nil, err
	}
	return &set, nil
}

func decodeJWKToRSA(j jwk) (*rsa.PublicKey, error) {
	if j.Kty != "RSA" {
		return nil, errors.New("unsupported kty")
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, err
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}
