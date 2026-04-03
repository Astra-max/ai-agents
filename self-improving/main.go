package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"self-improving/gemini"
	"self-improving/ws"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Initialize Gemini client factory
	geminiFactory := gemini.NewClientFactory(apiKey)

	// Initialize WebSocket handler
	wsHandler := ws.NewHandler(geminiFactory)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// WebSocket endpoint
	http.HandleFunc("/ws", wsHandler.ServeWS)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🪞 The Mirror is running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
