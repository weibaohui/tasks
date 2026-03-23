package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type AuthHandler struct {
	userService *application.UserApplicationService
	secretKey   []byte
	tokenTTL    time.Duration
}

func NewAuthHandler(userService *application.UserApplicationService, secret string, tokenTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		secretKey:   []byte(secret),
		tokenTTL:    tokenTTL,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenClaims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	UserCode  string `json:"user_code"`
	ExpiresAt int64  `json:"exp"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	user, err := h.userService.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: err.Error()})
		return
	}

	token, expiresAt, err := h.issueToken(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to issue token"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      token,
		"expires_at": expiresAt.UnixMilli(),
		"user":       userToMap(user),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}
	_ = json.NewEncoder(w).Encode(userToMap(user))
}

func (h *AuthHandler) Authorize(r *http.Request) (*domain.User, error) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return nil, errors.New("missing token")
	}
	claims, err := h.verifyToken(token)
	if err != nil {
		return nil, err
	}
	user, err := h.userService.GetUser(r.Context(), domain.NewUserID(claims.UserID))
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, errors.New("inactive user")
	}
	return user, nil
}

func (h *AuthHandler) issueToken(user *domain.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(h.tokenTTL)
	claims := tokenClaims{
		UserID:    user.ID().String(),
		Username:  user.Username(),
		UserCode:  user.UserCode().String(),
		ExpiresAt: expiresAt.Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := signPayload(h.secretKey, payloadEncoded)
	return payloadEncoded + "." + signature, expiresAt, nil
}

func (h *AuthHandler) verifyToken(token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	payloadEncoded := parts[0]
	signature := parts[1]
	expectedSignature := signPayload(h.secretKey, payloadEncoded)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return nil, err
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func signPayload(secretKey []byte, payloadEncoded string) string {
	mac := hmac.New(sha256.New, secretKey)
	_, _ = mac.Write([]byte(payloadEncoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
