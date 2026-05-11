// Package obs is a tiny observability bootstrap. It configures zerolog and
// exposes a Connect interceptor that logs each RPC. The OTel exporter hook
// is left as a no-op so the package has no heavyweight dependencies; wire
// in your tracing stack of choice at the application level.
package obs

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	ServiceName string `kong:"-"`
	Level       string `kong:"name='log-level',default='info'"`
	JSON        bool   `kong:"name='log-json',default='true'"`
}

// Init configures the global zerolog logger. Returns a shutdown func that
// flushes anything buffered.
func Init(_ context.Context, cfg Config) (shutdown func(context.Context) error) {
	lvl, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var w io.Writer = os.Stderr
	if !cfg.JSON {
		w = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}
	log.Logger = zerolog.New(w).With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Logger()
	return func(context.Context) error { return nil }
}

// ConnectInterceptor logs every unary RPC with status, duration and procedure.
func ConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			res, err := next(ctx, req)
			ev := log.Info()
			if err != nil {
				ev = log.Warn().Err(err)
			}
			ev.
				Str("rpc", req.Spec().Procedure).
				Dur("took", time.Since(start)).
				Msg("rpc")
			return res, err
		}
	}
}
