package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharetube/server/internal/controller"
	"github.com/sharetube/server/internal/service"
	"golang.org/x/exp/slog"
)

type AppConfig struct {
	Host            string
	Port            int
	MembersLimit    int
	PlaylistLimit   int
	UpdatesInterval time.Duration
	LogLevel        string
}

func Run(ctx context.Context, cfg *AppConfig) error {
	roomService := service.NewRoomService(cfg.UpdatesInterval, cfg.MembersLimit, cfg.PlaylistLimit)
	controller := controller.NewController(roomService)
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), Handler: controller.Mux()}

	// graceful shutdown.
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
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-serverCtx.Done()

	return nil
}
