package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/grainme/Chirpy/internal/auth"
	"github.com/grainme/Chirpy/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	JWTtoken  string    `json:"token"`
}
type parameters struct {
	Email            string         `json:"email"`
	Password         string         `json:"password"`
	ExpiresInSeconds *time.Duration `json:"expires_in_seconds"`
}

func (cfg *ApiConfig) HandlerUserLogin(w http.ResponseWriter, r *http.Request) {
	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("JSON Decode error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	// get user by email
	user, errMail := cfg.Db.GetUserByEmail(r.Context(), params.Email)
	match, errPassword := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if errMail != nil || errPassword != nil || !match {
		log.Printf("Incorrect email or password: \n%v\n%v", errMail, errPassword)
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	defaultExpiry := time.Hour
	if params.ExpiresInSeconds == nil {
		params.ExpiresInSeconds = &defaultExpiry
	} else if *params.ExpiresInSeconds > time.Hour {
		params.ExpiresInSeconds = &defaultExpiry
	}

	token, err := auth.MakeJWT(user.ID, cfg.JWTSecretToken, *params.ExpiresInSeconds)
	if err != nil {
		log.Printf("Could not make JWT Token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	respondWithJson(w, http.StatusOK, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		JWTtoken:  token,
	})
}

func (cfg *ApiConfig) HandlerInsertUser(w http.ResponseWriter, r *http.Request) {
	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("JSON Decode error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Password hashing failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash user's password")
		return
	}

	dbData, err := cfg.Db.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		Email:          params.Email,
		HashedPassword: hash,
	})
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't create the user")
		return
	}

	respondWithJson(w, http.StatusCreated, User{
		ID:        dbData.ID,
		CreatedAt: dbData.CreatedAt,
		UpdatedAt: dbData.UpdatedAt,
		Email:     dbData.Email,
	})
}

func (cfg *ApiConfig) HandlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		respondWithError(w, http.StatusForbidden, "This is not permissible in a non-dev env")
		return
	}

	cfg.FileServerHits.Store(0)
	err := cfg.Db.DeleteAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete all users")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0 and database reset to initial state."))
}
