package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

type returnVals struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
	Body  string `json:"body"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)))
}

func chirpHandler(w http.ResponseWriter, r *http.Request) {

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("loaded db %s\n", chirpdb.path)

	type parameters struct {
		Body string `json:"body"`
	}

	params := parameters{}

	respBody := returnVals{}

	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&params)
		if err != nil {
			fmt.Printf("Error decoding parameters: %s\n", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(201)
		chirp, err := chirpdb.CreateChirp(params.Body)
		respBody.ID = chirp.ID
		fmt.Printf("Added chirp: %s, err %s\n", chirp.Body, err)

	} else if r.Method == "GET" {

		chirps, err := chirpdb.GetChirps()
		if err != nil {
			fmt.Printf("Error getting chirps: %s", err)
		}
		fmt.Println(chirps)
		respondWithJSON(w, http.StatusOK, chirps)

		return
	} else {
		fmt.Printf("Method %s not allowed\n", r.Method)
		return
	}

	if len(params.Body) > 140 {
		respBody.Error = "Chirp is too long"
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	respBody.Body = cleanupBadWords(params.Body)

	respondWithJSON(w, http.StatusOK, respBody)
}

func getChirpByID(w http.ResponseWriter, r *http.Request) {

	pathVal := r.PathValue("id")
	chirpID, err := strconv.Atoi(pathVal)
	if err != nil {
		fmt.Printf("Cannot convert %s to integer\n", pathVal)
	}

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}

	chirp, err := chirpdb.GetChirp(chirpID)
	if err != nil {
		msg := fmt.Sprintf("Chirp does not exist: %s", err)
		respondWithError(w, 404, msg)
		return
	}

	fmt.Printf("Getting chirp %v\n", chirpID)
	retVals := returnVals{
		ID:   chirpID,
		Body: chirp.Body,
	}

	respondWithJSON(w, http.StatusOK, retVals)
}

func userHandler(w http.ResponseWriter, r *http.Request) {

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("loaded db %s\n", chirpdb.path)

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}

	type returnVals struct {
		ID    int    `json:"id"`
		Error string `json:"error"`
		Email string `json:"email"`
	}
	respBody := returnVals{}

	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&params)
		if err != nil {
			fmt.Printf("Error decoding parameters: %s\n", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(201)
		encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), 4)
		if err != nil {
			fmt.Printf("Error generating password: %s\n", err)
			w.WriteHeader(500)
			return
		}
		user, err := chirpdb.CreateUser(params.Email, encryptedPassword)
		respBody.ID = user.ID
		fmt.Printf("Added user: %s, err %s\n", user.Email, err)

	} else if r.Method == "GET" {

		users, err := chirpdb.GetUsers()
		if err != nil {
			fmt.Printf("Error getting users: %s", err)
		}
		fmt.Println(users)
		respondWithJSON(w, http.StatusOK, users)

		return
	} else {
		fmt.Printf("Method %s not allowed\n", r.Method)
		return
	}

	respBody.Email = params.Email

	respondWithJSON(w, http.StatusOK, respBody)
}

func getUserByID(w http.ResponseWriter, r *http.Request) {

	pathVal := r.PathValue("id")
	userID, err := strconv.Atoi(pathVal)
	if err != nil {
		fmt.Printf("Cannot convert %s to integer\n", pathVal)
	}

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}

	user, err := chirpdb.GetUser(userID)
	if err != nil {
		msg := fmt.Sprintf("User does not exist: %s", err)
		respondWithError(w, 404, msg)
		return
	}

	fmt.Printf("Getting user %v\n", user)
	retVals := returnVals{
		ID:   userID,
		Body: user.Email,
	}

	respondWithJSON(w, http.StatusOK, retVals)
}

func loginUser(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type returnVals struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
	}

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Error decoding parameters: %s\n", err)
		w.WriteHeader(500)
		return
	}

	user, err := chirpdb.GetUserByEmail(params.Email)
	if err != nil {
		msg := fmt.Sprintf("User does not exist: %s", err)
		respondWithError(w, 404, msg)
		return
	}

	retVals := returnVals{
		ID:    user.ID,
		Email: user.Email,
	}

	if err != nil {
		fmt.Println("Error decoding password")
	}
	err = bcrypt.CompareHashAndPassword(user.Password, []byte(params.Password))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Wrong password")
		return
	}
	respondWithJSON(w, http.StatusOK, retVals)

}

func cleanupBadWords(s string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	ss := strings.Split(s, " ")
	for i, x := range ss {
		if slices.Contains(badWords, strings.ToLower(x)) {
			ss[i] = "****"
		}
	}
	rss := strings.Join(ss, " ")
	return rss
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	fmt.Printf("responding with %v: %s\n", code, msg)
	type errResp struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errResp{
		Error: msg,
	})
}
