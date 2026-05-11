// Package auth issues and verifies JWTs (HS256) and hashes passwords (bcrypt).
//
// The token model is deliberately simple: short-lived access token + longer
// refresh token. The BFF stores the refresh token in an HttpOnly cookie and
// passes the access token to the FE via memory. Backend services verify the
// access token signature with the same shared secret. For production, swap
// HS256 for an asymmetric algorithm (RS256/EdDSA) and serve JWKS.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Secret     string        `kong:"name='secret',required"`
	Issuer     string        `kong:"name='issuer',default='base-microservice'"`
	AccessTTL  time.Duration `kong:"name='access-ttl',default='15m'"`
	RefreshTTL time.Duration `kong:"name='refresh-ttl',default='720h'"`
}

// Claims are the JWT body. Audience differentiates access vs refresh tokens.
type Claims struct {
	UserID string   `json:"sub"`
	Email  string   `json:"email,omitempty"`
	Name   string   `json:"name,omitempty"`
	Roles  []string `json:"roles,omitempty"`
	Kind   string   `json:"kind"` // "access" | "refresh"
	jwt.RegisteredClaims
}

type Issuer struct{ cfg Config }

func NewIssuer(cfg Config) (*Issuer, error) {
	if len(cfg.Secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 chars")
	}
	return &Issuer{cfg: cfg}, nil
}

func (i *Issuer) Issue(userID, email, name string, roles []string) (access, refresh string, err error) {
	access, err = i.sign(userID, email, name, roles, "access", i.cfg.AccessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = i.sign(userID, email, name, roles, "refresh", i.cfg.RefreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (i *Issuer) sign(userID, email, name string, roles []string, kind string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		Roles:  roles,
		Kind:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(i.cfg.Secret))
}

type Verifier struct {
	secret []byte
	issuer string
}

func NewVerifier(cfg Config) *Verifier {
	return &Verifier{secret: []byte(cfg.Secret), issuer: cfg.Issuer}
}

// Verify checks signature, expiry and issuer. Returns the claims on success.
func (v *Verifier) Verify(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithIssuer(v.issuer))
	if err != nil {
		return nil, err
	}
	c, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

// HashPassword returns a bcrypt hash suitable for storage.
func HashPassword(plaintext string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword returns nil if plaintext matches the stored bcrypt hash.
func CheckPassword(hashed, plaintext string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext))
}
