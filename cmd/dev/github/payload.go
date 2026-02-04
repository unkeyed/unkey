package github

import (
	"fmt"
	"time"
)

type pushPayloadInput struct {
	Branch         string
	CommitSHA      string
	CommitMessage  string
	InstallationID int64
	RepositoryID   int64
	Repository     string
	AuthorName     string
	AuthorUsername string
}

type pushPayload struct {
	Ref          string           `json:"ref"`
	After        string           `json:"after"`
	Installation pushInstallation `json:"installation"`
	Repository   pushRepository   `json:"repository"`
	Commits      []pushCommit     `json:"commits"`
	HeadCommit   *pushCommit      `json:"head_commit"`
	Sender       pushSender       `json:"sender"`
}

type pushInstallation struct {
	ID int64 `json:"id"`
}

type pushRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type pushCommit struct {
	ID        string           `json:"id"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
	Author    pushCommitAuthor `json:"author"`
}

type pushCommitAuthor struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type pushSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

func buildPushPayload(input pushPayloadInput) pushPayload {
	timestamp := time.Now().Format(time.RFC3339)

	commit := pushCommit{
		ID:        input.CommitSHA,
		Message:   input.CommitMessage,
		Timestamp: timestamp,
		Author: pushCommitAuthor{
			Name:     input.AuthorName,
			Username: input.AuthorUsername,
		},
	}

	return pushPayload{
		Ref:   fmt.Sprintf("refs/heads/%s", input.Branch),
		After: input.CommitSHA,
		Installation: pushInstallation{
			ID: input.InstallationID,
		},
		Repository: pushRepository{
			ID:       input.RepositoryID,
			FullName: input.Repository,
		},
		Commits:    []pushCommit{commit},
		HeadCommit: &commit,
		Sender: pushSender{
			Login:     input.AuthorUsername,
			AvatarURL: "",
		},
	}
}
