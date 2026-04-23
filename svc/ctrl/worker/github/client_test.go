package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCommitAuthor(t *testing.T) {
	tests := []struct {
		name              string
		commit            ghCommitResponse
		expectedHandle    string
		expectedAvatarURL string
	}{
		{
			name: "unverified commit with misconfigured email falls back to git metadata",
			commit: ghCommitResponse{
				SHA: "aaa1111111111111111111111111111111111111",
				Commit: ghCommitDetail{
					Message: "fix: update config",
					Author: ghCommitAuthor{
						Name:  "alice",
						Email: "alice@alices-MacBook-Pro.local",
						Date:  "2026-04-22T13:09:45Z",
					},
					Verification: ghCommitVerification{Verified: false},
				},
				Author: ghUser{
					Login:     "wronguser",
					AvatarURL: "https://avatars.githubusercontent.com/u/99999?v=4",
				},
			},
			expectedHandle:    "alice",
			expectedAvatarURL: "https://www.gravatar.com/avatar/b91be39b212b63b1fac5e8041f01ae89?d=identicon",
		},
		{
			name: "verified commit uses GitHub resolved author",
			commit: ghCommitResponse{
				SHA: "bbb2222222222222222222222222222222222222",
				Commit: ghCommitDetail{
					Message: "Update README.md",
					Author: ghCommitAuthor{
						Name:  "Alice",
						Email: "alice@example.com",
						Date:  "2026-04-22T13:50:03Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{
					Login:     "alice-gh",
					AvatarURL: "https://avatars.githubusercontent.com/u/12345?v=4",
				},
			},
			expectedHandle:    "alice-gh",
			expectedAvatarURL: "https://avatars.githubusercontent.com/u/12345?v=4",
		},
		{
			name: "verified commit from local with proper config uses GitHub resolved author",
			commit: ghCommitResponse{
				SHA: "ccc3333333333333333333333333333333333333",
				Commit: ghCommitDetail{
					Message: "feat: add endpoint",
					Author: ghCommitAuthor{
						Name:  "bob",
						Email: "bob@company.com",
						Date:  "2026-04-22T13:51:08Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{
					Login:     "bob-dev",
					AvatarURL: "https://avatars.githubusercontent.com/u/67890?v=4",
				},
			},
			expectedHandle:    "bob-dev",
			expectedAvatarURL: "https://avatars.githubusercontent.com/u/67890?v=4",
		},
		{
			name: "verified commit but empty GitHub login falls back to gravatar",
			commit: ghCommitResponse{
				SHA: "ddd4444444444444444444444444444444444444",
				Commit: ghCommitDetail{
					Message: "chore: cleanup",
					Author: ghCommitAuthor{
						Name:  "bob",
						Email: "bob@localhost",
						Date:  "2026-04-22T15:00:00Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{},
			},
			expectedHandle:    "bob",
			expectedAvatarURL: "https://www.gravatar.com/avatar/a1d4544a85858c1d7562c5659166af24?d=identicon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, avatarURL := resolveCommitAuthor(tt.commit)
			require.Equal(t, tt.expectedHandle, handle)
			require.Equal(t, tt.expectedAvatarURL, avatarURL)
		})
	}
}
