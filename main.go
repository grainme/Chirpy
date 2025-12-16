package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/grainme/Chirpy/handlers"
	"github.com/grainme/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	filepathRoot = "."
	port         = "8080"
)

func handler() http.Handler {
	return http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("couldn't read .env files: %s", err)
	}

	secretToken := os.Getenv("JWT_SecretToken")
	if secretToken == "" {
		log.Fatal("JWT_SecretToken must be set")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed opening database: %s", err)
	}
	defer db.Close()
	dbQueries := database.New(db)

	apiCfg := handlers.ApiConfig{
		FileServerHits: atomic.Int32{},
		Db:             dbQueries,
		Platform:       platform,
		JWTSecretToken: secretToken,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.MiddlewareMetricsInc(handler()))
	mux.HandleFunc("GET /admin/metrics", apiCfg.HandlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.HandlerReset)
	mux.HandleFunc("GET /api/healthz", handlers.HandlerReadiness)
	mux.HandleFunc("POST /api/users", apiCfg.HandlerInsertUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.HandlerValidateAndSaveChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.HandlerGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", apiCfg.HandlerGetChirpById)
	mux.HandleFunc("POST /api/login", apiCfg.HandlerUserLogin)

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
