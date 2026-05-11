// Package outbox is a minimal transactional outbox. Domain code calls
// Enqueue(tx, ...) inside its DB transaction; a dispatcher process drains
// pending rows and publishes them onto JetStream.
//
// This base ships only the writer side. Wire your own dispatcher (or copy
// xcap-v3's apps/outbox-dispatcher) when you need at-least-once delivery.
package outbox

import (
	"time"

	"gorm.io/gorm"
)

// Event is the canonical outbox row. AutoMigrate the table once per service
// DB at boot.
type Event struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	AggregateID   string    `gorm:"type:text;not null;index"`
	EventType     string    `gorm:"type:text;not null;index"`
	Subject       string    `gorm:"type:text;not null"`
	MsgID         string    `gorm:"type:text;not null;uniqueIndex"`
	Payload       []byte    `gorm:"type:bytea;not null"`
	Headers       []byte    `gorm:"type:jsonb"`
	PublishedAt   *time.Time
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (Event) TableName() string { return "outbox_events" }

// Enqueue inserts a new outbox row in the given transaction. Caller must
// own the transaction lifecycle.
func Enqueue(tx *gorm.DB, e *Event) error {
	return tx.Create(e).Error
}

// AutoMigrate ensures the outbox_events table exists.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Event{})
}
