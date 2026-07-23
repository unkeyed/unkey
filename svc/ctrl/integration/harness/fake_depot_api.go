package harness

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/unkeyed/unkey/svc/ctrl/internal/depotclient"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/registrysweep"
)

// FakeDepotAPI implements registrysweep.DepotAPI for tests: seed registry
// images and Depot projects, then assert against the recorded deletions.
type FakeDepotAPI struct {
	mu               sync.Mutex
	images           []depotclient.Image
	projects         []depotclient.Project
	deletedTags      []string
	deletedProjects  []string
	pageSize         int
	deleteTagErr     error
	deleteProjectErr error
}

var _ registrysweep.DepotAPI = (*FakeDepotAPI)(nil)

// NewFakeDepotAPI returns an empty fake.
func NewFakeDepotAPI() *FakeDepotAPI {
	return &FakeDepotAPI{
		mu:               sync.Mutex{},
		images:           nil,
		projects:         nil,
		deletedTags:      nil,
		deletedProjects:  nil,
		pageSize:         100,
		deleteTagErr:     nil,
		deleteProjectErr: nil,
	}
}

// SetDeleteTagError configures a persistent tag deletion failure.
func (f *FakeDepotAPI) SetDeleteTagError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteTagErr = err
}

// SetDeleteProjectError configures a persistent project deletion failure.
func (f *FakeDepotAPI) SetDeleteProjectError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteProjectErr = err
}

// SetDepotPageSize controls list pagination for tests.
func (f *FakeDepotAPI) SetDepotPageSize(pageSize int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if pageSize < 1 {
		panic("Depot page size must be positive")
	}
	f.pageSize = pageSize
}

// SeedImage adds one registry image.
func (f *FakeDepotAPI) SeedImage(img depotclient.Image) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images = append(f.images, img)
}

// SeedProject adds one Depot project.
func (f *FakeDepotAPI) SeedProject(p depotclient.Project) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.projects = append(f.projects, p)
}

// ListImages returns the requested page of seeded images.
func (f *FakeDepotAPI) ListImages(_ context.Context, _ string, page int32) ([]depotclient.Image, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if page < 1 {
		return nil, false, fmt.Errorf("page must be positive")
	}
	start := (int(page) - 1) * f.pageSize
	if start >= len(f.images) {
		return nil, false, nil
	}
	end := min(start+f.pageSize, len(f.images))
	return append([]depotclient.Image(nil), f.images[start:end]...), end < len(f.images), nil
}

// DeleteTag records a successful registry tag deletion.
func (f *FakeDepotAPI) DeleteTag(_ context.Context, _, tag string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteTagErr != nil {
		return f.deleteTagErr
	}
	f.deletedTags = append(f.deletedTags, tag)
	for i := range f.images {
		remaining := f.images[i].Tags[:0]
		for _, existing := range f.images[i].Tags {
			if existing != tag {
				remaining = append(remaining, existing)
			}
		}
		f.images[i].Tags = remaining
	}
	return nil
}

// ListProjects returns the requested page of seeded projects.
func (f *FakeDepotAPI) ListProjects(_ context.Context, pageToken string) ([]depotclient.Project, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	start := 0
	if pageToken != "" {
		var err error
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			return nil, "", fmt.Errorf("invalid page token %q", pageToken)
		}
	}
	if start >= len(f.projects) {
		return nil, "", nil
	}
	end := min(start+f.pageSize, len(f.projects))
	next := ""
	if end < len(f.projects) {
		next = strconv.Itoa(end)
	}
	return append([]depotclient.Project(nil), f.projects[start:end]...), next, nil
}

// DeleteProject records a successful Depot project deletion.
func (f *FakeDepotAPI) DeleteProject(_ context.Context, projectID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteProjectErr != nil {
		return f.deleteProjectErr
	}
	f.deletedProjects = append(f.deletedProjects, projectID)
	for i, project := range f.projects {
		if project.ID == projectID {
			f.projects = append(f.projects[:i], f.projects[i+1:]...)
			break
		}
	}
	return nil
}

// HasTag reports whether a seeded image still has tag.
func (f *FakeDepotAPI) HasTag(tag string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, image := range f.images {
		for _, existing := range image.Tags {
			if existing == tag {
				return true
			}
		}
	}
	return false
}

// HasProject reports whether a seeded Depot project still exists.
func (f *FakeDepotAPI) HasProject(projectID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, project := range f.projects {
		if project.ID == projectID {
			return true
		}
	}
	return false
}

// DeletedTags returns a copy of every tag deleted so far.
func (f *FakeDepotAPI) DeletedTags() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deletedTags...)
}

// DeletedProjects returns a copy of every Depot project id deleted so far.
func (f *FakeDepotAPI) DeletedProjects() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deletedProjects...)
}
