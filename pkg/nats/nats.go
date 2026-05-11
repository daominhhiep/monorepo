// Package nats centralises JetStream conventions used across the platform.
//
// Stream layout:
//   - 1 unified stream named "base"
//   - Subjects: base.<context>.v1.<EventType>
//   - Per-consumer durables: <consumer>-<context>
//   - Dedup via `Nats-Msg-Id` header = "<event>:<aggregate>:<version>"
package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Config struct {
	URL string `kong:"name='url',default='nats://localhost:4222'"`
}

func Connect(cfg Config) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("connect nats: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("init jetstream: %w", err)
	}
	return nc, js, nil
}

// EnsureStream creates or updates the `base` stream with the default
// subject filter. Call once at process start in services that publish.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "base",
		Subjects:  []string{"base.>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	return err
}

// DurableConsumer is a thin wrapper around CreateOrUpdateConsumer for the
// per-context filter subject pattern.
type DurableConsumerOptions struct {
	StreamName    string
	Durable       string
	FilterSubject string
	MaxDeliver    int
	AckWait       time.Duration
}

func CreateOrUpdateConsumer(ctx context.Context, js jetstream.JetStream, opts DurableConsumerOptions) (jetstream.Consumer, error) {
	if opts.StreamName == "" {
		opts.StreamName = "base"
	}
	if opts.MaxDeliver == 0 {
		opts.MaxDeliver = 10
	}
	if opts.AckWait == 0 {
		opts.AckWait = 30 * time.Second
	}
	if opts.Durable == "" {
		return nil, errors.New("durable name required")
	}
	return js.CreateOrUpdateConsumer(ctx, opts.StreamName, jetstream.ConsumerConfig{
		Durable:       opts.Durable,
		FilterSubject: opts.FilterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    opts.MaxDeliver,
		AckWait:       opts.AckWait,
	})
}

// Publish wraps js.Publish with the Nats-Msg-Id dedup header set.
func Publish(ctx context.Context, js jetstream.JetStream, subject, msgID string, payload []byte) error {
	msg := &nats.Msg{Subject: subject, Data: payload, Header: nats.Header{}}
	if msgID != "" {
		msg.Header.Set(nats.MsgIdHdr, msgID)
	}
	_, err := js.PublishMsg(ctx, msg)
	return err
}
