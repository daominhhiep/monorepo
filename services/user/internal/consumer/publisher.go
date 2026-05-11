// Package consumer publishes user-domain events onto JetStream.
//
// Naming follows the convention defined in pkg/nats:
//   base.user.v1.UserRegistered
//   base.user.v1.UserUpdated
//   base.user.v1.UserDeleted
package consumer

import (
	"context"
	"fmt"

	userpb "github.com/base/base-microservice/gen/user"
	bnats "github.com/base/base-microservice/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher { return &Publisher{js: js} }

func (p *Publisher) PublishUserRegistered(ctx context.Context, userID, email, name string) error {
	ev := &userpb.UserRegisteredEvent{
		UserId:       userID,
		Email:        email,
		Name:         name,
		RegisteredAt: timestamppb.Now(),
	}
	return p.publish(ctx, "base.user.v1.UserRegistered", userID, ev)
}

func (p *Publisher) PublishUserUpdated(ctx context.Context, userID, name string, roles []string) error {
	ev := &userpb.UserUpdatedEvent{
		UserId:    userID,
		Name:      name,
		Roles:     roles,
		UpdatedAt: timestamppb.Now(),
	}
	return p.publish(ctx, "base.user.v1.UserUpdated", userID, ev)
}

func (p *Publisher) PublishUserDeleted(ctx context.Context, userID string) error {
	ev := &userpb.UserDeletedEvent{UserId: userID, DeletedAt: timestamppb.Now()}
	return p.publish(ctx, "base.user.v1.UserDeleted", userID, ev)
}

func (p *Publisher) publish(ctx context.Context, subject, aggregateID string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	msgID := fmt.Sprintf("%s:%s", subject, aggregateID)
	return bnats.Publish(ctx, p.js, subject, msgID, data)
}
