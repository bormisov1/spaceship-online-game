package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwtExpiry       = 7 * 24 * time.Hour // 7 days
	bcryptCost      = 12
	minPasswordLen  = 4
	minUsernameLen  = 2
	maxUsernameLen  = 16
	loginRateWindow = 60 * time.Second
	maxLoginAttempts = 10
)

// Auth handles authentication
type Auth struct {
	db        *DB
	jwtSecret []byte

	// Rate limiting for login attempts (IP -> attempts)
	rateMu    sync.Mutex
	rateMap   map[string]*rateEntry
}

type rateEntry struct {
	Count    int
	ResetAt  time.Time
}

// NewAuth creates a new Auth handler
func NewAuth(db *DB) *Auth {
	secret := loadOrCreateSecret(db)
	return &Auth{
		db:        db,
		jwtSecret: secret,
		rateMap:   make(map[string]*rateEntry),
	}
}

// loadOrCreateSecret loads the JWT secret from the database, or generates
// and persists a new one if none exists.
func loadOrCreateSecret(db *DB) []byte {
	if db != nil {
		if h := db.GetSetting("jwt_secret"); h != "" {
			if b, err := hex.DecodeString(h); err == nil && len(b) == 32 {
				return b
			}
		}
	}
	// Generate a new secret
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("failed to generate JWT secret: " + err.Error())
	}
	if db != nil {
		if err := db.SetSetting("jwt_secret", hex.EncodeToString(secret)); err != nil {
			log.Printf("warning: could not persist JWT secret: %v", err)
		}
	}
	return secret
}

// Register creates a new account
func (a *Auth) Register(username, password string) (int64, string, error) {
	username = strings.TrimSpace(username)

	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return 0, "", fmt.Errorf("username must be %d-%d characters", minUsernameLen, maxUsernameLen)
	}
	if len(password) < minPasswordLen {
		return 0, "", fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	exists, err := a.db.UsernameExists(username)
	if err != nil {
		return 0, "", fmt.Errorf("database error")
	}
	if exists {
		return 0, "", fmt.Errorf("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return 0, "", fmt.Errorf("internal error")
	}

	id, err := a.db.CreatePlayer(username, "", string(hash))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create account")
	}

	token, err := a.generateToken(id, username)
	if err != nil {
		return 0, "", fmt.Errorf("internal error")
	}

	return id, token, nil
}

// Login authenticates a user and returns a JWT
func (a *Auth) Login(username, password, ip string) (int64, string, error) {
	// Rate limiting
	if !a.checkRate(ip) {
		return 0, "", fmt.Errorf("too many login attempts, try again later")
	}

	player, err := a.db.GetPlayerByUsername(username)
	if err != nil {
		return 0, "", fmt.Errorf("database error")
	}
	if player == nil {
		return 0, "", fmt.Errorf("invalid username or password")
	}
	if player.PassHash == "" {
		return 0, "", fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(player.PassHash), []byte(password)); err != nil {
		return 0, "", fmt.Errorf("invalid username or password")
	}

	token, err := a.generateToken(player.ID, player.Username)
	if err != nil {
		return 0, "", fmt.Errorf("internal error")
	}

	return player.ID, token, nil
}

// ValidateToken validates a JWT and returns (playerID, username, error)
func (a *Auth) ValidateToken(tokenStr string) (int64, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return a.jwtSecret, nil
	})
	if err != nil {
		return 0, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, "", fmt.Errorf("invalid token")
	}

	pidFloat, ok := claims["pid"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("invalid token claims")
	}
	username, ok := claims["usr"].(string)
	if !ok {
		return 0, "", fmt.Errorf("invalid token claims")
	}

	return int64(pidFloat), username, nil
}

func (a *Auth) generateToken(playerID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"pid": playerID,
		"usr": username,
		"exp": time.Now().Add(jwtExpiry).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *Auth) checkRate(ip string) bool {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()

	now := time.Now()
	entry, ok := a.rateMap[ip]
	if !ok || now.After(entry.ResetAt) {
		a.rateMap[ip] = &rateEntry{Count: 1, ResetAt: now.Add(loginRateWindow)}
		return true
	}
	entry.Count++
	return entry.Count <= maxLoginAttempts
}

// GenerateGuestName creates a unique guest name like "Guest_a3f2"
func GenerateGuestName() string {
	b := make([]byte, 3)
	rand.Read(b)
	return "Guest_" + hex.EncodeToString(b)
}
