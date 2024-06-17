package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Serving the web.")

	debugOn := flag.Bool("debug", false, "path to config file")
	flag.Parse()
	fmt.Println(debugOn)
	if *debugOn {
		os.Remove("database.json")
	}

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

	// api/chirps
	sm.HandleFunc("/api/chirps", chirpHandler)
	sm.HandleFunc("GET /api/chirps/{id}", getChirpByID)
	sm.HandleFunc("DELETE /api/chirps/{id}", deleteChirp)
	// api/users
	sm.HandleFunc("/api/users", userHandler)
	sm.HandleFunc("GET /api/users/{id}", getUserByID)
	sm.HandleFunc("POST /api/login", loginUser)
	// refresh / revoke
	sm.HandleFunc("POST /api/refresh", refreshToken)
	sm.HandleFunc("POST /api/revoke", revokeToken)

	// webhook for payment/upgrading user
	sm.HandleFunc("POST /api/polka/webhooks", polkaWebhook)

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
