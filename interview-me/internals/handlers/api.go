package handlers

import (
	"net/http"
)

func ApiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api", Home)
}