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
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file : %v", err)
	}
	ctx := context.Background()
	config, client, err := deploy.GetKubernetesClient(ctx, false)
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
