package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

// CFAccessVerifier はCloudflare Access JWTを検証する。
type CFAccessVerifier struct {
	audience string
	issuer   string
	certsURL string
	keys     map[string]*rsa.PublicKey
	mu       sync.RWMutex
}

// CFAccessClaims はCF Access JWTのクレーム。
type CFAccessClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// NewCFAccessVerifier は新しいCFAccessVerifierを生成する。
func NewCFAccessVerifier(teamDomain, audience string) (*CFAccessVerifier, error) {
	v := &CFAccessVerifier{
		audience: audience,
		issuer:   fmt.Sprintf("https://%s.cloudflareaccess.com", teamDomain),
		certsURL: fmt.Sprintf("https://%s.cloudflareaccess.com/cdn-cgi/access/certs", teamDomain),
		keys:     make(map[string]*rsa.PublicKey),
	}
	if err := v.refreshKeys(); err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	return v, nil
}

// Verify はCF Access JWTを検証してクレームを返す。
func (v *CFAccessVerifier) Verify(tokenStr string) (*CFAccessClaims, error) {
	claims := &CFAccessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		key, err := v.getKey(kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	},
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (v *CFAccessVerifier) getKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	v.mu.RUnlock()
	if ok {
		return key, nil
	}

	// kid未発見時にリフレッシュ
	if err := v.refreshKeys(); err != nil {
		return nil, fmt.Errorf("refresh JWKS: %w", err)
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown kid: %s", kid)
	}
	return key, nil
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *CFAccessVerifier) refreshKeys() error {
	resp, err := http.Get(v.certsURL)
	if err != nil {
		return fmt.Errorf("fetch certs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("certs endpoint returned %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" {
			continue
		}
		pub, err := parseRSAPublicKey(k)
		if err != nil {
			continue
		}
		keys[k.KID] = pub
	}

	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}

func parseRSAPublicKey(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
