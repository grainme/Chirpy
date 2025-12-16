package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grainme/Chirpy/internal/auth"
	"github.com/grainme/Chirpy/internal/database"
)

const (
	maxBodyLength = 140
)

type chirpsParams struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *ApiConfig) HandlerGetChirpById(w http.ResponseWriter, r *http.Request) {
	chirpUUID, err := uuid.Parse(r.PathValue("chirpId"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("%s", err))
		return
	}

	chirp, err := cfg.Db.GetChirpById(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("%s", err))
		return
	}
	respondWithJson(w, http.StatusOK, chirpsParams{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func (cfg *ApiConfig) HandlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.Db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("%s", err))
		return
	}

	chirpsMapped := make([]chirpsParams, 0, len(chirps))
	for _, chirp := range chirps {
		chirpsMapped = append(chirpsMapped, chirpsParams{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
	respondWithJson(w, http.StatusOK, chirpsMapped)
}

func (cfg *ApiConfig) HandlerValidateAndSaveChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("failed to get Bearer token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	userID, err := auth.ValidateJWT(bearerToken, cfg.JWTSecretToken)
	if err != nil {
		log.Printf("failed to validate JWT token: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Unauthorized to proceed with the request")
		return
	}

	// deserializing r.body (json) into parameters
	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("JSON Decode error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if params.Body == "" {
		respondWithError(w, http.StatusBadRequest, "Chirp body cannot be empty")
	} else if len(params.Body) > maxBodyLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
	} else {
		// THIS IS A VALID CHIRP
		// filtering logic based on requirements
		cleanedBody := make([]string, 0)
		for word := range strings.FieldsSeq(params.Body) {
			redFlag := []string{"kerfuffle", "sharbert", "fornax"}
			if slices.Contains(redFlag, strings.ToLower(word)) {
				cleanedBody = append(cleanedBody, "****")
			} else {
				cleanedBody = append(cleanedBody, word)
			}
		}

		chirp, err := cfg.Db.CreateChirp(r.Context(), database.CreateChirpParams{
			ID:     uuid.New(),
			Body:   strings.Join(cleanedBody, " "),
			UserID: userID,
		})
		if err != nil {
			log.Printf("failed to create chirp: %v", err)
			log.Printf("[debug] Params: %v", params)
			log.Printf("[debug] Database chirp: %v", chirp)
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("%s", err))
			return
		}

		respondWithJson(w, http.StatusCreated, chirpsParams{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
}

func respondWithJson(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("JSON marshall error: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnErr struct {
		Err string `json:"error"`
	}
	res := returnErr{
		Err: msg,
	}
	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("JSON marshall error: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
