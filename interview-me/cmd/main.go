package main

import (
	"app/internals/config"
	"app/internals/handlers"
	"log"
	"net/http"
	"strings"
)

func main() {
	/**
	 * Handles start
	 */
	Start()
}

/**
 * Handles start
 */
func Start() {
	mux := http.NewServeMux()
	c := config.Load()

	fs := http.FileServer(http.Dir("front-app"))
	mux.Handle("/", fs)
	handlers.ApiRoutes(mux)
	log.Printf("server listening on port %s\n", strings.TrimPrefix(c.Port, ":"))
	err := http.ListenAndServe(
		c.Port,
		mux,
	)

	if err != nil {
		log.Println("Failed to connect to server")
	}
}
