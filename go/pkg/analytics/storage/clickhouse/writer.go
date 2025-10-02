package clickhouse

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// Writer implements the analytics.Writer interface for ClickHouse storage.
// It wraps the existing ClickHouse client to provide analytics functionality
// while maintaining compatibility with the current batching system.
type Writer struct {
	client clickhouse.ClickHouse
}

// newWriter creates a new ClickHouse writer using the existing client.
func newWriter(client clickhouse.ClickHouse) analytics.Writer {
	return &Writer{
		client: client,
	}
}

// KeyVerification writes a key verification event to ClickHouse.
// It converts the v2 schema to v1 for compatibility with the existing system.
func (w *Writer) KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error {
	// Convert V2 to V1 format for compatibility with existing ClickHouse client
	v1Data := schema.KeyVerificationRequestV1{
		RequestID:   data.RequestID,
		Time:        data.Time,
		WorkspaceID: data.WorkspaceID,
		KeySpaceID:  data.KeySpaceID,
		IdentityID:  data.IdentityID,
		KeyID:       data.KeyID,
		Region:      data.Region,
		Outcome:     data.Outcome,
		Tags:        data.Tags,
	}

	w.client.BufferKeyVerification(v1Data)
	return nil
}

// Ratelimit writes a ratelimit event to ClickHouse.
// It converts the v2 schema to v1 for compatibility with the existing system.
func (w *Writer) Ratelimit(ctx context.Context, data schema.RatelimitV2) error {
	// Convert V2 to V1 format for compatibility with existing ClickHouse client
	v1Data := schema.RatelimitRequestV1{
		RequestID:   data.RequestID,
		Time:        data.Time,
		WorkspaceID: data.WorkspaceID,
		NamespaceID: data.NamespaceID,
		Identifier:  data.Identifier,
		Passed:      data.Passed,
	}

	w.client.BufferRatelimit(v1Data)
	return nil
}

// ApiRequest writes an API request event to ClickHouse.
// It uses the v2 schema directly as it's already supported by the existing client.
func (w *Writer) ApiRequest(ctx context.Context, data schema.ApiRequestV2) error {
	w.client.BufferApiRequest(data)
	return nil
}

// Close gracefully shuts down the ClickHouse writer.
// Note: The underlying ClickHouse client manages its own lifecycle,
// so this is primarily for interface compliance.
func (w *Writer) Close(ctx context.Context) error {
	// The ClickHouse client handles its own shutdown lifecycle
	// This method is here for Writer interface compliance
	return nil
}
