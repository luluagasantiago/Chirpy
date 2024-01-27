package main

import (
	"log"
	"net/http"
)

func main() {
	//look: serveMux implements the interface Handler ()
	mux := http.NewServeMux()
	// mux is a hanlder bc it has an implementation of the ServeHTTP function
	corsMux := middlewareCors(mux) //
	// http.HandlerFunc -> type HandlerFunc func(ResponseWriter, *Request)
	server := http.Server{
		Handler: corsMux,
	}
	mux.Handle("/", http.FileServer(http.Dir(".")))

	// by default listens to port 80
	log.Fatal(server.ListenAndServe())

}

/*
ListenAndServe starts an HTTP server with a given address and handler.
 The handler is usually nil, which means to use DefaultServeMux.
  Handle and HandleFunc add handlers to DefaultServeMux:


#What is a Handler?

A Handler responds to an HTTP request.

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

ServeHTTP should write reply headers and data to the ResponseWriter and then return.
Returning signals that the request is finished; it is not valid to use the ResponseWriter or read from the Request.
Body after or concurrently with the completion of the ServeHTTP call.

*/

/*

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
*/

func middlewareCors(next http.Handler) http.Handler {
	/*
		The HandlerFunc type is an adapter to allow the use
		of ordinary functions as HTTP handlers.
		 If f is a function with the appropriate signature,
		 -> HandlerFunc(f) is a Handler that calls f.

		- type HcandlerFunc func(ResponseWriter, *Request)

	*/ //---------
	// The HandlerFunc type is an adapter to allow the use of
	// ordinary functions as HTTP handlers. If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler that calls f.
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
		//whose pattern most closely matches the request URL.
	})
}
