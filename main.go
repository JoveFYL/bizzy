package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JoveFYL/bizzy/internal/api"
	"github.com/JoveFYL/bizzy/internal/handler"
	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/JoveFYL/bizzy/internal/worker"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	q := queue.NewMemoryQueue(100)
	r := api.NewRouter(q)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := worker.NewPool(3, q, q.Dequeue(), q.Enqueue)
	pool.RegisterHandler(model.TypeImageProcessing, handler.ProcessImageHandler)
	pool.Start(ctx)

	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	pool.Wait()
	log.Info("shutdown complete")
}
