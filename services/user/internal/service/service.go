// Package service is the application layer for the user service:
// it orchestrates the repo, password hashing, and event publishing.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgauth "github.com/base/base-microservice/pkg/auth"
	"github.com/base/base-microservice/services/user/internal/models"
	"github.com/base/base-microservice/services/user/internal/repo"
	"github.com/google/uuid"
)

type Publisher interface {
	PublishUserRegistered(ctx context.Context, userID, email, name string) error
	PublishUserUpdated(ctx context.Context, userID, name string, roles []string) error
	PublishUserDeleted(ctx context.Context, userID string) error
}

type Service struct {
	repo *repo.Repo
	pub  Publisher
	jwt  *pkgauth.Issuer
}

func New(r *repo.Repo, p Publisher, j *pkgauth.Issuer) *Service {
	return &Service{repo: r, pub: p, jwt: j}
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	if in.Email == "" || in.Name == "" || len(in.Password) < 8 {
		return nil, errors.New("invalid input")
	}
	hash, err := pkgauth.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &models.User{
		ID:           uuid.NewString(),
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	if s.pub != nil {
		_ = s.pub.PublishUserRegistered(ctx, u.ID, u.Email, u.Name)
	}
	return u, nil
}

type LoginOutput struct {
	User    *models.User
	Access  string
	Refresh string
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (s *Service) Login(ctx context.Context, email, password string) (*LoginOutput, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := pkgauth.CheckPassword(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	access, refresh, err := s.jwt.Issue(u.ID, u.Email, u.Name, u.RolesSlice())
	if err != nil {
		return nil, err
	}
	return &LoginOutput{User: u, Access: access, Refresh: refresh}, nil
}

func (s *Service) Get(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, pageSize int, pageToken string) ([]models.User, string, error) {
	return s.repo.List(ctx, pageSize, pageToken)
}

func (s *Service) Update(ctx context.Context, id, name string, roles []string) (*models.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		u.Name = name
	}
	if roles != nil {
		u.SetRoles(roles)
	}
	u.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	if s.pub != nil {
		_ = s.pub.PublishUserUpdated(ctx, u.ID, u.Name, u.RolesSlice())
	}
	return u, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.pub != nil {
		_ = s.pub.PublishUserDeleted(ctx, id)
	}
	return nil
}
