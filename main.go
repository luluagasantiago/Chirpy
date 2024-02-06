package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/luluagasantiago/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits int
	jwtSecret      string
	database       *database.DB
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")

	}
	jwtSecret1 := os.Getenv("JWT_SECRET")

	const filepathRoot = "."
	const port = "8080"
	router := chi.NewRouter()
	db, err := database.NewDB("database.json")

	if err != nil {
		fmt.Print("Unable to create db", err)
	}

	apcfg := apiConfig{
		fileserverHits: 0,
		jwtSecret:      jwtSecret1,
		database:       db,
	}

	fsHandler := apcfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	router.Handle("/app/*", fsHandler)
	router.Handle("/app", fsHandler)

	apiRouter := chi.NewRouter()

	apiRouter.Get("/reset", apcfg.handlerReset)
	apiRouter.Get("/healthz", handlerReadiness)
	//apiRouter.Post("/validate_chirp", handlerValidateChirp)
	chirpsPostHandler := middlewarePostChirps(db)
	apiRouter.Post("/chirps", chirpsPostHandler)
	chirpsGetHandler := middlewareGetChirps(db)
	apiRouter.Get("/chirps", chirpsGetHandler)
	chirpsGetByIdHandler := middlewareGetChirpsById(db)
	apiRouter.Get("/chirps/{chirpid}", chirpsGetByIdHandler)

	usersPostHandler := middlewarePostUsers(db)
	apiRouter.Post("/users", usersPostHandler)

	apiRouter.Post("/login", apcfg.loginPostHandler)
	apiRouter.Put("/users", apcfg.usersPutHandler)

	router.Mount("/api", apiRouter)

	adminRouter := chi.NewRouter()

	//apcfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	//adminRouter.Get("/metrics", apcfg.handlerMetrics(FileServer(http.Dir(filepathMetrics))))
	adminRouter.Get("/metrics", apcfg.handlerMetrics)
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

func middlewarePostChirps(db *database.DB) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		type parameters struct {
			Body string `json:"body"`
		}
		type returnVals struct {
			Id   int    `json:"id"`
			Body string `json:"body"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
			return
		}
		const maxChirpLenght = 140
		if len(params.Body) > maxChirpLenght {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		chirp, err := db.CreateChirp(params.Body)
		if err != nil {
			//fmt.Printf("Error when creating chirp in db %v\n", err)
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error when creating chirp in db %v\n", err))
			return
		}

		respondWithJSON(w, http.StatusCreated, chirp)

	})

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

func clean_string(words string) string {
	bad_words := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	splitWords := strings.Split(words, " ")
	new_string := []string{}
	for _, val := range splitWords {
		loweredWord := strings.ToLower(val)
		isBadWord := slices.Contains[[]string, string](bad_words, loweredWord)
		if isBadWord {
			new_string = append(new_string, "****")
		} else {
			new_string = append(new_string, val)
		}

	}

	return strings.Join(new_string, " ")

}

func middlewareGetChirps(db *database.DB) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		chirps, err := db.GetChirps()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error retrieving chirps from db %v\n", err))
			return

		}
		respondWithJSON(w, http.StatusOK, chirps)

	})

}

func middlewareGetChirpsById(db *database.DB) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirps, err := db.GetChirps()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error retrieving chirps from db %v\n", err))
			return
		} //func URLParam(r *http.Request, key string) string
		id := chi.URLParam(r, "chirpid")
		id_int, err := strconv.Atoi(id)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error converting str(id)  %v\n", err))

		}
		for _, chirp := range chirps {
			if chirp.Id == id_int {
				respondWithJSON(w, http.StatusOK, chirp)
				return
			}
		}

		respondWithError(w, http.StatusNotFound, fmt.Sprint("Not Found\n"))

	})

}

func middlewarePostUsers(db *database.DB) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
			return
		}

		user, err := db.CreateUser(params.Email, params.Password)
		if err != nil {
			//fmt.Printf("Error when creating chirp in db %v\n", err)
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error when creating user in db %v\n", err))
			return
		}

		respondWithJSON(w, http.StatusCreated, user)

	})

}

func (apiCfg *apiConfig) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email              string `json:"email"`
		Password           string `json:"password"`
		Expires_in_seconds int    `json:"expires_in_seconds,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error()+"DEC PARAMS")
		return
	}

	userNoPass, err := apiCfg.database.UserLookUp(params.Email, params.Password)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error()+" LookUP error")
		return
	}

	// Create JWT
	issuedAt := time.Now().UTC()
	//exp := issuedAt.Add(time.Duration(params.Expires_in_seconds))
	// when the client doesn't sets the expire_in_second, it is 0 by default,
	// in that case we change that to 24 hs
	if params.Expires_in_seconds == 0 {
		params.Expires_in_seconds = 24 * 3600
	}
	expires := issuedAt.Add(time.Second * time.Duration(params.Expires_in_seconds))

	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expires),
		Subject:   fmt.Sprint(userNoPass.Id),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tPrinf, _ := token.Claims.GetExpirationTime()
	fmt.Printf("\ntoken will expire at: %v\n", tPrinf)
	signedToken, err := token.SignedString([]byte(apiCfg.jwtSecret))

	if err != nil {
		fmt.Print("\n\n\nCouldn't sign the token\n\n\n")
		respondWithError(w, http.StatusInternalServerError, err.Error()+"Error signing TOken"+signedToken)
		return
	}

	userWithToken := struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}{
		Id:    userNoPass.Id,
		Email: userNoPass.Email,
		Token: signedToken,
	}

	respondWithJSON(w, http.StatusOK, userWithToken)

}

func (apiCfg *apiConfig) usersPutHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error()+"DEC PARAMS")
		return
	}

	t := strings.TrimSpace(strings.Split(r.Header.Get("Authorization"), " ")[1])
	fmt.Printf("\n\ntoken: %v %[1]T:\n\n", t)

	type customClaims struct {
		jwt.RegisteredClaims
	}

	tokenStruct, err := jwt.ParseWithClaims(t, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(apiCfg.jwtSecret), nil
	})

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	/*
		userWithToken := struct {
			Id    int    `json:"id"`
			Email string `json:"email"`
			Token string `json:"token"`
		}{
			Id:    userNoPass.Id,
			Email: userNoPass.Email,
			Token: signedToken,

		}*/

	subId, err := tokenStruct.Claims.GetSubject()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	subIdInt, err := strconv.Atoi(subId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updatedUser, err := apiCfg.database.UpdateUser(subIdInt, params.Email, params.Password)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, updatedUser)

}
