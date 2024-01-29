package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	const filepathRoot = "."
	const port = "8080"
	//look: serveMux implements the interface Handler ()
	apcfg := &apiConfig{fileserverHits: 0}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apcfg.middlewareMetricsInc(http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		//w.WriteHeader(200)
		w.Write([]byte(http.StatusText(200)))

	})
	mux.HandleFunc("/metrics", apcfg.middlewareCountRequests)
	mux.HandleFunc("/reset", apcfg.middlewareResetCounts)

	// mux is a hanlder bc it has an implementation of the ServeHTTP function
	corsMux := middlewareCors(mux) //
	// http.HandlerFunc -> type HandlerFunc func(ResponseWriter, *Request) -> Handler interface
	server := http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

/*
#What is a Handler?

A Handler responds to an HTTP request.

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

*/

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
		fmt.Println("HOLA")
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareCountRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	//w.WriteHeader(200)
	fmt.Println("HOLA1")
	w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits)))

}

func (cfg *apiConfig) middlewareResetCounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileserverHits = 0
	w.WriteHeader(200)
	fmt.Println("HOLA")

}
