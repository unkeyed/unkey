// Package deploymentgc implements deployment history retention and Depot to
// MySQL reconciliation for CronService.RunDeploymentGarbageCollection.
package deploymentgc

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploymentretention"
)

const (
	pageSize                = int32(500)
	maxDeploymentDeletes    = int32(1_000)
	maxDepotResourceDeletes = int32(1_000)
	depotGracePeriod        = 24 * time.Hour
	runMaxAttempts          = 5
)

// Config holds dependencies and ownership boundaries for garbage collection.
type Config struct {
	DB db.Database

	// Depot is nil when the worker does not use Depot. Deployment retention
	// still runs, but cross-system reconciliation is disabled.
	Depot              Depot
	RegistryRepository string
	RegistryProjectID  string
	ProjectPrefix      string
}

// Handler owns deployment retention and Depot reconciliation.
type Handler struct {
	db                 db.Database
	depot              Depot
	registryRepository string
	registryProjectID  string
	projectPrefix      string
}

type projectReference struct {
	ProjectID      string
	DepotProjectID string
}

type depotImagePage struct {
	Images        []DepotImage
	NextPageToken string
}

type depotProjectPage struct {
	Projects      []DepotProject
	NextPageToken string
}

type depotProjectLookup struct {
	Project DepotProject
	Exists  bool
}

// New constructs a deployment garbage collection handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.NotNil(cfg.DB, "DB must not be nil"); err != nil {
		return nil, err
	}
	if cfg.Depot != nil {
		if err := assert.All(
			assert.NotEmpty(cfg.RegistryRepository, "RegistryRepository must not be empty when Depot reconciliation is enabled"),
			assert.NotEmpty(cfg.RegistryProjectID, "RegistryProjectID must not be empty when Depot reconciliation is enabled"),
			assert.NotEmpty(cfg.ProjectPrefix, "ProjectPrefix must not be empty when Depot reconciliation is enabled"),
		); err != nil {
			return nil, err
		}
	}
	return &Handler{
		db:                 cfg.DB,
		depot:              cfg.Depot,
		registryRepository: strings.TrimSuffix(cfg.RegistryRepository, "/"),
		registryProjectID:  cfg.RegistryProjectID,
		projectPrefix:      cfg.ProjectPrefix,
	}, nil
}

// Handle dispatches serialized deployment deletion, then reconciles both
// directions between Depot and MySQL. Depot resources younger than 24 hours
// are never deleted.
func (h *Handler) Handle(ctx restate.ObjectContext, _ *hydrav1.RunDeploymentGarbageCollectionRequest) (*hydrav1.RunDeploymentGarbageCollectionResponse, error) {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get garbage collection time: %w", err)
	}

	response := &hydrav1.RunDeploymentGarbageCollectionResponse{}
	response.DeploymentsDispatched, err = h.dispatchExpiredDeployments(ctx, now)
	if err != nil {
		return nil, err
	}
	if h.depot == nil {
		logger.Warn("Depot reconciliation is disabled for deployment garbage collection")
		return response, nil
	}

	databaseImages, databaseProjects, err := h.loadDatabaseInventory(ctx)
	if err != nil {
		return nil, err
	}
	depotImages, depotProjects, err := h.loadDepotInventory(ctx)
	if err != nil {
		return nil, err
	}

	response.MissingDepotImages = h.reportMissingImages(databaseImages, depotImages)
	response.StaleProjectReferencesCleared, err = h.clearStaleProjectReferences(ctx, now, databaseProjects, depotProjects)
	if err != nil {
		return nil, err
	}
	imagesDeleted, err := h.deleteOrphanImages(ctx, now, databaseImages, depotImages, maxDepotResourceDeletes)
	if err != nil {
		return nil, err
	}
	projectsDeleted, err := h.deleteOrphanProjects(ctx, now, databaseProjects, depotProjects, maxDepotResourceDeletes-imagesDeleted)
	if err != nil {
		return nil, err
	}
	response.DepotResourcesDeleted = imagesDeleted + projectsDeleted

	logger.Info("deployment garbage collection completed",
		"deployments_dispatched", response.GetDeploymentsDispatched(),
		"missing_depot_images", response.GetMissingDepotImages(),
		"stale_project_references_cleared", response.GetStaleProjectReferencesCleared(),
		"depot_resources_deleted", response.GetDepotResourcesDeleted(),
	)
	return response, nil
}

func (h *Handler) dispatchExpiredDeployments(ctx restate.ObjectContext, now time.Time) (int32, error) {
	productionCutoff, previewCutoff := deploymentretention.Cutoffs(now)
	var cursor uint64
	var dispatched int32
	for dispatched < maxDeploymentDeletes {
		limit := min(pageSize, maxDeploymentDeletes-dispatched)
		deployments, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListDeploymentGCCandidatesRow, error) {
			return h.db.ListDeploymentGCCandidates(runCtx, db.ListDeploymentGCCandidatesParams{
				PaginationCursor: cursor,
				PreviewCutoff:    previewCutoff,
				ProductionCutoff: productionCutoff,
				KeepSuccessful:   deploymentretention.Successful,
				Limit:            limit,
			})
		}, restate.WithName("list expired deployments"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return dispatched, fmt.Errorf("list expired deployments: %w", err)
		}
		if len(deployments) == 0 {
			break
		}
		cursor = deployments[len(deployments)-1].Pk
		for _, deployment := range deployments {
			hydrav1.NewDeployServiceClient(ctx, deployment.ID).
				GarbageCollect().
				Send(&hydrav1.GarbageCollectDeploymentRequest{DeploymentId: deployment.ID})
			dispatched++
		}
	}
	return dispatched, nil
}

func (h *Handler) loadDatabaseInventory(ctx restate.ObjectContext) (map[string]struct{}, []projectReference, error) {
	images := make(map[string]struct{})
	var cursor uint64
	for {
		rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListDeploymentImagesForGCRow, error) {
			return h.db.ListDeploymentImagesForGC(runCtx, db.ListDeploymentImagesForGCParams{
				PaginationCursor: cursor,
				Limit:            pageSize,
			})
		}, restate.WithName("list deployment image references"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, nil, fmt.Errorf("list deployment image references: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		cursor = rows[len(rows)-1].Pk
		for _, row := range rows {
			if tag, ok := strings.CutPrefix(row.Image.String, h.registryRepository+":"); ok && tag != "" {
				images[tag] = struct{}{}
			}
		}
	}

	var projects []projectReference
	cursor = 0
	for {
		rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListProjectDepotReferencesRow, error) {
			return h.db.ListProjectDepotReferences(runCtx, db.ListProjectDepotReferencesParams{
				PaginationCursor: cursor,
				Limit:            pageSize,
			})
		}, restate.WithName("list Depot project references"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, nil, fmt.Errorf("list Depot project references: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		cursor = rows[len(rows)-1].Pk
		for _, row := range rows {
			projects = append(projects, projectReference{ProjectID: row.ID, DepotProjectID: row.DepotProjectID.String})
		}
	}
	return images, projects, nil
}

func (h *Handler) loadDepotInventory(ctx restate.ObjectContext) (map[string]DepotImage, map[string]DepotProject, error) {
	images, err := h.loadDepotImages(ctx)
	if err != nil {
		return nil, nil, err
	}

	projects := make(map[string]DepotProject)
	pageToken := ""
	for {
		page, err := restate.Run(ctx, func(runCtx restate.RunContext) (depotProjectPage, error) {
			items, nextPage, listErr := h.depot.ListProjects(runCtx, pageToken)
			return depotProjectPage{Projects: items, NextPageToken: nextPage}, listErr
		}, restate.WithName("list Depot build projects"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, nil, fmt.Errorf("list Depot build projects: %w", err)
		}
		for _, project := range page.Projects {
			projects[project.ID] = project
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return images, projects, nil
}

func (h *Handler) loadDepotImages(ctx restate.ObjectContext) (map[string]DepotImage, error) {
	images := make(map[string]DepotImage)
	pageToken := ""
	for {
		page, err := restate.Run(ctx, func(runCtx restate.RunContext) (depotImagePage, error) {
			items, nextPage, listErr := h.depot.ListImages(runCtx, h.registryProjectID, pageToken)
			return depotImagePage{Images: items, NextPageToken: nextPage}, listErr
		}, restate.WithName("list Depot registry images"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, fmt.Errorf("list Depot registry images: %w", err)
		}
		for _, image := range page.Images {
			images[image.Tag] = image
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return images, nil
}

func (h *Handler) reportMissingImages(databaseImages map[string]struct{}, depotImages map[string]DepotImage) int32 {
	var missing int32
	for tag := range databaseImages {
		if _, ok := depotImages[tag]; ok {
			continue
		}
		missing++
		logger.Error("deployment references a missing Depot image",
			"image", h.registryRepository+":"+tag,
		)
	}
	return missing
}

func (h *Handler) clearStaleProjectReferences(ctx restate.ObjectContext, now time.Time, databaseProjects []projectReference, depotProjects map[string]DepotProject) (int32, error) {
	var cleared int32
	for _, reference := range databaseProjects {
		if _, ok := depotProjects[reference.DepotProjectID]; ok {
			continue
		}

		lookup, err := restate.Run(ctx, func(runCtx restate.RunContext) (depotProjectLookup, error) {
			project, exists, getErr := h.depot.GetProject(runCtx, reference.DepotProjectID)
			return depotProjectLookup{Project: project, Exists: exists}, getErr
		}, restate.WithName("recheck missing Depot project"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return cleared, fmt.Errorf("recheck missing Depot project %s: %w", reference.DepotProjectID, err)
		}
		if lookup.Exists {
			continue
		}

		rows, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
			return h.db.ClearProjectDepotIDIfMatches(runCtx, db.ClearProjectDepotIDIfMatchesParams{
				UpdatedAt:      sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				ProjectID:      reference.ProjectID,
				DepotProjectID: sql.NullString{String: reference.DepotProjectID, Valid: true},
			})
		}, restate.WithName("clear missing Depot project reference"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return cleared, fmt.Errorf("clear missing Depot project reference %s: %w", reference.ProjectID, err)
		}
		cleared += int32(rows)
	}
	return cleared, nil
}

func (h *Handler) deleteOrphanImages(ctx restate.ObjectContext, now time.Time, databaseImages map[string]struct{}, depotImages map[string]DepotImage, limit int32) (int32, error) {
	if limit <= 0 {
		return 0, nil
	}
	cutoff := now.Add(-depotGracePeriod)
	candidates := make(map[string]DepotImage)
	for tag, image := range depotImages {
		if _, managed := managedImageDeploymentID(tag); !managed {
			continue
		}
		if _, referenced := databaseImages[tag]; referenced {
			continue
		}
		if image.Digest == "" {
			logger.Error("Depot image has no digest; garbage collection will retain it", "tag", tag)
			continue
		}
		if image.PushedAt.IsZero() {
			logger.Error("Depot image has no valid pushed time; garbage collection will retain it", "tag", tag)
			continue
		}
		if image.PushedAt.After(cutoff) {
			continue
		}
		candidates[tag] = image
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	// Depot does not expose a GetImage endpoint, so take a fresh full registry
	// snapshot before deleting anything from the candidate set.
	recheckedImages, err := h.loadDepotImages(ctx)
	if err != nil {
		return 0, fmt.Errorf("recheck Depot images before deletion: %w", err)
	}

	candidateTags := make([]string, 0, len(candidates))
	for tag := range candidates {
		candidateTags = append(candidateTags, tag)
	}
	slices.Sort(candidateTags)

	var deleted int32
	for _, tag := range candidateTags {
		if deleted >= limit {
			break
		}
		candidate := candidates[tag]
		image, exists := recheckedImages[tag]
		deploymentID, managed := managedImageDeploymentID(image.Tag)
		if !exists || !managed || image.Digest != candidate.Digest {
			continue
		}
		if image.PushedAt.IsZero() || image.PushedAt.After(cutoff) {
			continue
		}

		// A managed image tag contains its owning deployment ID. Rechecking that
		// row closes the gap between pushing an image and saving its tag in MySQL.
		ownerExists, lookupErr := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
			_, findErr := h.db.FindDeploymentById(runCtx, deploymentID)
			if findErr == nil {
				return true, nil
			}
			if db.IsNotFound(findErr) {
				return false, nil
			}
			return false, findErr
		}, restate.WithName("recheck Depot image owner"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if lookupErr != nil {
			return deleted, fmt.Errorf("recheck Depot image owner %s: %w", tag, lookupErr)
		}
		if ownerExists {
			continue
		}

		deleteErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return h.depot.DeleteImage(runCtx, h.registryProjectID, tag)
		}, restate.WithName("delete unreferenced Depot image"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if deleteErr != nil {
			return deleted, fmt.Errorf("delete Depot image %s: %w", tag, deleteErr)
		}
		deleted++
	}
	return deleted, nil
}

func managedImageDeploymentID(tag string) (string, bool) {
	projectID, deploymentID, ok := strings.Cut(tag, "-")
	if !ok || strings.Contains(deploymentID, "-") || !isResourceID(projectID, uid.ProjectPrefix) || !isResourceID(deploymentID, uid.DeploymentPrefix) {
		return "", false
	}
	return deploymentID, true
}

func (h *Handler) deleteOrphanProjects(ctx restate.ObjectContext, now time.Time, databaseProjects []projectReference, depotProjects map[string]DepotProject, limit int32) (int32, error) {
	if limit <= 0 {
		return 0, nil
	}
	referencedProjects := make(map[string]struct{}, len(databaseProjects))
	for _, reference := range databaseProjects {
		referencedProjects[reference.DepotProjectID] = struct{}{}
	}

	cutoff := now.Add(-depotGracePeriod)
	var candidates []string
	for id, project := range depotProjects {
		if id == h.registryProjectID || !h.isManagedBuildProject(project.Name) {
			continue
		}
		if _, referenced := referencedProjects[id]; referenced {
			continue
		}
		if project.CreatedAt.IsZero() {
			logger.Error("Depot project has no valid creation time; garbage collection will retain it",
				"depot_project_id", id,
				"name", project.Name,
			)
			continue
		}
		if project.CreatedAt.After(cutoff) {
			continue
		}
		candidates = append(candidates, id)
	}
	slices.Sort(candidates)

	var deleted int32
	for _, projectID := range candidates {
		if deleted >= limit {
			break
		}
		referenced, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
			return h.db.ProjectDepotIDExists(runCtx, sql.NullString{String: projectID, Valid: true})
		}, restate.WithName("recheck Depot project reference"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return deleted, fmt.Errorf("recheck Depot project %s: %w", projectID, err)
		}
		if referenced {
			continue
		}

		lookup, err := restate.Run(ctx, func(runCtx restate.RunContext) (depotProjectLookup, error) {
			project, exists, getErr := h.depot.GetProject(runCtx, projectID)
			return depotProjectLookup{Project: project, Exists: exists}, getErr
		}, restate.WithName("recheck unreferenced Depot project"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return deleted, fmt.Errorf("get Depot project %s: %w", projectID, err)
		}
		if !lookup.Exists || lookup.Project.ID != projectID || projectID == h.registryProjectID {
			continue
		}
		if !h.isManagedBuildProject(lookup.Project.Name) || lookup.Project.CreatedAt.IsZero() || lookup.Project.CreatedAt.After(cutoff) {
			continue
		}

		deleteErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return h.depot.DeleteProject(runCtx, projectID)
		}, restate.WithName("delete unreferenced Depot project"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if deleteErr != nil {
			return deleted, fmt.Errorf("delete Depot project %s: %w", projectID, deleteErr)
		}
		deleted++
	}
	return deleted, nil
}

func (h *Handler) isManagedBuildProject(name string) bool {
	projectID, ok := strings.CutPrefix(name, h.projectPrefix+"-")
	return ok && isResourceID(projectID, uid.ProjectPrefix)
}

func isResourceID(id string, prefix uid.Prefix) bool {
	suffix, ok := strings.CutPrefix(id, string(prefix)+"_")
	if !ok || suffix == "" {
		return false
	}
	for i := range len(suffix) {
		char := suffix[i]
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
