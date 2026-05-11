// Package repo is the data-access layer for the user service.
package repo

import (
	"context"
	"errors"

	"github.com/base/base-microservice/services/user/internal/models"
	"gorm.io/gorm"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already registered")
)

type Repo struct{ db *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(ctx context.Context, u *models.User) error {
	tx := r.db.WithContext(ctx).Create(u)
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			return ErrEmailConflict
		}
		return tx.Error
	}
	return nil
}

func (r *Repo) GetByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	tx := r.db.WithContext(ctx).First(&u, "id = ?", id)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, tx.Error
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	tx := r.db.WithContext(ctx).First(&u, "email = ?", email)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, tx.Error
}

func (r *Repo) List(ctx context.Context, limit int, cursor string) ([]models.User, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Order("created_at ASC, id ASC").Limit(limit + 1)
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	var rows []models.User
	if err := q.Find(&rows).Error; err != nil {
		return nil, "", err
	}
	var next string
	if len(rows) > limit {
		next = rows[limit-1].ID
		rows = rows[:limit]
	}
	return rows, next, nil
}

func (r *Repo) Update(ctx context.Context, u *models.User) error {
	tx := r.db.WithContext(ctx).Save(u)
	return tx.Error
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	tx := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// isUniqueViolation matches Postgres SQLSTATE 23505 without pulling in pgconn.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "SQLSTATE 23505") || contains(s, "unique constraint")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
