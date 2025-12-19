package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/grainme/Chirpy/internal/database"
)

type ApiConfig struct {
	FileServerHits atomic.Int32
	Db             *database.Queries
	Platform       string
	JWTSecretToken string
	PolkaKey       string
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
<html>
<body>
<h1>Welcome, Chirpy Admin</h1>
<p>Chirpy has been visited %d times!</p>
</body>
</html>`, cfg.FileServerHits.Load())
}
