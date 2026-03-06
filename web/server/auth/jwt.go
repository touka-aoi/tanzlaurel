package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService はJWT発行・検証を行う。
type JWTService struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	ttl        time.Duration
}

// NewJWTService はES256鍵ペアでJWTServiceを生成する。
// keyPath が空の場合はメモリ上で鍵を自動生成する。
func NewJWTService(keyPath string, ttl time.Duration) (*JWTService, error) {
	if keyPath != "" {
		return loadFromFile(keyPath, ttl)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return &JWTService{privateKey: key, publicKey: &key.PublicKey, ttl: ttl}, nil
}

func loadFromFile(path string, ttl time.Duration) (*JWTService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}
	return &JWTService{privateKey: key, publicKey: &key.PublicKey, ttl: ttl}, nil
}

// Sign はJWTトークンを発行する。
func (s *JWTService) Sign(subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(s.privateKey)
}

// Claims はJWTのクレーム。
type Claims = jwt.RegisteredClaims

// Verify はJWTトークンを検証し、クレームを返す。
func (s *JWTService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
