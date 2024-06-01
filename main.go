package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Serving the web.")
	sm := http.NewServeMux()
	sm.Handle("/", http.FileServer(http.Dir("html")))
	server := http.Server{
		Handler: sm,
		Addr:    ":8080",
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}

}
