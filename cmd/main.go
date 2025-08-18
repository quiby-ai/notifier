package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quiby-ai/notifier/config"
	"github.com/quiby-ai/notifier/internal/consumer"
	"github.com/quiby-ai/notifier/internal/registry"
	"github.com/quiby-ai/notifier/internal/ws"
)

func main() {
	cfg := config.Load()

	hub := registry.NewHub(cfg)
	go hub.Run()

	cons := consumer.NewKafkaConsumer(cfg, hub)
	ctx, cancel := context.WithCancel(context.Background())
	go cons.Run(ctx)

	mux := http.NewServeMux()
	mux.Handle("/ws", ws.WSHandler(hub, cfg))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("healthz write error: %v", err)
		}
	})

	srv := &http.Server{Addr: cfg.HTTP.Addr, Handler: mux}
	go func() {
		log.Printf("notifier listening on %s", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	shCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shCtx)
	hub.Close()
}
