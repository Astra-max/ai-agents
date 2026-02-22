package handlers

import (
	"net/http"
)

func ApiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/about-page", Home)
	mux.HandleFunc("/my-sessions", Home)
	mux.HandleFunc("/demo", Home)
}