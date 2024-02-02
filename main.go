package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	const filepathRoot = "."
	const port = "8080"
	router := chi.NewRouter()

	apcfg := apiConfig{
		fileserverHits: 0,
	}

	fsHandler := apcfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	router.Handle("/app/*", fsHandler)
	router.Handle("/app", fsHandler)

	apiRouter := chi.NewRouter()

	apiRouter.Get("/reset", apcfg.handlerReset)
	apiRouter.Get("/healthz", http.HandlerFunc(handlerReadiness))
	// Decided to decoupled the app from the api
	// non-website endpoints will go to the /api namespace.
	router.Mount("/api", apiRouter)

	adminRouter := chi.NewRouter()

	//apcfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	//adminRouter.Get("/metrics", apcfg.handlerMetrics(FileServer(http.Dir(filepathMetrics))))
	adminRouter.Get("/metrics", http.HandlerFunc(apcfg.handlerMetrics))
	router.Mount("/admin", adminRouter)

	corsMux := middlewareCors(router) //
	server := http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

// Here the middleware is just setting some headers and then passing the handler along with next.ServeHTTP()
func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//this wrapper sets the headers for the ResponseWriter
		// and calls ServeHTTP(w,r)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
		//ServeHTTP dispatches the request to the handler
		//whose pattern most closely matches the request URL from request.
	})
}

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		fmt.Println("Increment req....")
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Println("Count req...")

	const tpl = `<html>

	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
	
	</html>
	`

	w.Write([]byte(fmt.Sprintf(tpl, cfg.fileserverHits)))

}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileserverHits = 0
	fmt.Println("Reset req...")
	w.Write([]byte("Hits reset to 0"))

}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(http.StatusText(http.StatusOK)))

}
