// Package models declares the GORM schema for the user service.
package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email        string    `gorm:"type:citext;uniqueIndex;not null"`
	Name         string    `gorm:"type:text;not null"`
	PasswordHash string    `gorm:"type:text;not null"`
	Roles        string    `gorm:"type:text;not null;default:''"` // comma-separated; small + simple
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`
}

func (User) TableName() string { return "users" }

func (u *User) RolesSlice() []string {
	if u.Roles == "" {
		return nil
	}
	parts := strings.Split(u.Roles, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (u *User) SetRoles(roles []string) {
	u.Roles = strings.Join(roles, ",")
}

func AllModels() []any { return []any{&User{}} }

// AutoMigrate enables required extensions and creates the schema.
// Dev only — production should use SQL migrations under migrations/.
func AutoMigrate(db *gorm.DB) error {
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return err
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "citext"`).Error; err != nil {
		return err
	}
	return db.AutoMigrate(AllModels()...)
}
