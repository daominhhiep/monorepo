// Package handler implements the Connect-RPC service surface, translating
// proto requests/responses into calls on the application service layer.
package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	userpb "github.com/base/base-microservice/gen/user"
	"github.com/base/base-microservice/gen/user/userconnect"
	"github.com/base/base-microservice/services/user/internal/models"
	"github.com/base/base-microservice/services/user/internal/repo"
	"github.com/base/base-microservice/services/user/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	userconnect.UnimplementedUserServiceHandler
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(ctx context.Context, req *connect.Request[userpb.RegisterRequest]) (*connect.Response[userpb.RegisterResponse], error) {
	m := req.Msg
	u, err := h.svc.Register(ctx, service.RegisterInput{Email: m.GetEmail(), Name: m.GetName(), Password: m.GetPassword()})
	if err != nil {
		if errors.Is(err, repo.ErrEmailConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&userpb.RegisterResponse{User: toProto(u)}), nil
}

func (h *Handler) Login(ctx context.Context, req *connect.Request[userpb.LoginRequest]) (*connect.Response[userpb.LoginResponse], error) {
	out, err := h.svc.Login(ctx, req.Msg.GetEmail(), req.Msg.GetPassword())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&userpb.LoginResponse{
		User:         toProto(out.User),
		AccessToken:  out.Access,
		RefreshToken: out.Refresh,
	}), nil
}

func (h *Handler) GetCurrentUser(ctx context.Context, req *connect.Request[userpb.GetCurrentUserRequest]) (*connect.Response[userpb.GetCurrentUserResponse], error) {
	// Identity is carried via X-Forwarded-User; the actor package extracts it.
	uid := req.Header().Get("X-Forwarded-User")
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no actor"))
	}
	u, err := h.svc.Get(ctx, uid)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&userpb.GetCurrentUserResponse{User: toProto(u)}), nil
}

func (h *Handler) ListUsers(ctx context.Context, req *connect.Request[userpb.ListUsersRequest]) (*connect.Response[userpb.ListUsersResponse], error) {
	users, next, err := h.svc.List(ctx, int(req.Msg.GetPageSize()), req.Msg.GetPageToken())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*userpb.User, len(users))
	for i := range users {
		out[i] = toProto(&users[i])
	}
	return connect.NewResponse(&userpb.ListUsersResponse{Users: out, NextPageToken: next}), nil
}

func (h *Handler) UpdateUser(ctx context.Context, req *connect.Request[userpb.UpdateUserRequest]) (*connect.Response[userpb.UpdateUserResponse], error) {
	u, err := h.svc.Update(ctx, req.Msg.GetId(), req.Msg.GetName(), req.Msg.GetRoles())
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&userpb.UpdateUserResponse{User: toProto(u)}), nil
}

func (h *Handler) DeleteUser(ctx context.Context, req *connect.Request[userpb.DeleteUserRequest]) (*connect.Response[userpb.DeleteUserResponse], error) {
	if err := h.svc.Delete(ctx, req.Msg.GetId()); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&userpb.DeleteUserResponse{}), nil
}

func toProto(u *models.User) *userpb.User {
	if u == nil {
		return nil
	}
	return &userpb.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Roles:     u.RolesSlice(),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}
