package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"webhook/pkg/deploy"

	"github.com/google/go-github/github"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Server struct {
	Client           dynamic.Interface
	Config           *rest.Config
	GithubClient     *github.Client
	WebhookSecretKey string
}

func (s Server) getfiles(ownerName string, repoName string, filename string) ([]byte, error) {
	content, _, _, err := s.GithubClient.Repositories.GetContents(context.Background(), ownerName, repoName, filename, &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Printf("error getting file: %v", err)
		return nil, err
	}
	cont, err := content.GetContent()
	if err != nil {
		log.Printf("error getting file: %v", err)
		return nil, err
	}
	fileBody := []byte(cont)
	return fileBody, nil
}

func (s Server) getDeletedFile(ownerName string, repoName string, filename string) ([]byte, error) {

	commits, _, err := s.GithubClient.Repositories.ListCommits(context.Background(), ownerName, repoName, &github.CommitsListOptions{})
	if err != nil {
		log.Fatalf("Error listing commits: %v", err)
	}

	var parentSHA string
	for _, commit := range commits {
		files, _, err := s.GithubClient.Repositories.GetCommit(context.Background(), ownerName, repoName, *commit.SHA)
		if err != nil {
			log.Fatalf("Error getting commit: %v", err)
		}

		fileDeleted := false
		for _, file := range files.Files {
			if *file.Filename == filename && *file.Status == "removed" {
				fileDeleted = true
				break
			}
		}

		if fileDeleted {
			parentSHA = *commit.Parents[0].SHA
			break
		}
	}

	if parentSHA == "" {
		log.Fatalf("Couldn't find a commit with the deleted file: %s", filename)
	}

	content, _, _, err := s.GithubClient.Repositories.GetContents(context.Background(), ownerName, repoName, filename, &github.RepositoryContentGetOptions{
		Ref: parentSHA,
	})
	if err != nil {
		log.Fatalf("Error getting content of deleted file: %v", err)
	}

	cont, err := content.GetContent()
	if err != nil {
		log.Fatalf("Error decoding content of deleted file: %v", err)
	}

	fileBody := []byte(cont)
	fmt.Printf("Successfully downloaded the deleted file: %s\n", filename)
	return fileBody, nil
}

func (s Server) Webhook(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	payload, err := github.ValidatePayload(req, []byte(s.WebhookSecretKey))
	if err != nil {
		log.Printf("error while validating payload: %v", err)
		w.WriteHeader(500)
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("error while parsing payload: %v", err)
		return
	}
	deploy := deploy.Deployer{
		Config: s.Config,
		Client: s.Client,
	}
	switch event := event.(type) {
	case *github.Hook:
		log.Printf("hook is created\n")
	case *github.PushEvent:
		addedFiles, modifiedFiles, deletedFiles := extractChangedFiles(event)

		for _, fileWithType := range []struct {
			Files []string
			Type  string
		}{
			{Files: addedFiles, Type: "added"},
			{Files: modifiedFiles, Type: "modified"},
			{Files: deletedFiles, Type: "deleted"},
		} {
			for _, filename := range fileWithType.Files {
				log.Printf("processing %s file: %v", fileWithType.Type, filename)

				if fileWithType.Type == "added" || fileWithType.Type == "modified" {
					fileBody, err := s.getfiles(*event.Repo.Owner.Name, *event.Repo.Name, filename)
					if err != nil {
						w.WriteHeader(500)
						return
					}

					if fileWithType.Type == "added" {
						err = deploy.AddedFile(ctx, fileBody)
						if err != nil {
							log.Printf("error while deploying the added file: %v", err)
						}
					} else {
						err = deploy.ModifiedFile(ctx, fileBody)
						if err != nil {
							log.Printf("error while deploying the modified file: %v", err)
						}

					}
				}

				if fileWithType.Type == "deleted" {
					fileBody, err := s.getDeletedFile(*event.Repo.Owner.Name, *event.Repo.Name, filename)
					if err != nil {
						w.WriteHeader(500)
						return
					}
					err = deploy.DeletedFile(ctx, fileBody)
					if err != nil {
						log.Printf("error while deletting the file: %v", err)
					}
				}
				log.Printf("Deploy of %s (%s) finished\n", filename, fileWithType.Type)
			}
		}
	default:
		w.WriteHeader(500)
		log.Printf("Event not found: %s", event)
		return
	}
}

func extractChangedFiles(event *github.PushEvent) ([]string, []string, []string) {
	var addedFiles, modifiedFiles, deletedFiles []string
	uniqueAdded := make(map[string]bool)
	uniqueModified := make(map[string]bool)
	uniqueDeleted := make(map[string]bool)

	for _, commit := range event.Commits {
		for _, added := range commit.Added {
			uniqueAdded[added] = true
		}
		for _, modified := range commit.Modified {
			uniqueModified[modified] = true
		}
		for _, removed := range commit.Removed {
			uniqueDeleted[removed] = true
		}
	}

	for file := range uniqueAdded {
		println(file + " added")
		addedFiles = append(addedFiles, file)
	}
	for file := range uniqueModified {
		println(file + " modified")
		modifiedFiles = append(modifiedFiles, file)
	}
	for file := range uniqueDeleted {
		println(file + " deleted")
		deletedFiles = append(deletedFiles, file)
	}
	return addedFiles, modifiedFiles, deletedFiles
}
