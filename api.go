package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type apiConfig struct {
	fileserverHits int
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
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Body string `json:"body"`
	}

	type returnVals struct {
		ID    int    `json:"id"`
		Error string `json:"error"`
		Body  string `json:"body"`
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
