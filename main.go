package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"webhook/pkg/deploy"
	serve "webhook/pkg/server"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	ctx := context.Background()
	config, client, err := deploy.GetKubernetesClient(ctx)
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	serverInstance := serve.Server{
		Client:           client,
		Config:           config,
		WebhookSecretKey: os.Getenv("WEBHOOK_SECRET"),
		GithubClient:     deploy.GetGitHubClient(ctx, os.Getenv("GITHUB_TOKEN")),
	}

	http.HandleFunc("/webhook", serverInstance.Webhook)

	err = http.ListenAndServe(":8080", nil)
	log.Printf("Exited: %s\n", err)
}
