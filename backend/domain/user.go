package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrUserIDRequired       = errors.New("user id is required")
	ErrUserCodeRequired     = errors.New("user code is required")
	ErrUsernameRequired     = errors.New("username is required")
	ErrPasswordHashRequired = errors.New("password hash is required")
)

type UserID struct {
	value string
}

func NewUserID(value string) UserID {
	return UserID{value: value}
}

func (id UserID) String() string {
	return id.value
}

type UserCode struct {
	value string
}

func NewUserCode(value string) UserCode {
	return UserCode{value: value}
}

func (c UserCode) String() string {
	return c.value
}

type User struct {
	id           UserID
	userCode     UserCode
	username     string
	email        string
	displayName  string
	passwordHash string
	isActive     bool
	createdAt    time.Time
	updatedAt    time.Time
}

func NewUser(
	id UserID,
	userCode UserCode,
	username string,
	email string,
	displayName string,
	passwordHash string,
) (*User, error) {
	if id.String() == "" {
		return nil, ErrUserIDRequired
	}
	if userCode.String() == "" {
		return nil, ErrUserCodeRequired
	}
	if strings.TrimSpace(username) == "" {
		return nil, ErrUsernameRequired
	}
	if strings.TrimSpace(passwordHash) == "" {
		return nil, ErrPasswordHashRequired
	}

	now := time.Now()
	return &User{
		id:           id,
		userCode:     userCode,
		username:     username,
		email:        email,
		displayName:  displayName,
		passwordHash: passwordHash,
		isActive:     true,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func (u *User) ID() UserID {
	return u.id
}

func (u *User) UserCode() UserCode {
	return u.userCode
}

func (u *User) Username() string {
	return u.username
}

func (u *User) Email() string {
	return u.email
}

func (u *User) DisplayName() string {
	return u.displayName
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) IsActive() bool {
	return u.isActive
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *User) UpdateProfile(email, displayName string) {
	u.email = email
	u.displayName = displayName
	u.updatedAt = time.Now()
}

func (u *User) ChangePasswordHash(passwordHash string) error {
	if strings.TrimSpace(passwordHash) == "" {
		return ErrPasswordHashRequired
	}
	u.passwordHash = passwordHash
	u.updatedAt = time.Now()
	return nil
}

func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

// VerifyPassword verifies the plain password against the stored hash
func (u *User) VerifyPassword(plainPassword string) bool {
	return verifyPassword(u.passwordHash, plainPassword)
}

var sha256HexPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return "sha256$" + hex.EncodeToString(sum[:])
}

func verifyPassword(storedHash, plainPassword string) bool {
	if strings.HasPrefix(storedHash, "sha256$") {
		sum := sha256.Sum256([]byte(plainPassword))
		return strings.EqualFold(strings.TrimPrefix(storedHash, "sha256$"), hex.EncodeToString(sum[:]))
	}
	if strings.HasPrefix(storedHash, "sha256:") {
		sum := sha256.Sum256([]byte(plainPassword))
		return strings.EqualFold(strings.TrimPrefix(storedHash, "sha256:"), hex.EncodeToString(sum[:]))
	}
	if sha256HexPattern.MatchString(storedHash) {
		sum := sha256.Sum256([]byte(plainPassword))
		return strings.EqualFold(storedHash, hex.EncodeToString(sum[:]))
	}
	return false
}

// BuildStoredPasswordValue builds the stored password value from plain password or existing hash
func BuildStoredPasswordValue(plainPassword, passwordHash string) string {
	if plainPassword != "" {
		return hashPassword(plainPassword)
	}
	if passwordHash == "" {
		return ""
	}
	if strings.HasPrefix(passwordHash, "sha256$") || strings.HasPrefix(passwordHash, "sha256:") {
		return passwordHash
	}
	if sha256HexPattern.MatchString(passwordHash) {
		return "sha256$" + strings.ToLower(passwordHash)
	}
	return hashPassword(passwordHash)
}

type UserSnapshot struct {
	ID           UserID
	UserCode     UserCode
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) ToSnapshot() UserSnapshot {
	return UserSnapshot{
		ID:           u.id,
		UserCode:     u.userCode,
		Username:     u.username,
		Email:        u.email,
		DisplayName:  u.displayName,
		PasswordHash: u.passwordHash,
		IsActive:     u.isActive,
		CreatedAt:    u.createdAt,
		UpdatedAt:    u.updatedAt,
	}
}

func (u *User) FromSnapshot(snap UserSnapshot) {
	u.id = snap.ID
	u.userCode = snap.UserCode
	u.username = snap.Username
	u.email = snap.Email
	u.displayName = snap.DisplayName
	u.passwordHash = snap.PasswordHash
	u.isActive = snap.IsActive
	u.createdAt = snap.CreatedAt
	u.updatedAt = snap.UpdatedAt
}
