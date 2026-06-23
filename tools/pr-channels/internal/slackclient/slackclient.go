// Package slackclient wraps the Slack Web API calls the bot needs: creating and
// archiving public channels, inviting reviewers, and posting messages. slack-go
// handles 429 Retry-After backoff for us.
package slackclient

import (
	"context"
	"errors"
	"strings"

	"github.com/slack-go/slack"
)

// Client is a thin wrapper around *slack.Client.
type Client struct {
	api *slack.Client
}

// New constructs a client from a bot token.
func New(token string) *Client {
	return &Client{api: slack.New(token)}
}

// EnsureChannel returns the id of a public channel with the given name, creating
// it if necessary. If the channel already exists (name_taken) it is looked up
// and un-archived so a reopened PR reuses its channel.
func (c *Client) EnsureChannel(ctx context.Context, name string) (string, error) {
	ch, err := c.api.CreateConversationContext(ctx, slack.CreateConversationParams{
		ChannelName: name,
		IsPrivate:   false,
	})
	if err == nil {
		return ch.ID, nil
	}
	if err.Error() != "name_taken" {
		return "", err
	}
	// Already exists: find it and make sure it is not archived.
	id, findErr := c.findChannelID(ctx, name)
	if findErr != nil {
		return "", findErr
	}

	if unErr := c.api.UnArchiveConversationContext(ctx, id); unErr != nil &&
		!strings.Contains(unErr.Error(), "not_archived") {
		// Non-fatal: the channel exists and is usable.
		_ = unErr
	}

	return id, nil
}

// findChannelID pages conversations.list looking for an exact name match. This
// is the only rate-limit-sensitive call and runs only on the rare name_taken
// path; normally the channel id comes from the store.
func (c *Client) findChannelID(ctx context.Context, name string) (string, error) {
	params := &slack.GetConversationsParameters{
		Types:           []string{"public_channel"},
		ExcludeArchived: false,
		Limit:           1000,
	}

	for {
		channels, cursor, err := c.api.GetConversationsContext(ctx, params)
		if err != nil {
			return "", err
		}
		for _, ch := range channels {
			if ch.Name == name {
				return ch.ID, nil
			}
		}

		if cursor == "" {
			return "", errors.New("channel " + name + " reported taken but not found")
		}
		params.Cursor = cursor
	}
}

// Invite adds users to a channel, ignoring the harmless "already_in_channel"
// error. Unknown/empty ids are skipped by the caller.
func (c *Client) Invite(ctx context.Context, channelID string, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := c.api.InviteUsersToConversationContext(ctx, channelID, userIDs...)
	if err != nil && !ignorableInvite(err.Error()) {
		return err
	}
	return nil
}

func ignorableInvite(msg string) bool {
	switch msg {
	case "already_in_channel", "cant_invite_self", "cant_invite":
		return true
	}
	return false
}

// Post sends a markdown message to a channel.
func (c *Client) Post(ctx context.Context, channelID, markdown string) error {
	_, _, err := c.api.PostMessageContext(ctx, channelID,
		slack.MsgOptionText(markdown, false),
		slack.MsgOptionDisableLinkUnfurl(),
	)
	return err
}

// Archive archives a channel, ignoring the case where it is already archived.
func (c *Client) Archive(ctx context.Context, channelID string) error {
	err := c.api.ArchiveConversationContext(ctx, channelID)
	if err != nil && err.Error() != "already_archived" {
		return err
	}
	return nil
}

// UserByEmail resolves a Slack user id from an email address, used to seed the
// user map. Returns "" if no user matches.
func (c *Client) UserByEmail(ctx context.Context, email string) (string, error) {
	u, err := c.api.GetUserByEmailContext(ctx, email)
	if err != nil {
		if err.Error() == "users_not_found" {
			return "", nil
		}
		return "", err
	}
	return u.ID, nil
}
