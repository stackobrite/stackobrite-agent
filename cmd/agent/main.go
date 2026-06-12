package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackobrite/stackobrite-agent/internal/auth"
	"github.com/stackobrite/stackobrite-agent/internal/config"
	"github.com/stackobrite/stackobrite-agent/internal/handler"
	"github.com/stackobrite/stackobrite-agent/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	token := cfg.AgentToken
	if token == "" {
		token, err = auth.GenerateToken()
		if err != nil {
			log.Fatalf("failed to generate token: %v", err)
		}
		log.Printf("generated agent token: %s", token)
		log.Printf("store this in /etc/stackobrite/agent.key or set AGENT_TOKEN env var")
	}

	hmacAuth := auth.New(token)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/version", handler.VersionHandler)
	mux.HandleFunc("/exec", handler.Exec)
	mux.HandleFunc("/exec/shell", handler.Shell)
	mux.HandleFunc("/exec/stream", handler.Stream)
	mux.HandleFunc("/kubeconfig", handler.Kubeconfig(cfg.Kubeconfig))
	mux.HandleFunc("/update", handler.SelfUpdate("/usr/local/bin"))

	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	var h http.Handler = mux
	h = middleware.AuditLog(h)
	h = middleware.RequestID(h)
	h = rateLimiter.Middleware(h)
	h = hmacAuth.Middleware(h)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("stackobrite-agent %s starting on %s", handler.Version, addr)
	log.Printf("kubeconfig path: %s", cfg.Kubeconfig)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down agent...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("agent stopped")
}
