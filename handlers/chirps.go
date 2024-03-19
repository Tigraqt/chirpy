package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Tigraqt/chirpy/internal/auth"
	"github.com/Tigraqt/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
)

type ChirpHandler struct {
	DB        *database.DB
	jwtSecret string
}

type Chirp struct {
	ID       int    `json:"id"`
	Body     string `json:"body"`
	AuthorID int    `json:"author_id"`
}

func NewChirpHandler(cfg *HandlersConfig) *ChirpHandler {
	return &ChirpHandler{DB: cfg.DB, jwtSecret: cfg.JwtSecret}
}

func (h *ChirpHandler) HandlePostChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT")
		return
	}
	subject, err := auth.ValidateJWT(token, h.jwtSecret)
	if err != nil {
		println("err00: ", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT")
		return
	}

	userID, err := strconv.Atoi(subject)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse user ID")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if params.Body == "" {
		respondWithError(w, http.StatusBadRequest, "Body is empty")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := cleanBody(params.Body)

	chirp, err := h.DB.CreateChirp(cleanedBody, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create chirp")
		return
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		ID:       chirp.ID,
		Body:     chirp.Body,
		AuthorID: chirp.AuthorID,
	})
}

func (h *ChirpHandler) HandleGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := h.DB.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps")
		return
	}

	authorID := -1
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString != "" {
		authorID, err = strconv.Atoi(authorIDString)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author ID")
			return
		}
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		if authorID != -1 && dbChirp.AuthorID != authorID {
			continue
		}

		chirps = append(chirps, Chirp{
			ID:       dbChirp.ID,
			AuthorID: dbChirp.AuthorID,
			Body:     dbChirp.Body,
		})
	}

	sortString := r.URL.Query().Get("sort")

	sort.Slice(chirps, func(i, j int) bool {
		if sortString == "desc" {
			return chirps[i].ID > chirps[j].ID
		}

		return chirps[i].ID < chirps[j].ID
	})

	respondWithJSON(w, http.StatusOK, chirps)
}

func (h *ChirpHandler) HandleGetChirp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if s, err := strconv.Atoi(id); err == nil {
		chirp, err := h.DB.GetChirp(s)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		respondWithJSON(w, http.StatusOK, chirp)
	} else {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
	}
}

func (h *ChirpHandler) HandleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpIDString := chi.URLParam(r, "id")

	chirpID, err := strconv.Atoi(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT")
		return
	}
	subject, err := auth.ValidateJWT(token, h.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT")
		return
	}
	userID, err := strconv.Atoi(subject)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse user ID")
		return
	}

	dbChirp, err := h.DB.GetChirp(chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get chirp")
		return
	}
	if dbChirp.AuthorID != userID {
		respondWithError(w, http.StatusForbidden, "You can't delete this chirp")
		return
	}

	err = h.DB.DeleteChirp(chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete chirp")
		return
	}

	respondWithJSON(w, http.StatusOK, struct{}{})
}

func cleanBody(body string) string {
	wordsToReplace := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	words := strings.Fields(body)

	for i, word := range words {
		_, found := wordsToReplace[strings.ToLower(word)]
		if found {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
