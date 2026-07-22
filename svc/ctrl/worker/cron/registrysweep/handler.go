// Package registrysweep implements the CronService.RunRegistrySweep
// handler: the reverse of deploymentcleanup. Instead of walking MySQL and
// deleting from Depot, it enumerates Depot (registry image tags and build
// projects) and deletes whatever no longer has a backing row in MySQL.
// This is the safety net for every path that drops rows without calling
// Depot — app/environment/project deletes, or bugs — where the forward
// cleanup can never see the orphan again.
//
// Deletion is strictly by reference, never by age: a tag is removed only
// when the deployment it encodes does not exist, so a deployment that
// stays ready for years keeps its image forever. The only age check is a
// minimum age on *orphans*, to avoid racing resources whose MySQL rows or
// Depot writes are still in flight.
package registrysweep

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/depotclient"
)

// minOrphanAge is how old a Depot resource must be before an apparent
// orphan is deleted. Covers the window where an image was pushed or a
// Depot project created but the corresponding MySQL write has not
// committed yet.
const minOrphanAge = 24 * time.Hour

const depotRegistryHost = "registry.depot.dev"

// maxPagesPerRun caps each Depot enumeration so the Restate journal stays
// bounded. The fixed virtual-object key persists the next page between runs.
const maxPagesPerRun = 200

// idChunkSize bounds the IN clause of each existence check.
const idChunkSize = 500

const (
	stateKeyImagePage        = "image_page"
	stateKeyProjectPageToken = "project_page_token"
)

// DepotAPI is the Depot surface the sweep needs. Satisfied by
// depotclient.Client and depotclient.Noop.
type DepotAPI interface {
	ListImages(ctx context.Context, repository string, page int32) ([]depotclient.Image, bool, error)
	DeleteTag(ctx context.Context, repository, tag string) error
	ListProjects(ctx context.Context, pageToken string) ([]depotclient.Project, string, error)
	DeleteProject(ctx context.Context, projectID string) error
}

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database

	// Depot is the Depot API. Must not be nil; use depotclient.NewNoop()
	// where Depot is not configured.
	Depot DepotAPI

	// Repository is the full registry repository the control plane pushes to.
	// It must be exclusive to this database; empty skips the image sweep.
	Repository string

	// DepotProjectPrefix is the environment's Depot project name prefix
	// (depot config project_prefix). Only projects named
	// "{prefix}-{unkeyProjectID}" are considered; empty skips the project
	// sweep. The caller must pass only a prefix exclusive to this database.
	DepotProjectPrefix string

	// Heartbeat is pinged after a successful sweep. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunRegistrySweep.
type Handler struct {
	db                 db.Database
	depot              DepotAPI
	imageRepository    string
	repository         string
	depotProjectPrefix string
	heartbeat          healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Depot, "Depot must not be nil; use depotclient.NewNoop()"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	repository := ""
	if cfg.Repository != "" {
		var err error
		repository, err = depotRepositoryPath(cfg.Repository)
		if err != nil {
			return nil, err
		}
	}
	return &Handler{
		db:                 cfg.DB,
		depot:              cfg.Depot,
		imageRepository:    cfg.Repository,
		repository:         repository,
		depotProjectPrefix: cfg.DepotProjectPrefix,
		heartbeat:          cfg.Heartbeat,
	}, nil
}

// Handle sweeps registry tags, then Depot projects. Enumeration completes
// before deletion within each run, and pagination state advances only after
// all deletes succeed. Every external call is journaled for deterministic
// replay.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunRegistrySweepRequest,
) (*hydrav1.RunRegistrySweepResponse, error) {
	if h.repository == "" && h.depotProjectPrefix == "" {
		return nil, fmt.Errorf("registry sweep is disabled: no exclusive repository or Depot project prefix configured")
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get now: %w", err)
	}
	orphanCutoff := now.Add(-minOrphanAge)

	resp := &hydrav1.RunRegistrySweepResponse{
		TagsDeleted:          0,
		TagsSkipped:          0,
		DepotProjectsDeleted: 0,
	}

	if err := h.sweepImages(ctx, orphanCutoff, resp); err != nil {
		return nil, err
	}
	if err := h.sweepDepotProjects(ctx, orphanCutoff, resp); err != nil {
		return nil, err
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"), restate.WithMaxRetryAttempts(5)); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return resp, nil
}

// candidateTag is one registry tag that parses as ours, paired with the
// deployment id it encodes.
type candidateTag struct {
	Tag          string `json:"tag"`
	DeploymentID string `json:"deploymentId"`
	Image        string `json:"image"`
}

// sweepImages reconciles one bounded page range and persists where the next run starts.
func (h *Handler) sweepImages(ctx restate.ObjectContext, orphanCutoff time.Time, resp *hydrav1.RunRegistrySweepResponse) error {
	if h.repository == "" {
		return nil
	}

	pageStart, err := restate.Get[int32](ctx, stateKeyImagePage)
	if err != nil {
		return fmt.Errorf("get image page: %w", err)
	}
	// Depot's page-number pagination is 1-based; zero is only Restate's
	// default value for state that has not been initialized yet.
	if pageStart < 1 {
		pageStart = 1
	}

	var candidates []candidateTag
	pageNext := pageStart
	reachedEnd := false
	for pageOffset := int32(0); pageOffset < maxPagesPerRun; pageOffset++ {
		page := pageStart + pageOffset

		repository := h.repository
		images, err := restate.Run(ctx, func(rc restate.RunContext) (listImagesResult, error) {
			imgs, more, listErr := h.depot.ListImages(rc, repository, page)
			return listImagesResult{Images: imgs, HasMore: more}, listErr
		}, restate.WithName(fmt.Sprintf("list images page-%d", page)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return fmt.Errorf("list images page %d: %w", page, err)
		}

		for _, img := range images.Images {
			for _, tag := range img.Tags {
				deploymentID, ok := deploymentIDFromTag(tag)
				if !ok {
					resp.TagsSkipped++
					continue
				}
				// A fresh image may belong to a deployment whose row
				// write is still in flight; leave it for the next run.
				if img.PushedAt.After(orphanCutoff) {
					continue
				}
				candidates = append(candidates, candidateTag{
					Tag:          tag,
					DeploymentID: deploymentID,
					Image:        h.imageRepository + ":" + tag,
				})
			}
		}

		if !images.HasMore {
			pageNext = 1
			reachedEnd = true
			break
		}
		pageNext = page + 1
	}
	if !reachedEnd {
		logger.Info("registry sweep reached the per-run image page cap",
			"repository", h.repository,
			"page_next", pageNext,
		)
	}

	orphans, err := h.filterOrphans(ctx, candidates, func(rc restate.RunContext, ids []string) ([]string, error) {
		return h.db.FilterExistingDeploymentIds(rc, ids)
	})
	if err != nil {
		return err
	}
	for _, orphan := range orphans {
		repository, tag := h.repository, orphan.Tag
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return h.depot.DeleteTag(rc, repository, tag)
		}, restate.WithName("delete orphaned tag "+tag), restate.WithMaxRetryAttempts(5)); err != nil {
			return fmt.Errorf("delete orphaned tag %q: %w", tag, err)
		}
		resp.TagsDeleted++
	}
	if len(orphans) > 0 {
		// Offset pagination shifts after deletion. Restarting guarantees shifted
		// manifests are revisited instead of skipping them until a full wrap.
		pageNext = 1
	}
	restate.Set(ctx, stateKeyImagePage, pageNext)

	logger.Info("registry sweep deleted orphaned image tags",
		"repository", h.repository,
		"tags_deleted", resp.TagsDeleted,
		"tags_skipped", resp.TagsSkipped,
	)
	return nil
}

// listImagesResult bundles ListImages' two return values into one
// JSON-serializable journal entry.
type listImagesResult struct {
	Images  []depotclient.Image `json:"images"`
	HasMore bool                `json:"hasMore"`
}

// candidateProject is one Depot project whose name parses as ours, paired
// with the Unkey project id it encodes.
type candidateProject struct {
	DepotProjectID string `json:"depotProjectId"`
	Name           string `json:"name"`
}

// sweepDepotProjects reconciles one bounded token range and persists the next token.
func (h *Handler) sweepDepotProjects(ctx restate.ObjectContext, orphanCutoff time.Time, resp *hydrav1.RunRegistrySweepResponse) error {
	if h.depotProjectPrefix == "" {
		return nil
	}

	pageToken, err := restate.Get[string](ctx, stateKeyProjectPageToken)
	if err != nil {
		return fmt.Errorf("get depot project page token: %w", err)
	}

	var candidates []candidateProject
	seenTokens := map[string]bool{pageToken: true}
	reachedEnd := false
	for page := 1; page <= maxPagesPerRun; page++ {
		token := pageToken
		result, err := restate.Run(ctx, func(rc restate.RunContext) (listProjectsResult, error) {
			projects, next, listErr := h.depot.ListProjects(rc, token)
			return listProjectsResult{Projects: projects, NextPageToken: next}, listErr
		}, restate.WithName(fmt.Sprintf("list depot projects page-%d", page)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return fmt.Errorf("list depot projects page %d: %w", page, err)
		}

		for _, p := range result.Projects {
			_, ok := projectIDFromDepotName(p.Name, h.depotProjectPrefix)
			if !ok {
				continue
			}
			if p.CreatedAt.After(orphanCutoff) {
				continue
			}
			candidates = append(candidates, candidateProject{
				DepotProjectID: p.ID,
				Name:           p.Name,
			})
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			reachedEnd = true
			break
		}
		if seenTokens[pageToken] {
			return fmt.Errorf("list depot projects returned repeated page token %q", pageToken)
		}
		seenTokens[pageToken] = true
	}
	if !reachedEnd {
		logger.Info("registry sweep reached the per-run project page cap",
			"page_token_next", pageToken,
		)
	}

	existing, err := h.existingDepotProjectIDs(ctx, candidates)
	if err != nil {
		return err
	}

	var orphans []candidateProject
	for _, c := range candidates {
		if !existing[c.DepotProjectID] {
			orphans = append(orphans, c)
		}
	}
	for _, orphan := range orphans {
		depotProjectID := orphan.DepotProjectID
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return h.depot.DeleteProject(rc, depotProjectID)
		}, restate.WithName("delete orphaned depot project "+depotProjectID), restate.WithMaxRetryAttempts(5)); err != nil {
			return fmt.Errorf("delete orphaned depot project %q (%s): %w", depotProjectID, orphan.Name, err)
		}
		resp.DepotProjectsDeleted++
	}
	if len(orphans) > 0 {
		pageToken = ""
	}
	restate.Set(ctx, stateKeyProjectPageToken, pageToken)

	logger.Info("registry sweep deleted orphaned depot projects",
		"depot_projects_deleted", resp.DepotProjectsDeleted,
	)
	return nil
}

// existingDepotProjectIDs returns candidate Depot IDs referenced by projects.
func (h *Handler) existingDepotProjectIDs(ctx restate.ObjectContext, candidates []candidateProject) (map[string]bool, error) {
	unique := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if !seen[candidate.DepotProjectID] {
			seen[candidate.DepotProjectID] = true
			unique = append(unique, candidate.DepotProjectID)
		}
	}

	existing := make(map[string]bool, len(unique))
	for start := 0; start < len(unique); start += idChunkSize {
		end := min(start+idChunkSize, len(unique))
		chunk := make([]sql.NullString, end-start)
		for i, id := range unique[start:end] {
			chunk[i] = sql.NullString{String: id, Valid: true}
		}
		found, err := restate.Run(ctx, func(rc restate.RunContext) ([]sql.NullString, error) {
			return h.db.FilterExistingDepotProjectIds(rc, chunk)
		}, restate.WithName(fmt.Sprintf("filter existing depot projects %d-%d", start, end)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return nil, fmt.Errorf("filter existing Depot project IDs: %w", err)
		}
		for _, id := range found {
			if id.Valid {
				existing[id.String] = true
			}
		}
	}
	return existing, nil
}

// listProjectsResult bundles ListProjects' two return values into one
// JSON-serializable journal entry.
type listProjectsResult struct {
	Projects      []depotclient.Project `json:"projects"`
	NextPageToken string                `json:"nextPageToken"`
}

// filterOrphans returns candidates with neither their encoded deployment nor
// any deployment referencing the exact image. The second check preserves tags
// reused by rebuilds.
func (h *Handler) filterOrphans(
	ctx restate.ObjectContext,
	candidates []candidateTag,
	filter func(restate.RunContext, []string) ([]string, error),
) ([]candidateTag, error) {
	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = c.DeploymentID
	}
	existing, err := h.existingIDs(ctx, "deployments", ids, filter)
	if err != nil {
		return nil, err
	}
	existingImages, err := h.existingImages(ctx, candidates)
	if err != nil {
		return nil, err
	}

	var orphans []candidateTag
	for _, c := range candidates {
		if !existing[c.DeploymentID] && !existingImages[c.Image] {
			orphans = append(orphans, c)
		}
	}
	return orphans, nil
}

// existingImages returns the candidate image references still used by a deployment.
func (h *Handler) existingImages(ctx restate.ObjectContext, candidates []candidateTag) (map[string]bool, error) {
	unique := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if !seen[candidate.Image] {
			seen[candidate.Image] = true
			unique = append(unique, candidate.Image)
		}
	}

	existing := make(map[string]bool, len(unique))
	for start := 0; start < len(unique); start += idChunkSize {
		end := min(start+idChunkSize, len(unique))
		chunk := make([]sql.NullString, end-start)
		for i, image := range unique[start:end] {
			chunk[i] = sql.NullString{String: image, Valid: true}
		}
		found, err := restate.Run(ctx, func(rc restate.RunContext) ([]sql.NullString, error) {
			return h.db.FilterExistingDeploymentImages(rc, chunk)
		}, restate.WithName(fmt.Sprintf("filter existing images %d-%d", start, end)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return nil, fmt.Errorf("filter existing deployment images: %w", err)
		}
		for _, image := range found {
			if image.Valid {
				existing[image.String] = true
			}
		}
	}
	return existing, nil
}

// existingIDs checks which of the given ids exist, in bounded chunks.
func (h *Handler) existingIDs(
	ctx restate.ObjectContext,
	table string,
	ids []string,
	filter func(restate.RunContext, []string) ([]string, error),
) (map[string]bool, error) {
	// Dedupe so multi-tag manifests don't inflate the IN clauses.
	unique := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	existing := make(map[string]bool, len(unique))
	for start := 0; start < len(unique); start += idChunkSize {
		end := min(start+idChunkSize, len(unique))
		chunk := unique[start:end]
		found, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
			return filter(rc, chunk)
		}, restate.WithName(fmt.Sprintf("filter existing %s %d-%d", table, start, end)), restate.WithMaxRetryAttempts(5))
		if err != nil {
			return nil, fmt.Errorf("filter existing %s: %w", table, err)
		}
		for _, id := range found {
			existing[id] = true
		}
	}
	return existing, nil
}

// deploymentIDFromTag extracts the deployment id from a
// "{projectID}-{deploymentID}" image tag as written by the deploy
// workflow. Both ids are prefix + "_" + alphanumerics, so the first "-"
// is always the separator. Returns false for any other tag shape so
// hand-pushed images are never treated as orphans.
func deploymentIDFromTag(tag string) (string, bool) {
	projectPart, deploymentPart, found := strings.Cut(tag, "-")
	if !found {
		return "", false
	}
	if !validUID(projectPart, "proj_") {
		return "", false
	}
	if !validUID(deploymentPart, "d_") {
		return "", false
	}
	return deploymentPart, true
}

// projectIDFromDepotName extracts the Unkey project id from a Depot
// project named "{prefix}-{unkeyProjectID}". Returns false for any other
// name so projects of other environments (different prefix) or created by
// hand are never treated as orphans.
func projectIDFromDepotName(name, prefix string) (string, bool) {
	projectID, found := strings.CutPrefix(name, prefix+"-")
	if !found {
		return "", false
	}
	if !validUID(projectID, "proj_") {
		return "", false
	}
	return projectID, true
}

// validUID checks the generated UID grammar without accepting punctuation or
// additional separators that could belong to hand-created resources.
func validUID(value, prefix string) bool {
	suffix, found := strings.CutPrefix(value, prefix)
	if !found || suffix == "" {
		return false
	}
	for _, c := range []byte(suffix) {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// depotRepositoryPath validates that repository belongs to Depot and returns
// the path expected by the Depot registry API.
func depotRepositoryPath(repository string) (string, error) {
	host, path, found := strings.Cut(repository, "/")
	if !found || path == "" {
		return "", fmt.Errorf("registry repository %q must include a host and path", repository)
	}
	if host != depotRegistryHost {
		return "", fmt.Errorf("registry repository %q must use %s", repository, depotRegistryHost)
	}
	return path, nil
}
