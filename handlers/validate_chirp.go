package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
)

const (
	maxBodyLength = 140
)

func HandlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVal struct {
		CleanedBody string `json:"cleaned_body"`
		Valid       bool   `json:"valid"`
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
		respondWithJson(w, http.StatusOK, returnVal{CleanedBody: strings.Join(cleanedBody, " "), Valid: true})
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
