package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"hitalent-go-task/internal/config"
	"hitalent-go-task/internal/db"
	"hitalent-go-task/internal/httpapi"
	"hitalent-go-task/internal/service"
)

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("%s %s %s", r.Method, r.URL.RequestURI(), time.Since(start))
	})
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func main() {
	// -- Logger --
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// -- Configs preload --
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}

	// -- Connect to DB --
	database, err := db.Connect(cfg)
	if err != nil {
		logger.Fatalf("database connection error: %v", err)
	}

	departmentService := service.NewDepartmentService(database)
	handler := httpapi.NewHandler(departmentService, logger)

	// -- Router --
	mux := http.NewServeMux()
	mux.Handle("/departments", handler)
	mux.Handle("/departments/", handler)
	mux.HandleFunc("/healthcheck", healthcheck)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           loggingMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// -- Startup --
	logger.Printf("starting server, listening to port %s...", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("server failed: %v", err)
	}
}
