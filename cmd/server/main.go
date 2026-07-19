package main

import (
	"log"
	"net/http"
	"os"

	"manga-drama-studio/internal/api"
	"manga-drama-studio/internal/runner"
	"manga-drama-studio/internal/store"
)

func main() {
	address := env("APP_ADDR", ":8080")
	dataFile := env("APP_DATA_FILE", "./data/studio.json")
	webDir := env("APP_WEB_DIR", "./web/dist")
	data, err := store.Open(dataFile)
	if err != nil {
		log.Fatal(err)
	}
	broker := runner.NewBroker()
	runtime := runner.New(data, runner.MockExecutor{}, broker)
	server := api.New(data, runtime, broker, os.Getenv("APP_SECRET_KEY"), webDir)
	log.Printf("Manga Drama Studio listening on %s", address)
	if err := http.ListenAndServe(address, server.Handler()); err != nil {
		log.Fatal(err)
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
