package openapi

import (
	"github.com/oasdiff/oasdiff/checker"
	"github.com/oasdiff/oasdiff/diff"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func convertSummaryToProto(summary *diff.Summary) *ctrlv1.DiffSummary {
	// Helper function to get counts safely
	getCounts := func(name string) *ctrlv1.DiffCounts {
		if details, exists := summary.Details[diff.DetailName(name)]; exists {
			return &ctrlv1.DiffCounts{
				Added:    int32(details.Added),    //nolint: gosec
				Deleted:  int32(details.Deleted),  //nolint: gosec
				Modified: int32(details.Modified), //nolint: gosec
			}
		}
		return &ctrlv1.DiffCounts{Added: 0, Deleted: 0, Modified: 0}
	}

	return &ctrlv1.DiffSummary{
		Diff: summary.Diff,
		Details: &ctrlv1.DiffDetails{
			Endpoints: getCounts("endpoints"),
			Paths:     getCounts("paths"),
			Schemas:   getCounts("schemas"),
		},
	}
}

func convertChangesToProto(changes checker.Changes) []*ctrlv1.ChangelogEntry {
	localizer := checker.NewLocalizer("en")
	result := make([]*ctrlv1.ChangelogEntry, len(changes))

	for i, change := range changes {
		level := int32(1) // INFO
		//nolint: exhaustive
		switch change.GetLevel() {
		case checker.WARN:
			level = 2
		case checker.ERR:
			level = 3
		}

		result[i] = &ctrlv1.ChangelogEntry{
			Id:          change.GetId(),
			Text:        change.GetUncolorizedText(localizer),
			Level:       level,
			Operation:   change.GetOperation(),
			Path:        change.GetPath(),
			OperationId: ptr.P(change.GetOperationId()),
		}
	}

	return result
}
