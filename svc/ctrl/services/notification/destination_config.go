package notification

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/ssrf"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/notificationrouting"
	"google.golang.org/protobuf/proto"
)

var (
	slackPath   = regexp.MustCompile(`^/services/[A-Za-z0-9]+/[A-Za-z0-9]+/[A-Za-z0-9]+$`)
	discordPath = regexp.MustCompile(`^/api(?:/v[0-9]+)?/webhooks/[0-9]+/[A-Za-z0-9._-]+$`)
	headerName  = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")
)

func validateURL(channel, raw string) (*url.URL, error) {
	invalid := rpcError(connect.CodeInvalidArgument, "invalid destination URL")
	if len(raw) > 8192 || ssrf.ValidateEndpoint(raw) != nil || strings.ContainsAny(raw, "\\#\r\n\t ") {
		return nil, invalid
	}
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Opaque != "" || endpoint.Hostname() == "" || strings.Contains(endpoint.Hostname(), "%") {
		return nil, invalid
	}
	if ip := net.ParseIP(endpoint.Hostname()); ip != nil && ssrf.IsForbiddenIP(ip) {
		return nil, invalid
	}
	if endpoint.Port() != "" {
		port, err := strconv.Atoi(endpoint.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, invalid
		}
	}
	switch channel {
	case "slack":
		if endpoint.Host != "hooks.slack.com" || !slackPath.MatchString(endpoint.EscapedPath()) || endpoint.RawQuery != "" || endpoint.ForceQuery {
			return nil, invalid
		}
	case "discord":
		if (endpoint.Host != "discord.com" && endpoint.Host != "discordapp.com") || !discordPath.MatchString(endpoint.EscapedPath()) || endpoint.RawQuery != "" || endpoint.ForceQuery {
			return nil, invalid
		}
	case "webhook":
	default:
		return nil, rpcError(connect.CodeInvalidArgument, "unsupported destination channel")
	}
	return endpoint, nil
}

func validateHeaders(headers []*ctrlv1.NotificationHeader) error {
	if len(headers) > 16 {
		return rpcError(connect.CodeInvalidArgument, "at most 16 headers are allowed")
	}
	seen := make(map[string]bool, len(headers))
	size := 0
	for _, header := range headers {
		name, value := header.GetName(), header.GetValue()
		canonical := http.CanonicalHeaderKey(name)
		if len(name) > 128 || !headerName.MatchString(name) || len(value) > 2048 || seen[canonical] || strings.HasPrefix(canonical, "Unkey-") {
			return rpcError(connect.CodeInvalidArgument, "invalid or duplicate destination header")
		}
		switch canonical {
		case "Connection", "Content-Length", "Content-Type", "Host", "Proxy-Authorization", "Proxy-Connection", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
			return rpcError(connect.CodeInvalidArgument, "reserved destination header")
		}
		for _, character := range value {
			if (character < ' ' && character != '\t') || character == 0x7f {
				return rpcError(connect.CodeInvalidArgument, "invalid destination header value")
			}
		}
		seen[canonical] = true
		size += len(name) + len(value)
	}
	if size > 8192 {
		return rpcError(connect.CodeInvalidArgument, "destination headers are too large")
	}
	return nil
}

func (s *Service) configureDestination(ctx context.Context, workspaceID, channel string, config *notificationv1.DestinationConfig, endpoint *string, headers *ctrlv1.NotificationHeaders, secret *string) error {
	if channel != "webhook" && (headers != nil || secret != nil) {
		return rpcError(connect.CodeInvalidArgument, "headers and signing_secret require the webhook channel")
	}
	if endpoint != nil {
		if _, err := validateURL(channel, *endpoint); err != nil {
			return err
		}
	}
	if headers != nil {
		if err := validateHeaders(headers.GetItems()); err != nil {
			return err
		}
	}
	if secret != nil && (!utf8.ValidString(*secret) || utf8.RuneCountInString(*secret) < 32 || len(*secret) > 512) {
		return rpcError(connect.CodeInvalidArgument, "signing_secret must contain at least 32 characters and at most 512 bytes")
	}
	if endpoint != nil {
		encrypted, err := s.encrypt(ctx, workspaceID, *endpoint)
		if err != nil {
			return err
		}
		switch channel {
		case "slack":
			config.Destination = &notificationv1.DestinationConfig_Slack{Slack: &notificationv1.SlackConfig{EncryptedWebhookUrl: encrypted}}
		case "discord":
			config.Destination = &notificationv1.DestinationConfig_Discord{Discord: &notificationv1.DiscordConfig{EncryptedWebhookUrl: encrypted}}
		case "webhook":
			if config.GetWebhook() == nil {
				config.Destination = &notificationv1.DestinationConfig_Webhook{Webhook: &notificationv1.WebhookConfig{}}
			}
			config.GetWebhook().EncryptedUrl = encrypted
		}
	}
	if headers != nil {
		config.GetWebhook().Headers = nil
		for _, header := range headers.GetItems() {
			encrypted, err := s.encrypt(ctx, workspaceID, header.GetValue())
			if err != nil {
				return err
			}
			config.GetWebhook().Headers = append(config.GetWebhook().Headers, &notificationv1.WebhookHeader{Name: http.CanonicalHeaderKey(header.GetName()), EncryptedValue: encrypted})
		}
	}
	if secret != nil {
		encrypted, err := s.encrypt(ctx, workspaceID, *secret)
		if err != nil {
			return err
		}
		config.GetWebhook().EncryptedSigningSecret = encrypted
	}
	return nil
}

func decodeDestination(row db.NotificationDestination) (*notificationv1.DestinationConfig, error) {
	config := &notificationv1.DestinationConfig{}
	if !notificationrouting.ValidDestination(notification.Channel(row.Channel), row.Config) || proto.Unmarshal(row.Config, config) != nil {
		return nil, rpcError(connect.CodeInternal, "invalid stored destination")
	}
	return config, nil
}

func (s *Service) destinationDTO(ctx context.Context, row db.NotificationDestination) (*ctrlv1.NotificationDestination, error) {
	if s.vault == nil {
		return nil, rpcError(connect.CodeUnavailable, "notification encryption is not configured")
	}
	config, err := decodeDestination(row)
	if err != nil {
		return nil, err
	}
	var encryptedURL string
	switch row.Channel {
	case db.NotificationDestinationsChannelSlack:
		encryptedURL = config.GetSlack().GetEncryptedWebhookUrl()
	case db.NotificationDestinationsChannelDiscord:
		encryptedURL = config.GetDiscord().GetEncryptedWebhookUrl()
	case db.NotificationDestinationsChannelWebhook:
		encryptedURL = config.GetWebhook().GetEncryptedUrl()
	}
	decrypted, err := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{Keyring: row.WorkspaceID, Encrypted: encryptedURL})
	if err != nil {
		return nil, rpcError(connect.CodeInternal, "cannot read destination configuration")
	}
	endpoint, err := validateURL(string(row.Channel), decrypted.GetPlaintext())
	if err != nil {
		return nil, rpcError(connect.CodeInternal, "invalid stored destination URL")
	}
	names := make([]string, 0, len(config.GetWebhook().GetHeaders()))
	for _, header := range config.GetWebhook().GetHeaders() {
		names = append(names, header.GetName())
	}
	sort.Strings(names)
	return &ctrlv1.NotificationDestination{
		Id:          row.ID,
		Name:        row.Name,
		Channel:     string(row.Channel),
		Enabled:     row.Enabled,
		UrlHint:     (&url.URL{Scheme: endpoint.Scheme, Host: endpoint.Host}).String(),
		HeaderNames: names,
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (s *Service) encrypt(ctx context.Context, workspaceID, plaintext string) (string, error) {
	if s.vault == nil {
		return "", rpcError(connect.CodeUnavailable, "notification encryption is not configured")
	}
	response, err := s.vault.Encrypt(ctx, &vaultv1.EncryptRequest{Keyring: workspaceID, Data: plaintext})
	if err != nil {
		return "", rpcError(connect.CodeInternal, "cannot encrypt destination configuration")
	}
	return response.GetEncrypted(), nil
}
