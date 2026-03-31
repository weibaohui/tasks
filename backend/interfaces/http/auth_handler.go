package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
)

type AuthHandler struct {
	userService    *application.UserApplicationService
	userTokenRepo  *persistence.SQLiteUserTokenRepository
	idGenerator    *utils.NanoIDGenerator
	secretKey      []byte
}

func NewAuthHandler(userService *application.UserApplicationService, userTokenRepo *persistence.SQLiteUserTokenRepository, idGenerator *utils.NanoIDGenerator, secret string) *AuthHandler {
	return &AuthHandler{
		userService:   userService,
		userTokenRepo: userTokenRepo,
		idGenerator:  idGenerator,
		secretKey:    []byte(secret),
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

// Login 登录获取长期Token
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

	// 生成长期Token（默认30天过期，nil表示永久）
	tokenValue, err := generateSecureToken(32)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to generate token"})
		return
	}

	tokenHash := hashToken(tokenValue)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 默认30天

	token, err := domain.NewUserToken(
		domain.NewUserTokenID(h.idGenerator.Generate()),
		user.ID(),
		"默认Token",
		"登录自动创建",
		tokenHash,
		&expiresAt,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to create token"})
		return
	}

	// 只有在 userTokenRepo 可用时才保存 token
	if h.userTokenRepo != nil {
		if err := h.userTokenRepo.Save(r.Context(), token); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to save token"})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      tokenValue, // 只返回一次，之后不再显示
		"token_id":   token.ID().String(),
		"expires_at": expiresAt.UnixMilli(),
		"user":       userToMap(user),
	})
}

// Me 获取当前用户信息
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}
	_ = json.NewEncoder(w).Encode(userToMap(user))
}

// Authorize 验证请求权限，支持长期Token
func (h *AuthHandler) Authorize(r *http.Request) (*domain.User, error) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return nil, errors.New("missing token")
	}

	// 1. 先尝试验证长期Token（如果 userTokenRepo 可用）
	if h.userTokenRepo != nil {
		tokenHash := hashToken(token)
		userToken, err := h.userTokenRepo.FindByTokenHash(r.Context(), tokenHash)
		if err == nil && userToken != nil && userToken.IsActive() && !userToken.IsExpired() {
			user, err := h.userService.GetUser(r.Context(), userToken.UserID())
			if err != nil {
				return nil, err
			}
			if !user.IsActive() {
				return nil, errors.New("inactive user")
			}
			// 更新最后使用时间
			userToken.UpdateLastUsed()
			_ = h.userTokenRepo.Save(r.Context(), userToken)
			return user, nil
		}
	}

	// 2. 回退到JWT验证（兼容旧token）
	claims, err := h.verifyJWT(token)
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

func (h *AuthHandler) verifyJWT(token string) (*tokenClaims, error) {
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

// generateSecureToken 生成安全的随机Token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken 对Token进行哈希
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ============ Token管理API ============

type CreateTokenRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ExpiresInDays int   `json:"expires_in_days"` // 0或负数表示永久
}

type TokenResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ExpiresAt   *int64  `json:"expires_at,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	LastUsedAt  *int64  `json:"last_used_at,omitempty"`
	IsActive    bool    `json:"is_active"`
	IsExpired   bool    `json:"is_expired"`
}

// CreateToken 创建新的长期Token
func (h *AuthHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	user, err := h.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "name is required"})
		return
	}

	// 生成Token
	tokenValue, err := generateSecureToken(32)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to generate token"})
		return
	}

	tokenHash := hashToken(tokenValue)
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	token, err := domain.NewUserToken(
		domain.NewUserTokenID(h.idGenerator.Generate()),
		user.ID(),
		req.Name,
		req.Description,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to create token: %v", err)})
		return
	}

	if err := h.userTokenRepo.Save(r.Context(), token); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to save token"})
		return
	}

	// 返回Token（只显示一次）
	snap := token.ToSnapshot()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":       tokenValue, // 只在此刻返回一次
		"id":          snap.ID.String(),
		"name":        snap.Name,
		"description": snap.Description,
		"expires_at":  snap.ExpiresAt,
		"created_at":  snap.CreatedAt,
		"is_active":   snap.IsActive,
		"is_expired":  snap.IsExpired,
	})
}

// ListTokens 列出用户的所有Token
func (h *AuthHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	user, err := h.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	tokens, err := h.userTokenRepo.FindByUserID(r.Context(), user.ID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to list tokens"})
		return
	}

	result := make([]TokenResponse, 0, len(tokens))
	for _, token := range tokens {
		snap := token.ToSnapshot()
		result = append(result, TokenResponse{
			ID:          snap.ID.String(),
			Name:        snap.Name,
			Description: snap.Description,
			ExpiresAt:   snap.ExpiresAt,
			CreatedAt:   snap.CreatedAt,
			LastUsedAt:  snap.LastUsedAt,
			IsActive:    snap.IsActive,
			IsExpired:   snap.IsExpired,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tokens": result,
	})
}

// DeleteToken 删除Token
func (h *AuthHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	user, err := h.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
		return
	}

	// 从路径提取token ID
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "invalid token id"})
		return
	}
	tokenID := parts[len(parts)-1]

	// 验证token属于当前用户
	token, err := h.userTokenRepo.FindByID(r.Context(), domain.NewUserTokenID(tokenID))
	if err != nil || token == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "token not found"})
		return
	}
	if token.UserID().String() != user.ID().String() {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusForbidden, Message: "forbidden"})
		return
	}

	if err := h.userTokenRepo.Delete(r.Context(), domain.NewUserTokenID(tokenID)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: "failed to delete token"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "token deleted",
	})
}
