package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	fmt.Println("Serving the web.")
	sm := http.NewServeMux()

	healthzHandler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	}
	sm.HandleFunc("/healthz", healthzHandler)
	sm.Handle("/app/*", http.StripPrefix("/app/", http.FileServer(http.Dir("html"))))

	server := http.Server{
		Handler: sm,
		Addr:    ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}

}
