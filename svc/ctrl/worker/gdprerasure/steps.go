package gdprerasure

import restate "github.com/restatedev/sdk-go"

// step is one erasure action. A subject's steps run in registry order, and
// every step converges: run again, it finds nothing left and affects zero rows.
type step interface {
	name() string
	execute(ctx restate.Context, w *Workflow, s subject) (int64, error)
}

// mysqlDelete removes the rows of table that match where, in bounded batches.
// where may use the subject placeholders listed in subject.value. by tells
// apart two deletes from the same table in one registry.
type mysqlDelete struct {
	table string
	by    string
	where string
}

func (d mysqlDelete) name() string { return qualify("delete mysql "+d.table, d.by) }

// mysqlUpdate rewrites the rows of table that match where, in bounded batches.
// where must stop matching a row once set has rewritten it, so the batches
// drain.
type mysqlUpdate struct {
	table string
	by    string
	set   string
	where string
}

func (u mysqlUpdate) name() string { return qualify("update mysql "+u.table, u.by) }

// clickhouseDeleteMode is how a ClickHouse delete removes rows.
type clickhouseDeleteMode int

const (
	// deleteLightweight masks rows at once and drops them on the next merge.
	deleteLightweight clickhouseDeleteMode = iota

	// deleteMutation rewrites every part holding a matching row. Tables with a
	// projection reject lightweight deletes, so they need this.
	deleteMutation
)

// clickhouseDelete removes the rows of table that match where.
type clickhouseDelete struct {
	table string
	where string
	mode  clickhouseDeleteMode
}

func (d clickhouseDelete) name() string { return "delete clickhouse " + d.table }

// clickhouseUpdate rewrites the rows of table that match where with a mutation.
// where must stop matching a row once set has rewritten it.
type clickhouseUpdate struct {
	table string
	set   string
	where string
}

func (u clickhouseUpdate) name() string { return "update clickhouse " + u.table }

// teardownCompute stops the workspace's running Deploy compute and waits for it
// to drain, so no row is deleted from under a running pod.
type teardownCompute struct{}

func (teardownCompute) name() string { return "teardown compute" }

// deleteWorkOSOrganization deletes the workspace's WorkOS organization.
type deleteWorkOSOrganization struct{}

func (deleteWorkOSOrganization) name() string { return "delete workos organization" }

// deleteWorkOSUser deletes the dashboard user's WorkOS account.
type deleteWorkOSUser struct{}

func (deleteWorkOSUser) name() string { return "delete workos user" }

func qualify(name, by string) string {
	if by == "" {
		return name
	}
	return name + " by " + by
}
