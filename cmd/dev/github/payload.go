package github

import (
	"fmt"
)

type pushPayloadInput struct {
	Branch         string
	CommitSHA      string
	InstallationID int64
	RepositoryID   int64
	Repository     string
}

type pushPayload struct {
	Ref          string           `json:"ref"`
	After        string           `json:"after"`
	Installation pushInstallation `json:"installation"`
	Repository   pushRepository   `json:"repository"`
}

type pushInstallation struct {
	ID int64 `json:"id"`
}

type pushRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

func buildPushPayload(input pushPayloadInput) pushPayload {
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
	}
}
