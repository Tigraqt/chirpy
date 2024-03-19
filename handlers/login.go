package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Tigraqt/chirpy/internal/auth"
	"github.com/Tigraqt/chirpy/internal/database"
)

type LoginHandler struct {
	DB        *database.DB
	jwtSecret string
}

func NewLoginHandler(cfg *HandlersConfig) *LoginHandler {
	return &LoginHandler{
		DB:        cfg.DB,
		jwtSecret: cfg.JwtSecret,
	}
}

func (cfg *LoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is empty")
		return
	}

	if params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Password is empty")
		return
	}

	user, err := cfg.DB.GetUserByEmail(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user")
		return
	}

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	accessTokenExpiration := 60 * 60
	refreshTokenExpiration := 60 * 60 * 24 * 60

	accessToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, auth.TokenTypeAccess, time.Duration(accessTokenExpiration)*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create access token")
		return
	}

	refreshToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, auth.TokenTypeRefresh, time.Duration(refreshTokenExpiration)*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create refresh token")
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
}
