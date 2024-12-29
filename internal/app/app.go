package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sharetube/server/internal/controller"
	"github.com/sharetube/server/internal/repository/connection/inmemory"
	"github.com/sharetube/server/internal/repository/room/redis"
	"github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/ctxlogger"
	"github.com/sharetube/server/pkg/redisclient"
)

type AppConfig struct {
	Secret        string `json:"-"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	MembersLimit  int    `json:"members_limit"`
	PlaylistLimit int    `json:"playlist_limit"`
	LogLevel      string `json:"log_level"`
	RedisPort     int    `json:"redis_port"`
	RedisHost     string `json:"redis_host"`
	RedisPassword string `json:"-"`
}

// todo: add validation
func (cfg *AppConfig) Validate() error {
	if cfg.MembersLimit < 1 {
		return fmt.Errorf("members limit must be greater than 0")
	}
	if cfg.PlaylistLimit < 1 {
		return fmt.Errorf("playlist limit must be greater than 0")
	}
	return nil
}

func Run(ctx context.Context, cfg *AppConfig) error {
	logLevel := slog.LevelInfo
	if err := logLevel.UnmarshalText([]byte(strings.ToUpper(cfg.LogLevel))); err != nil {
		log.Fatal(err)
	}

	h := ctxlogger.ContextHandler{
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: true,
		}),
	}

	logger := slog.New(&h)

	rc, err := redisclient.NewRedisClient(&redisclient.Config{
		Port:     cfg.RedisPort,
		Host:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	defer rc.Close()

	roomRepo := redis.NewRepo(rc, 24*14*time.Hour)
	connectionRepo := inmemory.NewRepo()
	roomService := room.NewService(roomRepo, connectionRepo, &room.Config{
		MembersLimit:  cfg.MembersLimit,
		PlaylistLimit: cfg.PlaylistLimit,
		Secret:        cfg.Secret,
		RoomExp:       30 * time.Second,
	})
	controller := controller.NewController(roomService, logger)
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), Handler: controller.GetMux()}

	// graceful shutdown
	serverCtx, serverStopCtx := context.WithCancel(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, c := context.WithTimeout(serverCtx, 30*time.Second)
		defer c()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	slog.InfoContext(serverCtx, "starting server", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-serverCtx.Done()

	return nil
}
