package api

type pushPayload struct {
	Ref          string           `json:"ref"`
	After        string           `json:"after"`
	Created      bool             `json:"created"`
	Deleted      bool             `json:"deleted"`
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
	Fork     bool   `json:"fork"`
}

type pushCommit struct {
	ID        string           `json:"id"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
	Author    pushCommitAuthor `json:"author"`
	Added     []string         `json:"added"`
	Removed   []string         `json:"removed"`
	Modified  []string         `json:"modified"`
}

type pushCommitAuthor struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type pushSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type pullRequestPayload struct {
	Action       string            `json:"action"`
	Number       int64             `json:"number"`
	PullRequest  pullRequestDetail `json:"pull_request"`
	Installation pushInstallation  `json:"installation"`
	Sender       pushSender        `json:"sender"`
}

type pullRequestDetail struct {
	Title string         `json:"title"`
	User  pushSender     `json:"user"`
	Head  pullRequestRef `json:"head"`
	Base  pullRequestRef `json:"base"`
}

type pullRequestRef struct {
	Ref  string         `json:"ref"`
	SHA  string         `json:"sha"`
	Repo pushRepository `json:"repo"`
}
