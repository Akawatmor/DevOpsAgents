package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kong/devopsagents/backend/internal/auth"
	"github.com/kong/devopsagents/backend/internal/middleware"
	"github.com/kong/devopsagents/backend/internal/storage"
)

func main() {
	dbPath := getEnv("DB_PATH", "./auth.db")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-me")
	port := getEnv("PORT", "8080")

	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	svc := auth.NewService(store, jwtSecret)
	h := auth.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("GET /api/me", h.Me)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.CORS(mux)

	log.Printf("backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
