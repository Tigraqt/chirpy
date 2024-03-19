package handlers

import "github.com/Tigraqt/chirpy/internal/database"

type HandlersConfig struct {
	DB        *database.DB
	JwtSecret string
	PolkaApi  string
}

type MetricsConfig struct {
	fileserverHits int
}

func NewHandlersConfig(db *database.DB, jwtSecret, polkaApi string) *HandlersConfig {
	return &HandlersConfig{
		DB:        db,
		JwtSecret: jwtSecret,
		PolkaApi:  polkaApi,
	}
}

func NewMetricsConfig(fileserverHits int) *MetricsConfig {
	return &MetricsConfig{
		fileserverHits: fileserverHits,
	}
}
