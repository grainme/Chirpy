package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grainme/Chirpy/internal/auth"
	"github.com/grainme/Chirpy/internal/database"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	JWTtoken     string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}
type parameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

	token, err := auth.MakeJWT(user.ID, cfg.JWTSecretToken, time.Hour)
	if err != nil {
		log.Printf("Could not make JWT Token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Could not generate Refresh Token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	refreshTokenCreated, err := cfg.Db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		log.Printf("Could not store Refresh Token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	respondWithJson(w, http.StatusOK, User{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		JWTtoken:     token,
		RefreshToken: refreshTokenCreated.Token,
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

func (cfg *ApiConfig) HandlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("%v", err)
		respondWithError(w, http.StatusUnauthorized, "Email and password not found")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Password hashing failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash user's password")
		return
	}

	authorizationValue := strings.Fields(r.Header.Get("Authorization"))
	if len(authorizationValue) < 2 {
		log.Printf("Authorization header: %v", authorizationValue)
		respondWithError(w, http.StatusUnauthorized, "Bearer token not provided")
		return
	}

	bearerToken := authorizationValue[1]
	userID, err := auth.ValidateJWT(bearerToken, cfg.JWTSecretToken)
	if err != nil {
		log.Printf("%v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT")
		return
	}

	updatedUser, err := cfg.Db.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	})
	if err != nil {
		log.Printf("%v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't update user")
		return
	}

	respondWithJson(w, http.StatusOK, User{
		ID:        updatedUser.ID,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
		Email:     updatedUser.Email,
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

func (cfg *ApiConfig) HandlerRefresh(w http.ResponseWriter, r *http.Request) {
	authorizationValue := strings.Fields(r.Header.Get("Authorization"))
	if len(authorizationValue) < 2 {
		log.Printf("Authorization header: %v", authorizationValue)
		respondWithError(w, http.StatusBadRequest, "Bearer token not provided")
		return
	}

	bearerToken := authorizationValue[1]
	refreshTokenRow, err := cfg.Db.GetRefreshToken(r.Context(), bearerToken)
	if err != nil || refreshTokenRow.ExpiresAt.Before(time.Now()) || refreshTokenRow.RevokedAt.Valid {
		log.Printf("Refresh token not found or expired: %v", authorizationValue)
		respondWithError(w, http.StatusUnauthorized, "Refresh token not found or expired")
		return
	}

	newJWT, err := auth.MakeJWT(refreshTokenRow.UserID, cfg.JWTSecretToken, time.Hour)
	if err != nil {
		log.Printf("Could not make JWT Token: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	token := struct {
		Token string `json:"token"`
	}{
		Token: newJWT,
	}
	respondWithJson(w, http.StatusOK, token)
}

func (cfg *ApiConfig) HandlerRevoke(w http.ResponseWriter, r *http.Request) {
	authorizationValue := strings.Fields(r.Header.Get("Authorization"))
	if len(authorizationValue) < 2 {
		log.Printf("Authorization header: %v", authorizationValue)
		respondWithError(w, http.StatusBadRequest, "Bearer token not provided")
		return
	}

	bearerToken := authorizationValue[1]
	err := cfg.Db.UpdateRefreshToken(r.Context(), bearerToken)
	if err != nil {
		log.Printf("Could not update refresh token: %v", authorizationValue)
		respondWithError(w, http.StatusInternalServerError, "Could not update refresh token")
		return
	}

	respondWithJson(w, http.StatusNoContent, nil)
}
