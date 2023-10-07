package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/preslavmarinov/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits int
	DB             *database.DB
	jwtSecret      string
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load(".env")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment varibale is not set")
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
	}

	router := chi.NewRouter()
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	router.Handle("/app", fsHandler)
	router.Handle("/app/*", fsHandler)

	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handlerReadiness)
	apiRouter.Get("/reset", apiCfg.handlerReset)

	apiRouter.Post("/login", apiCfg.handlerLogin)

	apiRouter.Post("/users", apiCfg.handlerUsersCreate)
	apiRouter.Put("/users", apiCfg.handlerUsersUpdate)

	apiRouter.Post("/chirps", apiCfg.handlerChirpsCreate)
	apiRouter.Get("/chirps", apiCfg.handlerChirpsRetrieve)
	apiRouter.Get("/chirps/{chirpID}", apiCfg.handlerChirpsGet)
	router.Mount("/api", apiRouter)

	adminMetrics := chi.NewRouter()
	adminMetrics.Get("/metrics", apiCfg.handlerMetrics)
	router.Mount("/admin", adminMetrics)

	corsMux := middlewareCors(router)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
