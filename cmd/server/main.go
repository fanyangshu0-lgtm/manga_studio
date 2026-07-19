package main

import (
	"context"
	"log"
	"net/http"

	"manga-drama-studio/internal/api"
	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/runner"
	"manga-drama-studio/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	data, err := store.Open(cfg.DataFile)
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()
	if err := data.MarkActiveRunsInterrupted(context.Background()); err != nil {
		log.Fatal(err)
	}
	broker := runner.NewBroker()
	runtime := runner.New(data, runner.MockExecutor{}, broker)
	server := api.New(data, runtime, broker, cfg.SecretKey, cfg.WebDir)
	log.Printf("Manga Drama Studio listening on %s", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
