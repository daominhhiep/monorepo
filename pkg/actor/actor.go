// Package actor carries the authenticated principal between processes via
// forward-auth HTTP headers (X-Forwarded-User, X-Forwarded-Email,
// X-Forwarded-Roles). The BFF authenticates the user and injects these
// headers when calling backend services; backend services trust them and
// expose the principal through context.
//
// Trust model: these headers MUST be stripped at the ingress edge. They are
// only safe within the mesh. Never accept them from the public internet.
package actor

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

const (
	HeaderUserID = "X-Forwarded-User"
	HeaderEmail  = "X-Forwarded-Email"
	HeaderName   = "X-Forwarded-Name"
	HeaderRoles  = "X-Forwarded-Roles"
)

// Actor is the authenticated principal as seen by a backend service.
type Actor struct {
	UserID string
	Email  string
	Name   string
	Roles  []string
}

func (a Actor) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type ctxKey struct{}

// WithActor stores the actor in ctx.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

// FromContext extracts the actor; the second return is false when absent.
func FromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(ctxKey{}).(Actor)
	return a, ok
}

// FromHeaders builds an Actor from inbound HTTP headers.
func FromHeaders(h http.Header) (Actor, bool) {
	id := h.Get(HeaderUserID)
	if id == "" {
		return Actor{}, false
	}
	roles := strings.Split(h.Get(HeaderRoles), ",")
	cleaned := roles[:0]
	for _, r := range roles {
		if r = strings.TrimSpace(r); r != "" {
			cleaned = append(cleaned, r)
		}
	}
	return Actor{
		UserID: id,
		Email:  h.Get(HeaderEmail),
		Name:   h.Get(HeaderName),
		Roles:  cleaned,
	}, true
}

// SetHeaders writes the actor onto outbound HTTP headers.
func (a Actor) SetHeaders(h http.Header) {
	h.Set(HeaderUserID, a.UserID)
	if a.Email != "" {
		h.Set(HeaderEmail, a.Email)
	}
	if a.Name != "" {
		h.Set(HeaderName, a.Name)
	}
	if len(a.Roles) > 0 {
		h.Set(HeaderRoles, strings.Join(a.Roles, ","))
	}
}

// ConnectInterceptor reads forward-auth headers off inbound Connect
// requests and stuffs the resulting Actor into ctx. Unary + streaming.
func ConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if a, ok := FromHeaders(req.Header()); ok {
				ctx = WithActor(ctx, a)
			}
			return next(ctx, req)
		}
	}
}

// ForwardInterceptor is an outbound Connect interceptor that pulls the
// actor from ctx and writes forward-auth headers onto the outgoing request.
// Use this on Connect clients held by a BFF.
func ForwardInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if a, ok := FromContext(ctx); ok {
				a.SetHeaders(req.Header())
			}
			return next(ctx, req)
		}
	}
}

// Required returns a connect.CodeUnauthenticated error when no actor is
// present in ctx. Use at the top of handlers that require auth.
func Required(ctx context.Context) (Actor, error) {
	a, ok := FromContext(ctx)
	if !ok {
		return Actor{}, connect.NewError(connect.CodeUnauthenticated, errMissingActor)
	}
	return a, nil
}

var errMissingActor = &actorErr{msg: "actor missing from context"}

type actorErr struct{ msg string }

func (e *actorErr) Error() string { return e.msg }
