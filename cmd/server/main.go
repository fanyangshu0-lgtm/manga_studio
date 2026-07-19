package main

import (
	"context"
	"log"
	"net/http"

	"manga-drama-studio/internal/api"
	"manga-drama-studio/internal/assets"
	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/generation"
	"manga-drama-studio/internal/media"
	"manga-drama-studio/internal/newapi"
	"manga-drama-studio/internal/runner"
	"manga-drama-studio/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	var data store.Repository
	if cfg.Database.Host != "" {
		data, err = store.OpenMySQL(context.Background(), cfg.Database)
	} else {
		data, err = store.Open(cfg.DataFile)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()
	if err := data.MarkActiveRunsInterrupted(context.Background()); err != nil {
		log.Fatal(err)
	}
	broker := runner.NewBroker()
	var executor runner.Executor = runner.MockExecutor{}
	if cfg.Executor == "production" {
		client := newapi.New(cfg.NewAPI.BaseURL, cfg.NewAPI.Token, nil)
		production := runner.NewProductionExecutor(generation.NewPlanner(client, cfg.NewAPI.ScriptModel))
		production.ConfigureVideo(client, assets.New(cfg.AssetDir, cfg.Video.MaxDownloadBytes, nil), data,
			cfg.NewAPI.FastVideoModel, cfg.NewAPI.QualityVideoModel, cfg.NewAPI.DefaultVideoModel,
			cfg.Video.MaxConcurrency, cfg.Video.PollInterval, cfg.Video.TaskTimeout)
		composer := media.NewComposer()
		if err := composer.Check(context.Background()); err != nil {
			log.Fatal(err)
		}
		production.ConfigureComposer(composer)
		executor = production
	}
	runtime := runner.New(data, executor, broker)
	server := api.New(data, runtime, broker, cfg)
	if err := server.BootstrapProvider(context.Background(), cfg.NewAPI); err != nil {
		log.Fatal(err)
	}
	log.Printf("Manga Drama Studio listening on %s", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
