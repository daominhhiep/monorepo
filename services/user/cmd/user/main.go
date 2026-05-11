package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/base/base-microservice/gen/user/userconnect"
	"github.com/base/base-microservice/pkg/actor"
	pkgauth "github.com/base/base-microservice/pkg/auth"
	"github.com/base/base-microservice/pkg/config"
	"github.com/base/base-microservice/pkg/db"
	bnats "github.com/base/base-microservice/pkg/nats"
	"github.com/base/base-microservice/pkg/obs"
	"github.com/base/base-microservice/services/user/internal/consumer"
	"github.com/base/base-microservice/services/user/internal/handler"
	"github.com/base/base-microservice/services/user/internal/models"
	"github.com/base/base-microservice/services/user/internal/repo"
	"github.com/base/base-microservice/services/user/internal/service"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	HTTPListen string `kong:"name='http-listen',default=':8001'"`
	DB         db.Config       `kong:"embed,prefix='db-'"`
	NATS       bnats.Config    `kong:"embed,prefix='nats-'"`
	Obs        obs.Config      `kong:"embed,prefix='obs-'"`
	JWT        pkgauth.Config  `kong:"embed,prefix='jwt-'"`
}

func main() {
	var cfg Config
	config.Parse("user", &cfg)
	cfg.Obs.ServiceName = "user"

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownObs := obs.Init(ctx, cfg.Obs)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownObs(ctx)
	}()

	gormDB, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("open db")
	}
	if cfg.DB.AutoMigrate {
		if err := models.AutoMigrate(gormDB); err != nil {
			log.Fatal().Err(err).Msg("auto-migrate")
		}
	}

	nc, js, err := bnats.Connect(cfg.NATS)
	if err != nil {
		log.Fatal().Err(err).Msg("connect nats")
	}
	defer nc.Close()
	if err := bnats.EnsureStream(ctx, js); err != nil {
		log.Warn().Err(err).Msg("ensure stream (continuing)")
	}

	issuer, err := pkgauth.NewIssuer(cfg.JWT)
	if err != nil {
		log.Fatal().Err(err).Msg("jwt issuer")
	}

	r := repo.New(gormDB)
	pub := consumer.NewPublisher(js)
	svc := service.New(r, pub, issuer)
	h := handler.New(svc)

	mux := http.NewServeMux()
	path, connectHandler := userconnect.NewUserServiceHandler(h,
		connect.WithInterceptors(obs.ConnectInterceptor(), actor.ConnectInterceptor()),
	)
	mux.Handle(path, connectHandler)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := db.Ping(gormDB); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
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
		log.Info().Str("addr", cfg.HTTPListen).Msg("user service listening")
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
