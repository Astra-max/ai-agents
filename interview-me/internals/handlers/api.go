package handlers

import (
	"net/http"
)

/**
 * Handles api routes
 */
 
func ApiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/about-page", Home)
	mux.HandleFunc("/my-sessions", Home)
	mux.HandleFunc("/demo", Home)
}