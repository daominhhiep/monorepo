package internal

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	webappv1 "github.com/base/base-microservice/gen/apps/webapp/v1"
	"github.com/base/base-microservice/gen/apps/webapp/v1/webappv1connect"
	userpb "github.com/base/base-microservice/gen/user"
	"github.com/base/base-microservice/pkg/actor"
	pkgauth "github.com/base/base-microservice/pkg/auth"
)

// Handler implements the BFF APIService. It writes auth cookies on
// login/register/logout, and forwards reads to backend services with the
// session's principal attached via X-Forwarded-* headers.
type Handler struct {
	webappv1connect.UnimplementedAPIServiceHandler
	backends *Backends
	auth     *Authenticator
	issuer   *pkgauth.Issuer
}

func NewHandler(b *Backends, auth *Authenticator, issuer *pkgauth.Issuer) *Handler {
	return &Handler{backends: b, auth: auth, issuer: issuer}
}

// httpWriter lets handlers reach the underlying ResponseWriter to set cookies.
// Connect ordinarily abstracts the writer away — we smuggle it through ctx via
// a wrapper middleware (see ServeMux setup in main).
type ctxRespKey struct{}

func withResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, ctxRespKey{}, w)
}
func responseWriterFromCtx(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(ctxRespKey{}).(http.ResponseWriter)
	return w, ok
}

// CookieMiddleware wraps the Connect mux so handlers can set Set-Cookie headers.
func CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(withResponseWriter(r.Context(), w))
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Register(ctx context.Context, req *connect.Request[webappv1.RegisterRequest]) (*connect.Response[webappv1.RegisterResponse], error) {
	resp, err := h.backends.User.Register(ctx, connect.NewRequest(&userpb.RegisterRequest{
		Email:    req.Msg.GetEmail(),
		Name:     req.Msg.GetName(),
		Password: req.Msg.GetPassword(),
	}))
	if err != nil {
		return nil, err
	}
	loginResp, err := h.backends.User.Login(ctx, connect.NewRequest(&userpb.LoginRequest{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
	}))
	if err != nil {
		return nil, err
	}
	h.setCookiesFromCtx(ctx, loginResp.Msg.GetAccessToken(), loginResp.Msg.GetRefreshToken())
	return connect.NewResponse(&webappv1.RegisterResponse{
		Principal: principalFromUserPB(resp.Msg.GetUser()),
	}), nil
}

func (h *Handler) Login(ctx context.Context, req *connect.Request[webappv1.LoginRequest]) (*connect.Response[webappv1.LoginResponse], error) {
	resp, err := h.backends.User.Login(ctx, connect.NewRequest(&userpb.LoginRequest{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
	}))
	if err != nil {
		return nil, err
	}
	h.setCookiesFromCtx(ctx, resp.Msg.GetAccessToken(), resp.Msg.GetRefreshToken())
	return connect.NewResponse(&webappv1.LoginResponse{
		Principal: principalFromUserPB(resp.Msg.GetUser()),
	}), nil
}

func (h *Handler) Logout(ctx context.Context, _ *connect.Request[webappv1.LogoutRequest]) (*connect.Response[webappv1.LogoutResponse], error) {
	if w, ok := responseWriterFromCtx(ctx); ok {
		h.auth.ClearAuthCookies(w)
	}
	return connect.NewResponse(&webappv1.LogoutResponse{}), nil
}

func (h *Handler) GetSession(ctx context.Context, _ *connect.Request[webappv1.GetSessionRequest]) (*connect.Response[webappv1.GetSessionResponse], error) {
	a, ok := actor.FromContext(ctx)
	if !ok {
		return connect.NewResponse(&webappv1.GetSessionResponse{Authenticated: false}), nil
	}
	return connect.NewResponse(&webappv1.GetSessionResponse{
		Authenticated: true,
		Principal: &webappv1.Principal{
			UserId: a.UserID,
			Email:  a.Email,
			Name:   a.Name,
			Roles:  a.Roles,
		},
	}), nil
}

func (h *Handler) ListUsers(ctx context.Context, req *connect.Request[webappv1.ListUsersRequest]) (*connect.Response[webappv1.ListUsersResponse], error) {
	if _, err := actor.Required(ctx); err != nil {
		return nil, err
	}
	resp, err := h.backends.User.ListUsers(ctx, connect.NewRequest(&userpb.ListUsersRequest{
		PageSize:  req.Msg.GetPageSize(),
		PageToken: req.Msg.GetPageToken(),
	}))
	if err != nil {
		return nil, err
	}
	out := make([]*webappv1.UserSummary, len(resp.Msg.GetUsers()))
	for i, u := range resp.Msg.GetUsers() {
		out[i] = &webappv1.UserSummary{
			Id: u.GetId(), Email: u.GetEmail(), Name: u.GetName(),
			Roles: u.GetRoles(), CreatedAt: u.GetCreatedAt(),
		}
	}
	return connect.NewResponse(&webappv1.ListUsersResponse{Users: out, NextPageToken: resp.Msg.GetNextPageToken()}), nil
}

func principalFromUserPB(u *userpb.User) *webappv1.Principal {
	if u == nil {
		return nil
	}
	return &webappv1.Principal{
		UserId: u.GetId(), Email: u.GetEmail(),
		Name: u.GetName(), Roles: u.GetRoles(),
	}
}

func (h *Handler) setCookiesFromCtx(ctx context.Context, access, refresh string) {
	w, ok := responseWriterFromCtx(ctx)
	if !ok || access == "" {
		return
	}
	h.auth.SetAuthCookies(w, access, refresh)
}
