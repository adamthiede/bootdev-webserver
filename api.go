package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type MyCustomClaims struct {
	Foo string `json:"foo"`
	jwt.RegisteredClaims
}

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
	type returnVals struct {
		ID       int    `json:"id"`
		Error    string `json:"error"`
		Body     string `json:"body"`
		AuthorID int    `json:"author_id"`
	}

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
		// log in the user here

		validity, userID := IsJWTValid(w, r)
		if !validity {
			return
		} else {
			fmt.Printf("Got valid JWT for %v\n", userID)
		}

		chirp, err := chirpdb.CreateChirp(params.Body, userID)
		respBody.ID = chirp.ID
		respBody.AuthorID = chirp.AuthorID
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

func deleteChirp(w http.ResponseWriter, r *http.Request) {
	validity, userID := IsJWTValid(w, r)
	if !validity {
		respondWithError(w, 403, "Not allowed to delete chirp")
	} else {
		fmt.Printf("Got valid JWT for %v\n", userID)
	}

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

	if chirp.AuthorID == userID {
	    chirpdb.DeleteChirp(chirpID)
	} else {
	    respondWithError(w, 403, "You're only allowed to delete your own chirps")
	    return
	}

	respondWithJSON(w, 204, "")
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
	} else if r.Method == "PUT" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&params)
		if err != nil {
			fmt.Printf("Error decoding parameters: %s\n", err)
			w.WriteHeader(500)
			return
		}
		fmt.Printf("requesting to update user %s\n", params.Email)
		godotenv.Load()
		jwtSecret := os.Getenv("JWT_SECRET")
		// authorization header required
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, 401, "Authorization header required")
			return
		}
		authTokenS := strings.Split(authHeader, " ")
		authToken := authTokenS[len(authTokenS)-1]
		token, err := jwt.ParseWithClaims(authToken, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil {
			fmt.Printf("%s\n", err)
			respondWithError(w, 401, "Authorization failed: couldn't parse token")
			return
		} else if claims, ok := token.Claims.(*MyCustomClaims); ok {
			fmt.Printf("Got token: %s, %s\n", claims.Issuer, claims.Subject)
		} else {
			fmt.Printf("couldn't get token claims:%s, %t\n", err, ok)
			respondWithError(w, 401, "Authorization failed!")
			return
		}
		if token.Valid {
			userID, err := token.Claims.GetSubject()
			if err != nil {
				respondWithError(w, 401, "Authorization failed: couldn't parse token")
				return
			}
			userIDI, err := strconv.Atoi(userID)
			fmt.Printf("Got valid token for user %v\n", userIDI)
			user, err := chirpdb.GetUser(userIDI)
			if err != nil {
				erro := fmt.Sprintf("Couldn't get user from token: %s", err)
				respondWithError(w, 401, erro)
				return
			}
			fmt.Printf("Got user! %s %v \n", user.Email, user.ID)
			encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), 4)
			if err != nil {
				fmt.Printf("Error generating password: %s\n", err)
				w.WriteHeader(500)
				return
			}
			upUser, err := chirpdb.UpdateUser(userIDI, params.Email, encryptedPassword)
			if err != nil {
				erro := fmt.Sprintf("couldn't update user: %s", err)
				respondWithError(w, 500, erro)
				return
			}
			respBody.ID = upUser.ID
			respBody.Email = upUser.Email
			respondWithJSON(w, http.StatusOK, respBody)
		}
		return

	} else {
		fmt.Printf("Method %s not allowed\n", r.Method)
		return
	}

	respBody.Email = params.Email

	respondWithJSON(w, http.StatusOK, respBody)
}

func IsJWTValid(w http.ResponseWriter, r *http.Request) (bool, int) {
	userID := 0
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	// authorization header required
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, 401, "Authorization header required")
		return false, userID
	}
	authTokenS := strings.Split(authHeader, " ")
	authToken := authTokenS[len(authTokenS)-1]
	token, err := jwt.ParseWithClaims(authToken, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		fmt.Printf("%s\n", err)
		respondWithError(w, 401, "Authorization failed: couldn't parse token")
		return false, userID
	} else if claims, ok := token.Claims.(*MyCustomClaims); ok {
		userID, err = strconv.Atoi(claims.Subject)
		if err != nil {
			fmt.Printf("Error converting %s to int: %s\n", claims.Subject, err)
		}
		fmt.Printf("Got token: %s, %s\n", claims.Issuer, claims.Subject)
	} else {
		fmt.Printf("couldn't get token claims:%s, %t\n", err, ok)
		respondWithError(w, 401, "Authorization failed!")
		return false, 0
	}
	return true, userID
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

func generateToken(userID int) (string, error) {
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	claims := MyCustomClaims{
		"bar",
		jwt.RegisteredClaims{
			Issuer:   "Chirpy",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			//ExpiresAt: expire_time,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   fmt.Sprintf("%v", userID),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := jwtToken.SignedString([]byte(jwtSecret))
	return ss, err
}

func loginUser(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	type returnVals struct {
		ID           int    `json:"id"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
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

	/*
		expire_time := jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
		if params.ExpiresInSeconds != 0 {
		    expire_time = jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(params.ExpiresInSeconds)))
		}
	*/

	ss, err := generateToken(user.ID)
	if err != nil {
		fmt.Printf("Token error: %s %s\n", ss, err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't produce token")
		return
	}

	// create refresh token as random text
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		fmt.Println("error reading random data?:", err)
		return
	}
	refreshToken := hex.EncodeToString(b)
	user, err = chirpdb.AddRefreshToken(user.ID, refreshToken)
	fmt.Printf("refresh token added to %v: %s\n", user.ID, user.RefreshToken.Token)

	retVals := returnVals{
		ID:           user.ID,
		Email:        user.Email,
		Token:        ss,
		RefreshToken: refreshToken,
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

func refreshToken(w http.ResponseWriter, r *http.Request) {

	type returnVals struct {
		Token string `json:"token"`
	}

	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, 401, "Authorization header required")
		return
	}
	authTokenS := strings.Split(authHeader, " ")
	authToken := authTokenS[len(authTokenS)-1]
	fmt.Printf("You sent %s\n", authToken)

	user, err := chirpdb.GetUserByRefreshToken(authToken)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("GetUserByRefreshToken err: %s", err))
		return
	}
	token, err := generateToken(user.ID)
	retVals := returnVals{
		Token: token,
	}
	respondWithJSON(w, http.StatusOK, retVals)
}

func revokeToken(w http.ResponseWriter, r *http.Request) {
	chirpdb, err := NewDB("database.json")
	if err != nil {
		fmt.Println(err)
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, 401, "Authorization header required")
		return
	}
	authTokenS := strings.Split(authHeader, " ")
	authToken := authTokenS[len(authTokenS)-1]
	fmt.Printf("You sent %s\n", authToken)

	user, err := chirpdb.GetUserByRefreshToken(authToken)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("GetUserByRefreshToken err: %s", err))
		return
	}
	chirpdb.RevokeRefreshToken(user.ID)

	respondWithJSON(w, 204, "")
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
