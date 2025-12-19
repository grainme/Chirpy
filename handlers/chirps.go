package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sort"
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
	chirpUUID, err := uuid.Parse(r.PathValue("chirpID"))
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

func (cfg *ApiConfig) HandlerDeleteChirpById(w http.ResponseWriter, r *http.Request) {
	header := strings.Fields(r.Header.Get("Authorization"))
	if len(header) < 2 {
		respondWithError(w, http.StatusUnauthorized, "Bearer token is missing")
		return
	}

	bearerToken := header[1]
	userID, err := auth.ValidateJWT(bearerToken, cfg.JWTSecretToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("%v", err))
		return
	}

	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusForbidden, fmt.Sprintf("%s", err))
		return
	}

	chirp, err := cfg.Db.GetChirpById(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Couldn't get chirp: %v", err))
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, fmt.Sprintf("You can't delete this chirp: %v", err))
		return
	}

	err = cfg.Db.DeleteChirpById(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusForbidden, fmt.Sprintf("%s", err))
		return
	}
	respondWithJson(w, http.StatusNoContent, nil)
}

func (cfg *ApiConfig) HandlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	var chirps []database.Chirp
	var err error

	author_id := r.URL.Query().Get("author_id")
	sort_type := r.URL.Query().Get("sort")

	if author_id != "" {
		userId, err := uuid.Parse(author_id)
		if err != nil {
			log.Printf("Failed to fetch all chirps: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Could not parse userID into UUID format")
			return
		}
		chirps, err = cfg.Db.GetChirpByUserId(r.Context(), userId)
	} else {
		chirps, err = cfg.Db.GetAllChirps(r.Context())
		if err != nil {
			log.Printf("Failed to fetch all chirps: %v", err)
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("%s", err))
			return
		}
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

	if sort_type == "desc" {
		sort.Slice(chirpsMapped, func(i, j int) bool {
			return chirpsMapped[i].CreatedAt.After(chirpsMapped[j].CreatedAt)
		})
	} else {
		sort.Slice(chirpsMapped, func(i, j int) bool {
			return chirpsMapped[i].CreatedAt.Before(chirpsMapped[j].CreatedAt)
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
		log.Printf("%v", err)
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
