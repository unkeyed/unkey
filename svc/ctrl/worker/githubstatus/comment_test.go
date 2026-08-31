package githubstatus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpsertRowKeepsTargetsWithMatchingSlugs(t *testing.T) {
	firstKey := "workspace_1:project_1:app_1:environment_1"
	secondKey := "workspace_2:project_2:app_2:environment_2"
	body := buildFullComment(buildRow(firstKey, "project", "app", "production", "", "https://example.com/first", "Queued"))
	secondRow := buildRow(secondKey, "project", "app", "production", "", "https://example.com/second", "Queued")

	updated := upsertRow(secondKey, "app", "production", body, secondRow)

	require.Contains(t, updated, rowMarker(firstKey))
	require.Contains(t, updated, rowMarker(secondKey))
}

func TestUpsertRowMigratesLegacyTargetMarker(t *testing.T) {
	rowKey := "workspace_1:project_1:app_1:environment_1"
	legacyMarker := "<!-- row:app:production -->"
	body := buildFullComment("| " + legacyMarker + " **project / app** (production) | Queued | — | [Inspect](https://example.com) | now |")
	row := buildRow(rowKey, "project", "app", "production", "", "https://example.com", "Ready")

	updated := upsertRow(rowKey, "app", "production", body, row)

	require.NotContains(t, updated, legacyMarker)
	require.Contains(t, updated, rowMarker(rowKey))
	require.Contains(t, updated, "Ready")
}

func TestUpsertRowReplacesMatchingTarget(t *testing.T) {
	rowKey := "workspace_1:project_1:app_1:environment_1"
	body := buildFullComment(buildRow(rowKey, "project", "app", "production", "", "https://example.com", "Queued"))
	row := buildRow(rowKey, "project", "app", "production", "", "https://example.com", "Ready")

	updated := upsertRow(rowKey, "app", "production", body, row)

	require.Equal(t, 1, strings.Count(updated, rowMarker(rowKey)))
	require.NotContains(t, updated, "Queued")
	require.Contains(t, updated, "Ready")
}

func TestPullRequestCommentObjectKeyIsolatesPullRequests(t *testing.T) {
	key := pullRequestCommentObjectKey(123, 456, "unkeyed/unkey", 42)

	require.Equal(t, key, pullRequestCommentObjectKey(123, 456, "unkeyed/renamed", 42))
	require.NotEqual(t, key, pullRequestCommentObjectKey(124, 456, "unkeyed/unkey", 42))
	require.NotEqual(t, key, pullRequestCommentObjectKey(123, 789, "unkeyed/unkey", 42))
	require.NotEqual(t, key, pullRequestCommentObjectKey(123, 456, "unkeyed/unkey", 43))
	fallbackKey := pullRequestCommentObjectKey(123, 0, "Unkeyed/Unkey", 42)
	require.Equal(t, fallbackKey, pullRequestCommentObjectKey(123, 0, "unkeyed/unkey", 42))
	require.NotEqual(t, fallbackKey, pullRequestCommentObjectKey(123, 0, "unkeyed/other", 42))
}
