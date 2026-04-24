package auditlogarchive

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// safetyBufferMillis is subtracted from now() to compute the cutoff. Lets
// in-flight inserts (live drainer or backfill VO) finish before their rows
// are eligible for archival, so we don't race with a row being inserted as
// we DELETE it. 5 minutes is generous; the drainer's longest tail is well
// under that.
const safetyBufferMillis int64 = 5 * 60 * 1000

// archiveResult is the journaled outcome of a single archival pass.
type archiveResult struct {
	RowsArchived int64 `json:"rows_archived"`
	CutoffMillis int64 `json:"cutoff_millis"`
}

// RunArchive executes one archival pass:
//
//  1. Compute cutoff = now() - safetyBufferMillis.
//  2. Count rows past retention (cheap; uses the partition prune on
//     inserted_at + bloom on event). Skip everything if zero.
//  3. INSERT INTO FUNCTION s3(...) SELECT * FROM audit_logs_raw_v1
//     WHERE expires_at < cutoff.
//  4. ALTER TABLE audit_logs_raw_v1 DELETE WHERE expires_at < cutoff
//     with mutations_sync = 2 so the cron tick blocks until the deletion
//     has materialized.
//  5. Heartbeat.
//
// Each step is its own restate.Run so a crash mid-pass replays only the
// last incomplete step. Step 3 is the only step where retry can produce a
// duplicate object in S3; that object will be a strict superset of the
// prior one (same cutoff, same WHERE) so consumers must dedup by event_id
// when reading back.
func (s *Service) RunArchive(
	ctx restate.ObjectContext,
	_ *hydrav1.RunArchiveRequest,
) (*hydrav1.RunArchiveResponse, error) {
	if s.disabled {
		logger.Warn("audit log archive disabled by config kill switch; skipping")
		return &hydrav1.RunArchiveResponse{RowsArchived: 0, CutoffMillis: 0}, nil
	}

	cutoff, err := restate.Run(ctx, func(restate.RunContext) (int64, error) {
		return time.Now().UnixMilli() - safetyBufferMillis, nil
	}, restate.WithName("compute cutoff"))
	if err != nil {
		return nil, fmt.Errorf("compute cutoff: %w", err)
	}

	logger.Info("audit log archive starting pass", "cutoff_millis", cutoff)

	count, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return s.countExpired(rc, cutoff)
	}, restate.WithName("count expired"))
	if err != nil {
		return nil, fmt.Errorf("count expired: %w", err)
	}
	if count == 0 {
		logger.Info("audit log archive nothing to do", "cutoff_millis", cutoff)
		if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, s.heartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
		return &hydrav1.RunArchiveResponse{RowsArchived: 0, CutoffMillis: cutoff}, nil
	}

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.exportToS3(rc, cutoff)
	}, restate.WithName("export to s3")); err != nil {
		return nil, fmt.Errorf("export to s3: %w", err)
	}

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.deleteExpired(rc, cutoff)
	}, restate.WithName("delete expired")); err != nil {
		return nil, fmt.Errorf("delete expired: %w", err)
	}

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	logger.Info("audit log archive pass complete",
		"rows_archived", count,
		"cutoff_millis", cutoff,
	)
	return &hydrav1.RunArchiveResponse{
		RowsArchived: count,
		CutoffMillis: cutoff,
	}, nil
}

// countExpired returns how many rows are past retention as of cutoff. We
// count before exporting so an empty pass exits without producing an
// empty Parquet object in S3.
func (s *Service) countExpired(ctx context.Context, cutoffMillis int64) (int64, error) {
	rows, err := s.clickhouse.QueryToMaps(ctx,
		"SELECT count() AS c FROM default.audit_logs_raw_v1 WHERE expires_at < ?",
		cutoffMillis,
	)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	switch v := rows[0]["c"].(type) {
	case uint64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected count type %T", rows[0]["c"])
	}
}

// exportToS3 writes a Parquet file containing every expired row to the
// configured bucket. The filename includes the cutoff so retries clobber
// the same file; CH multipart upload only commits on success, so a failed
// upload leaves nothing partial behind.
func (s *Service) exportToS3(ctx context.Context, cutoffMillis int64) error {
	objectURL := s.objectURL(cutoffMillis)

	// We pass the cutoff as a parameter for the SELECT, but the s3() URL +
	// credentials must be string-baked because CH doesn't parameterize
	// table function arguments. Inputs are validated at New() so injection
	// is not a risk.
	stmt := fmt.Sprintf(
		`INSERT INTO FUNCTION s3('%s', '%s', '%s', 'Parquet')
         SELECT *
         FROM default.audit_logs_raw_v1
         WHERE expires_at < ?`,
		objectURL,
		s.cfg.AccessKey,
		// Secret is escaped against ' which is the only character that
		// would break out of the SQL string literal. Validation in
		// validateEndpoint forbids ' in the endpoint and prefix.
		strings.ReplaceAll(s.cfg.SecretKey, "'", "''"),
	)
	return s.clickhouse.Exec(ctx, stmt, cutoffMillis)
}

// deleteExpired runs an ALTER TABLE DELETE that physically removes the
// archived rows. mutations_sync = 2 makes Exec block until the mutation is
// done on all replicas, so the heartbeat fires only after deletion has
// landed. ALTER TABLE DELETE in CH is heavyweight; we lean on the cron
// being infrequent (once per day, say) to avoid stacking mutations.
func (s *Service) deleteExpired(ctx context.Context, cutoffMillis int64) error {
	return s.clickhouse.Exec(ctx,
		"ALTER TABLE default.audit_logs_raw_v1 DELETE WHERE expires_at < ? SETTINGS mutations_sync = 2",
		cutoffMillis,
	)
}

// objectURL builds the S3 object URL the cron writes to. Includes the
// cutoff so re-runs targeting the same window land on the same object,
// and cutoff_iso so a human can find the right file by date.
func (s *Service) objectURL(cutoffMillis int64) string {
	cutoffIso := time.UnixMilli(cutoffMillis).UTC().Format("2006-01-02T15-04-05")
	suffix := fmt.Sprintf("cutoff=%s/run_%d.parquet", cutoffIso, cutoffMillis)
	if s.cfg.Prefix == "" {
		return s.cfg.Endpoint + "/" + suffix
	}
	return s.cfg.Endpoint + "/" + s.cfg.Prefix + "/" + suffix
}

// validIdentifierComponent matches segments safe to splice into both URL
// paths and SQL string literals. Rejects ', ", \, whitespace, and any
// character outside [A-Za-z0-9._/-]. Bucket / prefix come from config so
// validation at boot is enough.
var validIdentifierComponent = regexp.MustCompile(`^[A-Za-z0-9._\-/:]+$`)

// validateEndpoint requires a parseable HTTPS URL with no path-style
// trailing slash and no characters that would break a SQL literal.
func validateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("not a URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if strings.HasSuffix(endpoint, "/") {
		return fmt.Errorf("must not end with slash")
	}
	if !validIdentifierComponent.MatchString(endpoint) {
		return fmt.Errorf("contains characters that could break SQL or URL parsing")
	}
	return nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	if strings.HasPrefix(prefix, "/") || strings.HasSuffix(prefix, "/") {
		return fmt.Errorf("must not start or end with slash")
	}
	if !validIdentifierComponent.MatchString(prefix) {
		return fmt.Errorf("contains characters that could break SQL or URL parsing")
	}
	return nil
}
