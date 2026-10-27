package githubstatus

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploymentnotify"
	workernotification "github.com/unkeyed/unkey/svc/ctrl/worker/notification"
)

type reportedStatus struct {
	DeploymentID   int64
	State          string
	EnvironmentURL string
	LogURL         string
	Description    string
}

type reportedCommitStatus struct {
	SHA         string
	State       string
	LogURL      string
	Description string
	Context     string
}

type fakeGitHub struct {
	githubclient.Noop
	mu               sync.Mutex
	deployments      int64
	comments         int64
	bodies           map[int]string
	statuses         []reportedStatus
	commitStatuses   []reportedCommitStatus
	failCommitStatus bool
	failStatus       bool
	loseCreate       bool
	failFind         bool
	failUpdate       bool
}

func (g *fakeGitHub) CreateDeployment(int64, string, string, string, string, bool) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.deployments++
	return g.deployments, nil
}

func (g *fakeGitHub) CreateDeploymentStatus(_ int64, _ string, id int64, state, environmentURL, logURL, description string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.failStatus {
		return errors.New("connection closed after write")
	}
	g.statuses = append(g.statuses, reportedStatus{DeploymentID: id, State: state, EnvironmentURL: environmentURL, LogURL: logURL, Description: description})
	return nil
}

func (g *fakeGitHub) CreateCommitStatus(_ int64, _ string, sha, state, logURL, description, statusContext string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.commitStatuses = append(g.commitStatuses, reportedCommitStatus{SHA: sha, State: state, LogURL: logURL, Description: description, Context: statusContext})
	if g.failCommitStatus {
		return errors.New("connection closed after commit status write")
	}
	return nil
}

func (g *fakeGitHub) FindPullRequestForBranch(int64, string, string) (int, error) {
	return 17, nil
}

func (g *fakeGitHub) FindBotComment(_ int64, _ string, pr int, _ string) (int64, string, error) {
	g.mu.Lock()
	if g.failFind {
		g.failFind = false
		g.mu.Unlock()
		return 0, "", errors.New("temporary read failure")
	}
	body := g.bodies[pr]
	g.mu.Unlock()
	time.Sleep(10 * time.Millisecond)
	if body == "" {
		return 0, "", nil
	}
	return int64(pr), body, nil
}

func (g *fakeGitHub) CreateIssueComment(_ int64, _ string, pr int, body string) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.comments++
	g.bodies[pr] = body
	if g.loseCreate {
		g.loseCreate = false
		return 0, errors.New("lost create response")
	}
	return int64(pr), nil
}

func (g *fakeGitHub) UpdateIssueComment(_ int64, _ string, id int64, body string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.bodies[int(id)] = body
	if g.failUpdate {
		g.failUpdate = false
		return errors.New("lost update response")
	}
	return nil
}

func TestDeploymentNotificationsThroughRestate(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	_, err = database.RW().ExecContext(t.Context(), "DROP TABLE notification_deliveries, notifications, notification_subscription_targets, notification_subscriptions, notification_destinations, notification_preferences")
	require.NoError(t, err)
	seeder := seed.New(t, database, nil)
	seeder.Seed(t.Context())
	workspace := seeder.Resources.UserWorkspace
	github := &fakeGitHub{bodies: make(map[int]string)}
	svc := New(Config{DB: database, GitHub: github})
	dispatcher, err := workernotification.NewDispatcher(workernotification.DispatcherConfig{
		Definitions: deployment.Definitions(), Integrations: []workernotification.Integration{svc}, DeliveryEnabled: false,
	})
	require.NoError(t, err)
	source, err := deploymentnotify.New(deploymentnotify.Config{
		DB: database, Clock: clock.New(), DashboardURL: "https://dashboard.example.com", Dispatcher: dispatcher,
	})
	require.NoError(t, err)
	enabledDispatcher, err := workernotification.NewDispatcher(workernotification.DispatcherConfig{
		Definitions: deployment.Definitions(), Integrations: []workernotification.Integration{svc}, DeliveryEnabled: true,
	})
	require.NoError(t, err)
	planned := make(chan notification.Event, 1)
	failedPlanner := restate.NewObject(notification.ServiceName).
		Handler("Notify", restate.NewObjectHandler(func(ctx restate.ObjectContext, event notification.Event) (string, error) {
			err := restate.RunVoid(ctx, func(restate.RunContext) error {
				planned <- event
				return restate.ToTerminalError(errors.New("subscription planner unavailable"))
			})
			return "", err
		}))
	bridge := restate.NewService("test.Notifications").
		Handler("Report", restate.NewServiceHandler(func(ctx restate.Context, req deployment.ReportRequest) (string, error) {
			deployment.Report(ctx, req.DeploymentID, req.Change)
			return restate.Object[string](ctx, deployment.ServiceName, req.DeploymentID, "Report").Request(req)
		})).
		Handler("Fanout", restate.NewServiceHandler(enabledDispatcher.Dispatch)).
		Handler("Event", restate.NewServiceHandler(func(ctx restate.Context, event notification.Event) (ReportResult, error) {
			payload, err := deployment.Decode(event)
			if err != nil {
				return ReportResult{}, restate.ToTerminalError(err)
			}
			return restate.Object[ReportResult](ctx, deploymentServiceName, payload.DeploymentID, "Report").Request(event)
		}))
	runtime := containers.Restate(t, bridge, source.Definition(), svc.DeploymentDefinition(), svc.CommentDefinition(), failedPlanner)
	client := ingress.Service[deployment.ReportRequest, string](runtime.IngressClient, "test.Notifications", "Report")
	events := ingress.Service[notification.Event, ReportResult](runtime.IngressClient, "test.Notifications", "Event")

	newApp := func(t *testing.T, name string) (db.Project, db.App, db.Environment) {
		t.Helper()
		project := seeder.CreateProject(t.Context(), seed.CreateProjectRequest{ID: uid.New("project"), WorkspaceID: workspace.ID, Name: name, Slug: name})
		app := seeder.CreateApp(t.Context(), seed.CreateAppRequest{ID: uid.New("app"), WorkspaceID: workspace.ID, ProjectID: project.ID, Name: "API", Slug: "default"})
		environment := seeder.CreateEnvironment(t.Context(), seed.CreateEnvironmentRequest{
			ID: uid.New("env"), WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: app.ID, Slug: "preview", Kind: mysqltype.EnvironmentKindPreview,
		})
		require.NoError(t, database.InsertGithubRepoConnection(t.Context(), db.InsertGithubRepoConnectionParams{
			WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: app.ID, InstallationID: 51, RepositoryID: 71, RepositoryFullName: "unkey/example", CreatedAt: 1,
		}))
		return project, app, environment
	}
	newDeployment := func(t *testing.T, project db.Project, app db.App, environment db.Environment, pr int64) db.Deployment {
		t.Helper()
		row := seeder.CreateDeployment(t.Context(), seed.CreateDeploymentRequest{
			WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: app.ID, EnvironmentID: environment.ID, Status: mysqltype.DeploymentsStatusReady,
			GitCommitSha: sql.NullString{String: "abcdef12345", Valid: true}, GitBranch: sql.NullString{String: "feature", Valid: true},
			PrNumber: sql.NullInt64{Int64: pr, Valid: pr != 0},
		})
		require.NoError(t, database.InsertFrontlineRoute(t.Context(), db.InsertFrontlineRouteParams{
			ID: uid.New("route"), ProjectID: project.ID, AppID: app.ID, EnvironmentID: environment.ID, DeploymentID: row.ID,
			Sticky: db.FrontlineRoutesStickyDeployment, FullyQualifiedDomainName: strings.ToLower(row.ID) + ".example.com", CreatedAt: 1,
		}))
		return row
	}
	makeEvent := func(t *testing.T, row db.Deployment, change deployment.Change) notification.Event {
		t.Helper()
		event, err := deployment.Event(row.WorkspaceID, notification.Resource{ProjectID: row.ProjectID, AppID: row.AppID, EnvironmentID: row.EnvironmentID}, 12345,
			deployment.Payload{DeploymentID: row.ID, Change: change, Project: "project", App: "default", Environment: "preview",
				EnvironmentURL: "https://" + strings.ToLower(row.ID) + ".example.com", LogURL: "https://dashboard.example.com/" + row.ID})
		require.NoError(t, err)
		return event
	}

	t.Run("typed reports preserve order and links without notification tables or email services", func(t *testing.T) {
		project, app, environment := newApp(t, "first")
		row := newDeployment(t, project, app, environment, 0)
		for _, change := range []deployment.Change{deployment.Preparing, deployment.RegionsStarted, deployment.RoutingStarted, deployment.Live, deployment.Live, deployment.RegionsStarted} {
			_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: change})
			require.NoError(t, err)
		}
		github.mu.Lock()
		defer github.mu.Unlock()
		require.EqualValues(t, 1, github.deployments)
		require.EqualValues(t, 1, github.comments)
		require.Len(t, github.statuses, 4)
		require.Equal(t, []string{"pending", "in_progress", "in_progress", "success"}, []string{github.statuses[0].State, github.statuses[1].State, github.statuses[2].State, github.statuses[3].State})
		require.Equal(t, "Configuring routing", github.statuses[2].Description)
		require.Equal(t, "https://"+strings.ToLower(row.ID)+".example.com", github.statuses[3].EnvironmentURL)
		require.Equal(t, fmt.Sprintf("https://dashboard.example.com/%s/projects/%s/apps/%s/deployments/%s", workspace.Slug, project.ID, app.ID, row.ID), github.statuses[3].LogURL)
		require.Len(t, github.commitStatuses, 4)
		require.Equal(t, []string{"pending", "pending", "pending", "success"}, []string{github.commitStatuses[0].State, github.commitStatuses[1].State, github.commitStatuses[2].State, github.commitStatuses[3].State})
		require.Equal(t, reportedCommitStatus{
			SHA: row.GitCommitSha.String, State: "success", LogURL: github.statuses[3].LogURL,
			Description: "Deployment is live", Context: "Unkey / first - preview",
		}, github.commitStatuses[3])
		require.Contains(t, github.bodies[17], "Ready")
		require.Contains(t, github.bodies[17], github.statuses[3].EnvironmentURL)
		require.Contains(t, github.bodies[17], github.statuses[3].LogURL)
	})

	t.Run("build outcomes and skipped promotion do not announce live traffic", func(t *testing.T) {
		type completionCase struct {
			name        string
			status      mysqltype.DeploymentsStatus
			change      deployment.Change
			state       string
			description string
		}
		for _, tc := range []completionCase{
			{name: "build-failed", status: mysqltype.DeploymentsStatusFailed, change: deployment.BuildFailed, state: "failure", description: "Build or image preparation failed"},
			{name: "load-failed", status: mysqltype.DeploymentsStatusFailed, change: deployment.LoadFailed, state: "error", description: "Could not load deployment after build"},
			{name: "cancelled", status: mysqltype.DeploymentsStatusCancelled, change: deployment.Cancelled, state: "error", description: "Deployment stopped: cancelled"},
			{name: "superseded", status: mysqltype.DeploymentsStatusSuperseded, change: deployment.Superseded, state: "error", description: "Deployment stopped: superseded"},
			{name: "promotion-skipped", status: mysqltype.DeploymentsStatusReady, change: deployment.CompletedWithoutPromotion, state: "inactive", description: "Deployment completed without changing live traffic"},
			{name: "newer-deployment", status: mysqltype.DeploymentsStatusSuperseded, change: deployment.CompletedWithoutPromotion, state: "inactive", description: "Deployment completed without changing live traffic"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				project, app, environment := newApp(t, tc.name)
				row := newDeployment(t, project, app, environment, 61)
				require.NoError(t, database.UpdateDeploymentStatus(t.Context(), db.UpdateDeploymentStatusParams{ID: row.ID, Status: tc.status}))
				github.mu.Lock()
				count := len(github.statuses)
				github.mu.Unlock()
				for _, change := range []deployment.Change{tc.change, tc.change, deployment.Live, deployment.Preparing} {
					_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: change})
					require.NoError(t, err)
				}
				github.mu.Lock()
				defer github.mu.Unlock()
				require.Len(t, github.statuses, count+1)
				require.Equal(t, tc.state, github.statuses[count].State)
				require.Equal(t, tc.description, github.statuses[count].Description)
				if tc.state == "inactive" {
					require.Contains(t, github.bodies[61], "Not promoted")
					require.Equal(t, "success", github.commitStatuses[len(github.commitStatuses)-1].State)
				} else {
					require.Equal(t, tc.state, github.commitStatuses[len(github.commitStatuses)-1].State)
				}
			})
		}
	})

	t.Run("concurrent apps retain both PR rows and an older deployment cannot replace its successor", func(t *testing.T) {
		projectA, appA, environmentA := newApp(t, "alpha")
		projectB, appB, environmentB := newApp(t, "beta")
		older := newDeployment(t, projectA, appA, environmentA, 23)
		newer := newDeployment(t, projectA, appA, environmentA, 23)
		other := newDeployment(t, projectB, appB, environmentB, 23)
		failures := make(chan error, 2)
		for _, row := range []db.Deployment{newer, other} {
			go func() {
				_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: deployment.Live})
				failures <- err
			}()
		}
		firstErr, secondErr := <-failures, <-failures
		require.NoError(t, errors.Join(firstErr, secondErr))
		_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: older.ID, Change: deployment.Live})
		require.NoError(t, err)
		github.mu.Lock()
		defer github.mu.Unlock()
		body := github.bodies[23]
		require.Contains(t, body, strings.ToLower(newer.ID)+".example.com")
		require.Contains(t, body, strings.ToLower(other.ID)+".example.com")
		require.NotContains(t, body, strings.ToLower(older.ID)+".example.com")
		require.Equal(t, 2, strings.Count(body, "<!-- row:"))
	})

	t.Run("duplicate events do not post again and conflicting contents fail", func(t *testing.T) {
		project, app, environment := newApp(t, "duplicate")
		row := newDeployment(t, project, app, environment, 29)
		event := makeEvent(t, row, deployment.Live)
		first, err := events.Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, ReportResult{Commit: accepted, Deployment: accepted, Comment: accepted}, first)
		github.mu.Lock()
		count := len(github.statuses)
		commitCount := len(github.commitStatuses)
		github.mu.Unlock()
		second, err := events.Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, first, second)
		event.OccurredAt++
		_, err = events.Request(t.Context(), event)
		require.Error(t, err)
		github.mu.Lock()
		defer github.mu.Unlock()
		require.Len(t, github.statuses, count)
		require.Len(t, github.commitStatuses, commitCount)
	})

	t.Run("status failure does not block the comment and is not reported accepted", func(t *testing.T) {
		project, app, environment := newApp(t, "status-failure")
		row := newDeployment(t, project, app, environment, 31)
		github.mu.Lock()
		github.failStatus = true
		github.mu.Unlock()
		result, err := events.Request(t.Context(), makeEvent(t, row, deployment.Live))
		require.NoError(t, err)
		require.Equal(t, ReportResult{Commit: accepted, Deployment: unknown, Comment: accepted}, result)
		github.mu.Lock()
		github.failStatus = false
		github.mu.Unlock()
	})

	t.Run("an uncertain commit status does not block other reports or resend on replay", func(t *testing.T) {
		project, app, environment := newApp(t, "commit-failure")
		row := newDeployment(t, project, app, environment, 67)
		event := makeEvent(t, row, deployment.Live)
		github.mu.Lock()
		count := len(github.commitStatuses)
		github.failCommitStatus = true
		github.mu.Unlock()
		first, err := events.Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, ReportResult{Commit: unknown, Deployment: accepted, Comment: accepted}, first)
		second, err := events.Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, first, second)
		github.mu.Lock()
		defer github.mu.Unlock()
		github.failCommitStatus = false
		require.Len(t, github.commitStatuses, count+1)
		require.Contains(t, github.bodies[67], "Ready")
	})

	t.Run("lost comment receipt is reconciled on the next update without a second comment", func(t *testing.T) {
		project, app, environment := newApp(t, "lost-receipt")
		row := newDeployment(t, project, app, environment, 37)
		github.mu.Lock()
		count := github.comments
		github.loseCreate = true
		github.mu.Unlock()
		result, err := events.Request(t.Context(), makeEvent(t, row, deployment.RegionsStarted))
		require.NoError(t, err)
		require.Equal(t, unknown, result.Comment)
		result, err = events.Request(t.Context(), makeEvent(t, row, deployment.Live))
		require.NoError(t, err)
		require.Equal(t, accepted, result.Comment)
		github.mu.Lock()
		defer github.mu.Unlock()
		require.Equal(t, count+1, github.comments)
		require.Contains(t, github.bodies[37], "Ready")
	})

	t.Run("disconnected repositories receive no further reports", func(t *testing.T) {
		project, app, environment := newApp(t, "disconnect")
		row := newDeployment(t, project, app, environment, 41)
		_, err := events.Request(t.Context(), makeEvent(t, row, deployment.RegionsStarted))
		require.NoError(t, err)
		require.NoError(t, database.DeleteGithubRepoConnectionsByAppId(t.Context(), app.ID))
		result, err := events.Request(t.Context(), makeEvent(t, row, deployment.Live))
		require.NoError(t, err)
		require.Equal(t, ReportResult{Commit: skipped, Deployment: skipped, Comment: skipped}, result)
	})

	t.Run("subscription planner failure does not block GitHub", func(t *testing.T) {
		project, app, environment := newApp(t, "fanout")
		row := newDeployment(t, project, app, environment, 43)
		event := makeEvent(t, row, deployment.Live)
		id, err := ingress.Service[notification.Event, string](runtime.IngressClient, "test.Notifications", "Fanout").Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, event.ID(), id)
		select {
		case received := <-planned:
			require.Equal(t, event.ID(), received.ID())
		case <-time.After(5 * time.Second):
			t.Fatal("subscription event was not dispatched")
		}
		github.mu.Lock()
		defer github.mu.Unlock()
		require.Contains(t, github.bodies[43], "Ready")
	})

	t.Run("comment reads and updates retry without creating another comment", func(t *testing.T) {
		project, app, environment := newApp(t, "retry-comment")
		row := newDeployment(t, project, app, environment, 47)
		_, err := events.Request(t.Context(), makeEvent(t, row, deployment.RegionsStarted))
		require.NoError(t, err)
		github.mu.Lock()
		count := github.comments
		github.failFind = true
		github.failUpdate = true
		github.mu.Unlock()
		result, err := events.Request(t.Context(), makeEvent(t, row, deployment.Live))
		require.NoError(t, err)
		require.Equal(t, accepted, result.Comment)
		github.mu.Lock()
		defer github.mu.Unlock()
		require.Equal(t, count, github.comments)
		require.Contains(t, github.bodies[47], "Ready")
	})

	t.Run("cancelled and superseded deployments cannot announce success", func(t *testing.T) {
		project, app, environment := newApp(t, "suppressed")
		for _, status := range []mysqltype.DeploymentsStatus{mysqltype.DeploymentsStatusCancelled, mysqltype.DeploymentsStatusSuperseded} {
			t.Run(string(status), func(t *testing.T) {
				row := newDeployment(t, project, app, environment, 53)
				require.NoError(t, database.UpdateDeploymentStatus(t.Context(), db.UpdateDeploymentStatusParams{ID: row.ID, Status: status}))
				id, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: deployment.Live})
				require.NoError(t, err)
				require.Empty(t, id)
				github.mu.Lock()
				defer github.mu.Unlock()
				require.Empty(t, github.bodies[53])
			})
		}
	})

	t.Run("late cancellation and supersession replace progressing statuses", func(t *testing.T) {
		for _, stopped := range []struct {
			status mysqltype.DeploymentsStatus
			change deployment.Change
		}{
			{status: mysqltype.DeploymentsStatusCancelled, change: deployment.Cancelled},
			{status: mysqltype.DeploymentsStatusSuperseded, change: deployment.Superseded},
		} {
			project, app, environment := newApp(t, "late-"+string(stopped.status))
			row := newDeployment(t, project, app, environment, 67)
			_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: deployment.RoutingStarted})
			require.NoError(t, err)
			require.NoError(t, database.UpdateDeploymentStatus(t.Context(), db.UpdateDeploymentStatusParams{ID: row.ID, Status: stopped.status}))
			_, err = client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: deployment.FinalizationFailed})
			require.NoError(t, err)
			github.mu.Lock()
			status := github.statuses[len(github.statuses)-1]
			github.mu.Unlock()
			require.Equal(t, stopped.change.Description(), status.Description)
			require.Equal(t, "error", status.State)
		}
	})

	t.Run("existing GitHub deployment IDs are reused", func(t *testing.T) {
		project, app, environment := newApp(t, "existing-deployment")
		row := newDeployment(t, project, app, environment, 59)
		require.NoError(t, database.UpdateDeploymentGithubDeploymentId(t.Context(), db.UpdateDeploymentGithubDeploymentIdParams{
			ID: row.ID, GithubDeploymentID: sql.NullInt64{Int64: 991, Valid: true},
		}))
		github.mu.Lock()
		count := github.deployments
		github.mu.Unlock()
		_, err := client.Request(t.Context(), deployment.ReportRequest{DeploymentID: row.ID, Change: deployment.Live})
		require.NoError(t, err)
		github.mu.Lock()
		defer github.mu.Unlock()
		require.Equal(t, count, github.deployments)
		require.EqualValues(t, 991, github.statuses[len(github.statuses)-1].DeploymentID)
	})
}
