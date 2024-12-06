package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharetube/server/internal/controller"
	"github.com/sharetube/server/internal/repository/connection/inmemory"
	"github.com/sharetube/server/internal/repository/room/redis"
	roomS "github.com/sharetube/server/internal/service/room"
	"github.com/sharetube/server/pkg/redisclient"
)

type AppConfig struct {
	Host            string
	Port            int
	MembersLimit    int
	PlaylistLimit   int
	UpdatesInterval time.Duration
	LogLevel        string
	RedisPort       int
	RedisHost       string
	RedisPassword   string `json:"-"`
}

// todo: add validation
func (cfg *AppConfig) Validate() error {
	if cfg.MembersLimit < 1 {
		return fmt.Errorf("members limit must be greater than 0")
	}
	if cfg.PlaylistLimit < 1 {
		return fmt.Errorf("playlist limit must be greater than 0")
	}
	if cfg.UpdatesInterval < 1 {
		return fmt.Errorf("updates interval must be greater than 0")
	}
	return nil
}

func Run(ctx context.Context, cfg *AppConfig) error {
	rc, err := redisclient.NewRedisClient(&redisclient.Config{
		Port:     cfg.RedisPort,
		Host:     cfg.RedisHost,
		Password: cfg.RedisPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}
	defer rc.Close()

	roomRepo := redis.NewRepo(rc)
	connectionRepo := inmemory.NewRepo()
	roomService := roomS.NewService(roomRepo, connectionRepo, cfg.MembersLimit, cfg.PlaylistLimit)
	controller := controller.NewController(roomService)
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
