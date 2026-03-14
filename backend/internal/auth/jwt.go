package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenKind distinguishes access from refresh tokens.
type TokenKind string

const (
	TokenKindAccess  TokenKind = "access"
	TokenKindRefresh TokenKind = "refresh"
)

// Claims is the JWT payload for Cinova tokens.
type Claims struct {
	UserID    string    `json:"uid"`
	SessionID string    `json:"sid,omitempty"` // anonymous session UUID
	Anonymous bool      `json:"anon"`
	Kind      TokenKind `json:"kind"`
	jwt.RegisteredClaims
}

// JWTService issues and validates Cinova JWTs.
type JWTService struct {
	secret        []byte
	accessTTLSec  int
	refreshTTLSec int
}

// NewJWTService constructs a JWTService with the given HMAC secret and TTLs.
func NewJWTService(secret string, accessTTLSec, refreshTTLSec int) *JWTService {
	return &JWTService{
		secret:        []byte(secret),
		accessTTLSec:  accessTTLSec,
		refreshTTLSec: refreshTTLSec,
	}
}

// GenerateAccessToken creates a short-lived access token for the given identity.
func (s *JWTService) GenerateAccessToken(userID string, sessionID string, anonymous bool) (string, error) {
	return s.generate(userID, sessionID, anonymous, TokenKindAccess, time.Duration(s.accessTTLSec)*time.Second)
}

// GenerateRefreshToken creates a long-lived refresh token for the given identity.
func (s *JWTService) GenerateRefreshToken(userID string, sessionID string, anonymous bool) (string, error) {
	return s.generate(userID, sessionID, anonymous, TokenKindRefresh, time.Duration(s.refreshTTLSec)*time.Second)
}

func (s *JWTService) generate(userID, sessionID string, anonymous bool, kind TokenKind, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		Anonymous: anonymous,
		Kind:      kind,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "cinova",
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a signed JWT string, returning the Claims.
func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt parse: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// AccessTTLSec returns the configured access token TTL in seconds.
func (s *JWTService) AccessTTLSec() int { return s.accessTTLSec }
