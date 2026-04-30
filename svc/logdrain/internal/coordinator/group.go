package coordinator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/unkeyed/unkey/pkg/db"
)

// Source is the ClickHouse table a group reads from. One drain can
// subscribe to multiple sources; the coordinator forks the drain into one
// group entry per source so the per-table query stays homogeneous.
type Source string

const (
	SourceRuntime Source = "runtime"
	SourceRequest Source = "request"
)

// GroupKey is the cursor namespace shared by every drain that reads the
// same (workspace, project, environment, source) slice of ClickHouse.
// Stored as a hex-encoded SHA-256 so the MySQL primary key has fixed
// width regardless of the underlying ID lengths.
type GroupKey string

func MakeGroupKey(workspace, project, environment string, src Source) GroupKey {
	h := sha256.New()
	// Field separator is U+001E (record separator), not present in any
	// of the ID alphabets, so MakeGroupKey("a", "b|c", ...) cannot collide
	// with MakeGroupKey("a|b", "c", ...).
	const sep = "\x1e"
	h.Write([]byte(workspace))
	h.Write([]byte(sep))
	h.Write([]byte(project))
	h.Write([]byte(sep))
	h.Write([]byte(environment))
	h.Write([]byte(sep))
	h.Write([]byte(src))
	return GroupKey(hex.EncodeToString(h.Sum(nil)))
}

// Drain is the subset of a log_drains row the coordinator carries through
// the pipeline. Credentials and provider-specific config have already
// been resolved into a constructed Sink by the time the coordinator
// reaches the worker stage.
type Drain struct {
	ID            string
	WorkspaceID   string
	ProjectID     string // empty means workspace-wide
	EnvironmentID string // empty means all environments
	Apps          []string
	Source        Source
	Filters       json.RawMessage
}

// Group is one (workspace, project, environment, source) bucket plus
// every drain attached to it. Cursor advances are per-group; sink fan-out
// is per-drain.
type Group struct {
	Key       GroupKey
	Workspace string
	Project   string
	Env       string
	Source    Source
	Drains    []Drain
}

// BuildGroups expands each log_drains row into one Drain entry per
// (source, environment) selection it carries, then buckets them into
// Groups by their GroupKey. Drains with empty ProjectID (workspace-wide)
// stay in their own group rather than fanning out across every project —
// the worker compares against the actual rows pulled from CH.
func BuildGroups(rows []db.ListEnabledLogDrainsRow) ([]Group, error) {
	type bucket struct {
		Group
	}
	buckets := map[GroupKey]*bucket{}

	for _, row := range rows {
		var sources []string
		if err := json.Unmarshal(row.Sources, &sources); err != nil {
			return nil, err
		}
		var envs []string
		if err := json.Unmarshal(row.Environments, &envs); err != nil {
			return nil, err
		}
		var apps []string
		if err := json.Unmarshal(row.Apps, &apps); err != nil {
			return nil, err
		}

		project := ""
		if row.ProjectID.Valid {
			project = row.ProjectID.String
		}

		for _, src := range sources {
			source := Source(src)
			for _, env := range envs {
				key := MakeGroupKey(row.WorkspaceID, project, env, source)
				b, ok := buckets[key]
				if !ok {
					b = &bucket{Group: Group{
						Key:       key,
						Workspace: row.WorkspaceID,
						Project:   project,
						Env:       env,
						Source:    source,
						Drains:    nil,
					}}
					buckets[key] = b
				}
				b.Drains = append(b.Drains, Drain{
					ID:            row.ID,
					WorkspaceID:   row.WorkspaceID,
					ProjectID:     project,
					EnvironmentID: env,
					Apps:          apps,
					Source:        source,
					Filters:       row.Filters,
				})
			}
		}
	}

	out := make([]Group, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, b.Group)
	}
	// Stable order so logs/metrics that include the group key are
	// reproducible across ticks for the same input.
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
