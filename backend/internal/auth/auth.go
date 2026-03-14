package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

// ctxKey is an unexported type for context keys in this package.
type ctxKey int

const (
	ctxKeyUserID    ctxKey = iota
	ctxKeySessionID ctxKey = iota
	ctxKeyAnonymous ctxKey = iota
)

// UserIDFromCtx returns the authenticated user ID from the request context.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// SessionIDFromCtx returns the session UUID from the request context.
func SessionIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySessionID).(string)
	return v
}

// IsAnonymousFromCtx returns true if the request is from an anonymous session.
func IsAnonymousFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyAnonymous).(bool)
	return v
}

// Handler holds dependencies for auth HTTP handlers.
type Handler struct {
	pg     *store.PostgresStore
	redis  *store.RedisStore
	jwtSvc *JWTService
}

// NewHandler creates a new auth Handler.
func NewHandler(pg *store.PostgresStore, redis *store.RedisStore, jwtSvc *JWTService) *Handler {
	return &Handler{pg: pg, redis: redis, jwtSvc: jwtSvc}
}

// AnonymousHandler handles POST /api/v1/auth/anonymous.
// Generates a UUID session, stores it in Postgres, and returns a JWT.
func (h *Handler) AnonymousHandler(w http.ResponseWriter, r *http.Request) {
	var req models.AnonymousRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, "invalid_request", "cannot decode request body", http.StatusBadRequest)
		return
	}

	sessionUUID := uuid.New().String()

	if err := h.pg.CreateAnonymousSession(r.Context(), sessionUUID, req.DeviceID); err != nil {
		log.Error().Err(err).Msg("create anonymous session")
		writeError(w, "internal_error", "failed to create session", http.StatusInternalServerError)
		return
	}

	accessToken, err := h.jwtSvc.GenerateAccessToken(sessionUUID, sessionUUID, true)
	if err != nil {
		log.Error().Err(err).Msg("generate anonymous access token")
		writeError(w, "internal_error", "failed to generate token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.jwtSvc.GenerateRefreshToken(sessionUUID, sessionUUID, true)
	if err != nil {
		log.Error().Err(err).Msg("generate anonymous refresh token")
		writeError(w, "internal_error", "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	tokenID := uuid.New().String()
	tokenHash := hashToken(refreshToken)
	expiresAt := time.Now().Add(time.Duration(h.jwtSvc.refreshTTLSec) * time.Second)

	if err := h.pg.StoreRefreshToken(r.Context(), tokenID, sessionUUID, sessionUUID, tokenHash, true, expiresAt); err != nil {
		log.Error().Err(err).Msg("store anonymous refresh token")
		writeError(w, "internal_error", "failed to store refresh token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtSvc.AccessTTLSec(),
		UserID:       sessionUUID,
		Anonymous:    true,
	})
}

// SignupHandler handles POST /api/v1/auth/signup.
func (h *Handler) SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "cannot decode request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, "invalid_request", "email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		writeError(w, "invalid_request", "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check for existing account
	existing, err := h.pg.GetUserByEmail(r.Context(), req.Email)
	if err == nil && existing != nil {
		writeError(w, "conflict", "email already registered", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("bcrypt hash")
		writeError(w, "internal_error", "failed to hash password", http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()
	user, err := h.pg.CreateUser(r.Context(), userID, req.Email, string(hash), req.DisplayName)
	if err != nil {
		log.Error().Err(err).Msg("create user")
		writeError(w, "internal_error", "failed to create user", http.StatusInternalServerError)
		return
	}

	// Merge anonymous session if provided
	if req.SessionUUID != "" {
		if err := h.pg.MergeAnonymousSession(r.Context(), req.SessionUUID, user.ID); err != nil {
			log.Warn().Err(err).Str("session_uuid", req.SessionUUID).Msg("merge anonymous session failed (non-fatal)")
		}
	}

	accessToken, refreshToken, err := h.issueTokenPair(r.Context(), user.ID, "", false)
	if err != nil {
		log.Error().Err(err).Msg("issue token pair on signup")
		writeError(w, "internal_error", "failed to issue tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtSvc.AccessTTLSec(),
		UserID:       user.ID,
		Anonymous:    false,
	})
}

// LoginHandler handles POST /api/v1/auth/login.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "cannot decode request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, "invalid_request", "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.pg.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, "unauthorized", "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, "unauthorized", "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Merge anonymous session if provided
	if req.SessionUUID != "" {
		if err := h.pg.MergeAnonymousSession(r.Context(), req.SessionUUID, user.ID); err != nil {
			log.Warn().Err(err).Str("session_uuid", req.SessionUUID).Msg("merge anonymous session failed (non-fatal)")
		}
	}

	accessToken, refreshToken, err := h.issueTokenPair(r.Context(), user.ID, "", false)
	if err != nil {
		log.Error().Err(err).Msg("issue token pair on login")
		writeError(w, "internal_error", "failed to issue tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtSvc.AccessTTLSec(),
		UserID:       user.ID,
		Anonymous:    false,
	})
}

// RefreshHandler handles POST /api/v1/auth/refresh.
func (h *Handler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "cannot decode request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		writeError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)
		return
	}

	// Validate JWT signature and kind
	claims, err := h.jwtSvc.ValidateToken(req.RefreshToken)
	if err != nil {
		writeError(w, "unauthorized", "invalid refresh token", http.StatusUnauthorized)
		return
	}

	if claims.Kind != TokenKindRefresh {
		writeError(w, "unauthorized", "not a refresh token", http.StatusUnauthorized)
		return
	}

	// Verify token is in the database and not revoked
	tokenHash := hashToken(req.RefreshToken)
	row, err := h.pg.GetRefreshToken(r.Context(), tokenHash)
	if err != nil {
		writeError(w, "unauthorized", "refresh token not found or expired", http.StatusUnauthorized)
		return
	}

	// Rotate: revoke old, issue new pair
	if err := h.pg.RevokeRefreshToken(r.Context(), row.ID); err != nil {
		log.Error().Err(err).Msg("revoke old refresh token")
		writeError(w, "internal_error", "failed to rotate token", http.StatusInternalServerError)
		return
	}

	accessToken, newRefreshToken, err := h.issueTokenPair(r.Context(), row.UserID, row.SessionID, row.Anonymous)
	if err != nil {
		log.Error().Err(err).Msg("issue token pair on refresh")
		writeError(w, "internal_error", "failed to issue tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtSvc.AccessTTLSec(),
		UserID:       row.UserID,
		Anonymous:    row.Anonymous,
	})
}

// ExtractSession is middleware that parses the Bearer JWT and injects
// userID, sessionID, and anonymous flag into the request context.
// Requests without a valid token are rejected with 401.
func (h *Handler) ExtractSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, "unauthorized", "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, "unauthorized", "Authorization header must be Bearer <token>", http.StatusUnauthorized)
			return
		}

		claims, err := h.jwtSvc.ValidateToken(parts[1])
		if err != nil {
			writeError(w, "unauthorized", "invalid or expired token", http.StatusUnauthorized)
			return
		}

		if claims.Kind != TokenKindAccess {
			writeError(w, "unauthorized", "not an access token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxKeySessionID, claims.SessionID)
		ctx = context.WithValue(ctx, ctxKeyAnonymous, claims.Anonymous)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// issueTokenPair generates an access + refresh token pair and stores the
// refresh token in Postgres.
func (h *Handler) issueTokenPair(ctx context.Context, userID, sessionID string, anonymous bool) (accessToken, refreshToken string, err error) {
	accessToken, err = h.jwtSvc.GenerateAccessToken(userID, sessionID, anonymous)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = h.jwtSvc.GenerateRefreshToken(userID, sessionID, anonymous)
	if err != nil {
		return "", "", err
	}

	tokenID := uuid.New().String()
	tokenHash := hashToken(refreshToken)
	expiresAt := time.Now().Add(time.Duration(h.jwtSvc.refreshTTLSec) * time.Second)

	if err := h.pg.StoreRefreshToken(ctx, tokenID, userID, sessionID, tokenHash, anonymous, expiresAt); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// hashToken returns the SHA-256 hex digest of a token string.
// Tokens are never stored in plain text.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("write json response")
	}
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, errCode, message string, status int) {
	writeJSON(w, status, models.ErrorResponse{
		Error:   errCode,
		Message: message,
		Code:    status,
	})
}

