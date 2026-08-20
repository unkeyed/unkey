package cron_test

import (
	"context"
	"database/sql"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploymentgc"
)

const (
	testRegistryRepository = "registry.depot.dev/registry-project"
	testProjectPrefix      = "builds-test"
	testProjectID          = "proj_projectgc"
	testReferencedTag      = testProjectID + "-d_reference"
	testOrphanTag          = testProjectID + "-d_orphanabc"
	testChangedTag         = testProjectID + "-d_changedab"
	testRestoredTag        = testProjectID + "-d_restoredx"
	testBuildingTag        = testProjectID + "-d_buildingx"
)

func TestDeploymentGarbageCollection_RetainsHistoryAndReconcilesDepot(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	young := now.Add(-time.Hour)
	depot := newFakeDepot(
		[]deploymentgc.DepotImage{
			{Tag: testReferencedTag, Digest: "sha256:referenced", PushedAt: old},
			{Tag: testOrphanTag, Digest: "sha256:orphan", PushedAt: old},
			{Tag: testChangedTag, Digest: "sha256:old", PushedAt: old},
			{Tag: testRestoredTag, Digest: "sha256:restored", PushedAt: old},
			{Tag: testBuildingTag, Digest: "sha256:building", PushedAt: old},
			{Tag: testProjectID + "-d_youngabcd", Digest: "sha256:young", PushedAt: young},
			{Tag: testProjectID + "-d_nodigestx", PushedAt: old},
			{Tag: testProjectID + "-d_notimeabc", Digest: "sha256:notime"},
			{Tag: "foreign", Digest: "sha256:foreign", PushedAt: old},
		},
		[]deploymentgc.DepotProject{
			{ID: "depot-referenced", Name: testProjectPrefix + "-" + testProjectID, CreatedAt: old},
			{ID: "depot-transient", Name: testProjectPrefix + "-proj_transient", CreatedAt: old},
			{ID: "depot-orphan", Name: testProjectPrefix + "-proj_orphanabc", CreatedAt: old},
			{ID: "depot-restored", Name: testProjectPrefix + "-proj_restoredx", CreatedAt: old},
			{ID: "depot-young", Name: testProjectPrefix + "-proj_youngabcd", CreatedAt: young},
			{ID: "depot-no-time", Name: testProjectPrefix + "-proj_notimeabc"},
			{ID: "registry-project", Name: testProjectPrefix + "-proj_registryx", CreatedAt: old},
			{ID: "foreign-project", Name: "builds-canary-proj_foreign", CreatedAt: old},
			{ID: "malformed-project", Name: testProjectPrefix + "-manual-project", CreatedAt: old},
		},
	)
	h := harness.New(t, harness.WithDeploymentGC(deploymentgc.Config{
		Depot:              depot,
		RegistryRepository: testRegistryRepository,
		RegistryProjectID:  "registry-project",
		ProjectPrefix:      testProjectPrefix,
	}))

	workspace := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               testProjectID,
		WorkspaceID:      workspace.ID,
		Name:             "retention",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "retention",
		Slug:        "retention",
	})
	production := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		DeleteProtection: false,
	})

	productionDeployments := make([]db.Deployment, 0, 13)
	for i := range 13 {
		createdAt := now.Add(time.Duration(-40+i) * 24 * time.Hour).UnixMilli()
		if i == 12 {
			createdAt = now.Add(-24 * time.Hour).UnixMilli()
		}
		deployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			AppID:         app.ID,
			EnvironmentID: production.ID,
			Status:        mysqltype.DeploymentsStatusStopped,
			CreatedAt:     createdAt,
		})
		setDeploymentDesiredState(t, h, deployment.ID, mysqltype.DeploymentsDesiredStateStopped, deployment.CreatedAt)
		productionDeployments = append(productionDeployments, deployment)
	}
	require.NoError(t, h.DB.UpdateAppDeployments(h.Ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: productionDeployments[0].ID, Valid: true},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		AppID:               app.ID,
	}))
	activeFailedDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: production.ID,
		Status:        mysqltype.DeploymentsStatusFailed,
		CreatedAt:     now.Add(-60 * 24 * time.Hour).UnixMilli(),
	})
	region := h.Seed.CreateRegion(h.Ctx, seed.CreateRegionRequest{Name: "gc-test", Platform: "gc-test"})
	h.Seed.CreateInstance(h.Ctx, seed.CreateInstanceRequest{
		DeploymentID: activeFailedDeployment.ID,
		WorkspaceID:  workspace.ID,
		ProjectID:    project.ID,
		AppID:        app.ID,
		RegionID:     region.ID,
		Address:      "127.0.0.1",
	})
	wakingStoppedDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: production.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		CreatedAt:     now.Add(-60 * 24 * time.Hour).UnixMilli(),
	})

	preview := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Kind:             mysqltype.EnvironmentKindPreview,
		DeleteProtection: false,
	})
	oldPreview := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		CreatedAt:     now.Add(-31 * 24 * time.Hour).UnixMilli(),
	})
	setDeploymentDesiredState(t, h, oldPreview.ID, mysqltype.DeploymentsDesiredStateStopped, oldPreview.CreatedAt)
	recentPreview := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		CreatedAt:     now.Add(-29 * 24 * time.Hour).UnixMilli(),
	})
	setDeploymentDesiredState(t, h, recentPreview.ID, mysqltype.DeploymentsDesiredStateStopped, recentPreview.CreatedAt)
	referencedImageDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "d_reference",
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	missingImageDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	setDeploymentImage(t, h, referencedImageDeployment.ID, testRegistryRepository+":"+testReferencedTag)
	setDeploymentImage(t, h, missingImageDeployment.ID, testRegistryRepository+":missing")
	restoredImageDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "d_restoredx",
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	buildingImageDeployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "d_buildingx",
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: "depot-referenced", Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:             project.ID,
	}))
	staleProject := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "stale",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: "depot-missing", Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:             staleProject.ID,
	}))
	transientProject := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "transient",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: "depot-transient", Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:             transientProject.ID,
	}))
	depot.omitProjectFromList("depot-transient")
	restoredProject := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "restored",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})

	depot.setImageForRecheck(deploymentgc.DepotImage{
		Tag:      testChangedTag,
		Digest:   "sha256:new",
		PushedAt: old,
	})
	depot.setBeforeImageRecheck(func() error {
		if err := h.DB.UpdateDeploymentImage(h.Ctx, db.UpdateDeploymentImageParams{
			Image:     sql.NullString{String: testRegistryRepository + ":" + testRestoredTag, Valid: true},
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:        restoredImageDeployment.ID,
		}); err != nil {
			return err
		}
		return h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
			DepotProjectID: sql.NullString{String: "depot-restored", Valid: true},
			UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:             restoredProject.ID,
		})
	})

	response, err := hydrav1.NewCronServiceIngressClient(h.Restate, "deployment-garbage-collection").
		RunDeploymentGarbageCollection().
		Request(h.Ctx, &hydrav1.RunDeploymentGarbageCollectionRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(3), response.GetDeploymentsDispatched())
	require.Equal(t, int32(1), response.GetMissingDepotImages())
	require.Equal(t, int32(1), response.GetStaleProjectReferencesCleared())
	require.Equal(t, int32(2), response.GetDepotResourcesDeleted(), "deleted images=%v projects=%v", depot.deletedImageTags(), depot.deletedProjectIDs())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, firstErr := h.DB.FindDeploymentById(h.Ctx, productionDeployments[1].ID)
		_, secondErr := h.DB.FindDeploymentById(h.Ctx, productionDeployments[2].ID)
		_, previewErr := h.DB.FindDeploymentById(h.Ctx, oldPreview.ID)
		assert.True(c, db.IsNotFound(firstErr))
		assert.True(c, db.IsNotFound(secondErr))
		assert.True(c, db.IsNotFound(previewErr))
	}, 30*time.Second, 100*time.Millisecond)

	_, err = h.DB.FindDeploymentById(h.Ctx, productionDeployments[0].ID)
	require.NoError(t, err, "the current deployment must be retained")
	_, err = h.DB.FindDeploymentById(h.Ctx, productionDeployments[3].ID)
	require.NoError(t, err, "the newest ten successful revisions must be retained")
	_, err = h.DB.FindDeploymentById(h.Ctx, activeFailedDeployment.ID)
	require.NoError(t, err, "deployments with instances must be retained")
	_, err = h.DB.FindDeploymentById(h.Ctx, wakingStoppedDeployment.ID)
	require.NoError(t, err, "stopped deployments being woken must be retained")
	_, err = h.DB.FindDeploymentById(h.Ctx, recentPreview.ID)
	require.NoError(t, err, "preview deployments younger than 30 days must be retained")
	_, err = h.DB.FindDeploymentById(h.Ctx, buildingImageDeployment.ID)
	require.NoError(t, err, "deployments whose images are not recorded yet must retain their Depot image")

	updatedStaleProject, err := h.DB.FindProjectById(h.Ctx, staleProject.ID)
	require.NoError(t, err)
	require.False(t, updatedStaleProject.DepotProjectID.Valid)
	updatedTransientProject, err := h.DB.FindProjectById(h.Ctx, transientProject.ID)
	require.NoError(t, err)
	require.Equal(t, "depot-transient", updatedTransientProject.DepotProjectID.String)
	require.Equal(t, []string{testOrphanTag}, depot.deletedImageTags())
	require.Equal(t, []string{"depot-orphan"}, depot.deletedProjectIDs())
	restoredProject, err = h.DB.FindProjectById(h.Ctx, restoredProject.ID)
	require.NoError(t, err)
	require.Equal(t, "depot-restored", restoredProject.DepotProjectID.String)
}

func setDeploymentImage(t *testing.T, h *harness.Harness, deploymentID, image string) {
	t.Helper()
	require.NoError(t, h.DB.UpdateDeploymentImage(h.Ctx, db.UpdateDeploymentImageParams{
		Image:     sql.NullString{String: image, Valid: true},
		UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:        deploymentID,
	}))
}

func setDeploymentDesiredState(t *testing.T, h *harness.Harness, deploymentID string, desiredState mysqltype.DeploymentsDesiredState, updatedAt int64) {
	t.Helper()
	require.NoError(t, h.DB.UpdateDeploymentDesiredState(h.Ctx, db.UpdateDeploymentDesiredStateParams{
		ID:           deploymentID,
		DesiredState: desiredState,
		UpdatedAt:    sql.NullInt64{Int64: updatedAt, Valid: true},
	}))
}

type fakeDepot struct {
	mu                 sync.Mutex
	images             map[string]deploymentgc.DepotImage
	projects           map[string]deploymentgc.DepotProject
	imagesForRecheck   map[string]deploymentgc.DepotImage
	omittedProjects    map[string]struct{}
	beforeImageRecheck func() error
	imageListCalls     int
	deletedImages      []string
	deletedProjects    []string
}

func newFakeDepot(images []deploymentgc.DepotImage, projects []deploymentgc.DepotProject) *fakeDepot {
	fake := &fakeDepot{
		images:           make(map[string]deploymentgc.DepotImage, len(images)),
		projects:         make(map[string]deploymentgc.DepotProject, len(projects)),
		imagesForRecheck: make(map[string]deploymentgc.DepotImage),
		omittedProjects:  make(map[string]struct{}),
	}
	for _, image := range images {
		fake.images[image.Tag] = image
	}
	for _, project := range projects {
		fake.projects[project.ID] = project
	}
	return fake
}

func (f *fakeDepot) ListImages(context.Context, string, string) ([]deploymentgc.DepotImage, string, error) {
	f.mu.Lock()
	f.imageListCalls++
	if f.imageListCalls == 2 {
		for tag, image := range f.imagesForRecheck {
			f.images[tag] = image
		}
	}
	beforeRecheck := f.beforeImageRecheck
	isRecheck := f.imageListCalls == 2
	f.mu.Unlock()
	if isRecheck && beforeRecheck != nil {
		if err := beforeRecheck(); err != nil {
			return nil, "", err
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	images := make([]deploymentgc.DepotImage, 0, len(f.images))
	for _, image := range f.images {
		images = append(images, image)
	}
	return images, "", nil
}

func (f *fakeDepot) setImageForRecheck(image deploymentgc.DepotImage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.imagesForRecheck[image.Tag] = image
}

func (f *fakeDepot) setBeforeImageRecheck(callback func() error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.beforeImageRecheck = callback
}

func (f *fakeDepot) DeleteImage(_ context.Context, _ string, tag string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.images, tag)
	f.deletedImages = append(f.deletedImages, tag)
	return nil
}

func (f *fakeDepot) ListProjects(context.Context, string) ([]deploymentgc.DepotProject, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	projects := make([]deploymentgc.DepotProject, 0, len(f.projects))
	for id, project := range f.projects {
		if _, omitted := f.omittedProjects[id]; omitted {
			continue
		}
		projects = append(projects, project)
	}
	return projects, "", nil
}

func (f *fakeDepot) omitProjectFromList(projectID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.omittedProjects[projectID] = struct{}{}
}

func (f *fakeDepot) GetProject(_ context.Context, projectID string) (deploymentgc.DepotProject, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	project, ok := f.projects[projectID]
	return project, ok, nil
}

func (f *fakeDepot) DeleteProject(_ context.Context, projectID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.projects, projectID)
	f.deletedProjects = append(f.deletedProjects, projectID)
	return nil
}

func (f *fakeDepot) deletedImageTags() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.deletedImages)
}

func (f *fakeDepot) deletedProjectIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.deletedProjects)
}
