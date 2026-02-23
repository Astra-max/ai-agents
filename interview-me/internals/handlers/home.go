package handlers

import (
	"fmt"
	"net/http"
)

/**
 * Handles home
 */
func Home(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "%s a %v page!", "This is a ", req.URL)
}