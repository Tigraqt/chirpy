package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/Tigraqt/chirpy/handlers"
	"github.com/Tigraqt/chirpy/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type apiConfig struct {
	fileserverHits int
	DB             *database.DB
	jwtSecret      string
	polkaApi       string
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load(".env")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	polkaApi := os.Getenv("POLKA")
	if polkaApi == "" {
		log.Fatal("POLKA environment variable is not set")
	}

	db, err := database.NewDB("database.json")
	if err != nil {
		log.Fatal(err)
	}

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if dbg != nil && *dbg {
		err := db.ResetDB()
		if err != nil {
			log.Fatal(err)
		}
	}

	apiCfg := apiConfig{
		fileserverHits: 0,
		DB:             db,
		jwtSecret:      jwtSecret,
		polkaApi:       polkaApi,
	}

	handlerConfig := handlers.NewHandlersConfig(apiCfg.DB, apiCfg.jwtSecret, apiCfg.polkaApi)
	metricsConfig := handlers.NewMetricsConfig(apiCfg.fileserverHits)

	fsHandler := metricsConfig.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	chirpHandler := handlers.NewChirpHandler(handlerConfig)
	userHandler := handlers.NewUserHandler(handlerConfig)
	loginHandler := handlers.NewLoginHandler(handlerConfig)
	refreshHandler := handlers.NewRefreshHandler(handlerConfig)
	webhookHandler := handlers.NewWebhookHandler(handlerConfig)

	router := chi.NewRouter()
	router.Handle("/app/*", fsHandler)
	router.Handle("/app", fsHandler)

	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handlers.HandlerReadiness)
	apiRouter.Get("/reset", metricsConfig.HandlerReset)

	apiRouter.Get("/chirps", chirpHandler.HandleGetChirps)
	apiRouter.Get("/chirps/{id}", chirpHandler.HandleGetChirp)
	apiRouter.Delete("/chirps/{id}", chirpHandler.HandleDeleteChirp)
	apiRouter.Post("/chirps", chirpHandler.HandlePostChirp)

	apiRouter.Post("/users", userHandler.HandlePostUser)
	apiRouter.Put("/users", userHandler.HandleUpdateUser)

	apiRouter.Post("/login", loginHandler.HandleLogin)

	apiRouter.Post("/refresh", refreshHandler.HandleRefresh)
	apiRouter.Post("/revoke", refreshHandler.HandleRevoke)

	apiRouter.Post("/polka/webhooks", webhookHandler.HandleWebhook)

	router.Mount("/api", apiRouter)

	adminRouter := chi.NewRouter()
	adminRouter.Get("/metrics", metricsConfig.HandlerMetrics)
	router.Mount("/admin", adminRouter)

	corsMux := middlewareCors(router)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("\nServing files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
