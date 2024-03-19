package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Tigraqt/chirpy/internal/auth"
	"github.com/Tigraqt/chirpy/internal/database"
)

type WebhookHandler struct {
	DB        *database.DB
	JwtSecret string
	PolkaApi  string
}

func NewWebhookHandler(cfg *HandlersConfig) *WebhookHandler {
	return &WebhookHandler{DB: cfg.DB, JwtSecret: cfg.JwtSecret, PolkaApi: cfg.PolkaApi}
}

func (cfg *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID int `json:"user_id"`
		}
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find api key")
		return
	}
	if apiKey != cfg.PolkaApi {
		respondWithError(w, http.StatusUnauthorized, "API key is invalid")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusOK, struct{}{})
		return
	}

	_, err = cfg.DB.UpgradeChirpyRed(params.Data.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotExist) {
			respondWithError(w, http.StatusNotFound, "Couldn't find user")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update user")
		return
	}

	respondWithJSON(w, http.StatusOK, struct{}{})
}
