package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	fmt.Println("Serving the web.")

	apiCfg := apiConfig{
		fileserverHits: 0,
	}

	sm := http.NewServeMux()

	// healthz
	healthzHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	}
	sm.HandleFunc("GET /api/healthz", healthzHandler)

	// metrics
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("Hits: %v", apiCfg.fileserverHits))
	}
	sm.HandleFunc("GET /api/metrics", metricsHandler)

	//reset
	sm.HandleFunc("/api/reset", apiCfg.resetHandler)

	// validateChirp
	sm.HandleFunc("POST /api/validate_chirp", vChirpHandler)

	// admin metrics
	adminMetricsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", apiCfg.fileserverHits))
	}
	sm.HandleFunc("GET /admin/metrics", adminMetricsHandler)

	// app
	appHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("html"))))
	sm.Handle("/app/", appHandler)

	server := http.Server{
		Handler: sm,
		Addr:    ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}

}
