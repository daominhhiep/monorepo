package internal

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/base/base-microservice/pkg/actor"
	pkgauth "github.com/base/base-microservice/pkg/auth"
)

const (
	AccessCookie  = "_base_access"
	RefreshCookie = "_base_refresh"
)

type CookieConfig struct {
	Domain   string `kong:"name='cookie-domain',default='localhost'"`
	Secure   bool   `kong:"name='cookie-secure',default='false'"`
	SameSite string `kong:"name='cookie-samesite',default='lax'"`
}

type Authenticator struct {
	verifier *pkgauth.Verifier
	cookie   CookieConfig
	access   time.Duration
	refresh  time.Duration
}

func NewAuthenticator(verifier *pkgauth.Verifier, cookie CookieConfig, accessTTL, refreshTTL time.Duration) *Authenticator {
	return &Authenticator{verifier: verifier, cookie: cookie, access: accessTTL, refresh: refreshTTL}
}

// SetAuthCookies issues the access + refresh cookies after a successful login.
func (a *Authenticator) SetAuthCookies(w http.ResponseWriter, access, refresh string) {
	http.SetCookie(w, a.cookie.build(AccessCookie, access, a.access))
	http.SetCookie(w, a.cookie.build(RefreshCookie, refresh, a.refresh))
}

// ClearAuthCookies expires both cookies — used by logout.
func (a *Authenticator) ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, a.cookie.build(AccessCookie, "", -time.Hour))
	http.SetCookie(w, a.cookie.build(RefreshCookie, "", -time.Hour))
}

func (c CookieConfig) build(name, value string, ttl time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   c.Domain,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: parseSameSite(c.SameSite),
	}
}

func parseSameSite(s string) http.SameSite {
	switch s {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// ConnectInterceptor reads the access cookie, verifies the JWT, and stuffs
// the resulting actor into ctx. Unauthenticated calls are still permitted to
// pass through — handlers gate themselves with actor.Required.
func (a *Authenticator) ConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if claims := a.claimsFromHeader(req.Header()); claims != nil {
				ctx = actor.WithActor(ctx, actor.Actor{
					UserID: claims.UserID,
					Email:  claims.Email,
					Name:   claims.Name,
					Roles:  claims.Roles,
				})
			}
			return next(ctx, req)
		}
	}
}

func (a *Authenticator) claimsFromHeader(h http.Header) *pkgauth.Claims {
	cookie := readCookie(h.Get("Cookie"), AccessCookie)
	if cookie == "" {
		return nil
	}
	claims, err := a.verifier.Verify(cookie)
	if err != nil || claims.Kind != "access" {
		return nil
	}
	return claims
}

func readCookie(header, name string) string {
	// Lightweight cookie parser; net/http parses through *http.Request only.
	for len(header) > 0 {
		var part string
		if idx := indexByte(header, ';'); idx >= 0 {
			part, header = header[:idx], header[idx+1:]
		} else {
			part, header = header, ""
		}
		part = trimSpace(part)
		if eq := indexByte(part, '='); eq >= 0 {
			if part[:eq] == name {
				return part[eq+1:]
			}
		}
	}
	return ""
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

// ErrUnauthenticated is returned by Require when no actor is on ctx.
var ErrUnauthenticated = errors.New("unauthenticated")
