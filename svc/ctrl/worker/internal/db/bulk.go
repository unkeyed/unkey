package db

import "context"

// InsertAuditLogs inserts audit logs sequentially in a single step context/transaction.
func (q *Queries) InsertAuditLogs(ctx context.Context, args []InsertAuditLogParams) error {
	for _, arg := range args {
		if err := q.InsertAuditLog(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

// InsertAuditLogTargets inserts audit log targets sequentially in a single step context/transaction.
func (q *Queries) InsertAuditLogTargets(ctx context.Context, args []InsertAuditLogTargetParams) error {
	for _, arg := range args {
		if err := q.InsertAuditLogTarget(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

// InsertDeploymentTopologies inserts deployment topology rows sequentially in a single step context/transaction.
func (q *Queries) InsertDeploymentTopologies(ctx context.Context, args []InsertDeploymentTopologyParams) error {
	for _, arg := range args {
		if err := q.InsertDeploymentTopology(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}
