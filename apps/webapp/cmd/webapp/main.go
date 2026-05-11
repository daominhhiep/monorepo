package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/base/base-microservice/apps/webapp/internal"
	"github.com/base/base-microservice/gen/apps/webapp/v1/webappv1connect"
	pkgauth "github.com/base/base-microservice/pkg/auth"
	"github.com/base/base-microservice/pkg/config"
	"github.com/base/base-microservice/pkg/obs"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	HTTPListen string                `kong:"name='http-listen',default=':8082'"`
	Cookie     internal.CookieConfig `kong:"embed"`
	Backends   internal.BackendURLs  `kong:"embed"`
	JWT        pkgauth.Config        `kong:"embed,prefix='jwt-'"`
	Obs        obs.Config            `kong:"embed,prefix='obs-'"`
}

func main() {
	var cfg Config
	config.Parse("webapp", &cfg)
	cfg.Obs.ServiceName = "webapp"

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownObs := obs.Init(ctx, cfg.Obs)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownObs(ctx)
	}()

	issuer, err := pkgauth.NewIssuer(cfg.JWT)
	if err != nil {
		log.Fatal().Err(err).Msg("jwt issuer")
	}
	verifier := pkgauth.NewVerifier(cfg.JWT)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	backends := internal.NewBackends(httpClient, cfg.Backends)
	auth := internal.NewAuthenticator(verifier, cfg.Cookie, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	h := internal.NewHandler(backends, auth, issuer)

	mux := http.NewServeMux()
	path, connectHandler := webappv1connect.NewAPIServiceHandler(h,
		connect.WithInterceptors(obs.ConnectInterceptor(), auth.ConnectInterceptor()),
	)
	mux.Handle(path, internal.CookieMiddleware(connectHandler))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	srv := &http.Server{
		Addr:              cfg.HTTPListen,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Info().Str("addr", cfg.HTTPListen).Msg("webapp bff listening")
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})
	if err := g.Wait(); err != nil {
		log.Fatal().Err(err).Msg("server exit")
	}
}
