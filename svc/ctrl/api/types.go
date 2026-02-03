package api

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
