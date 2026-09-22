package deployment

// reconcileRevision serializes desired state operations for one deployment and
// ignores revisions older than the last successful operation. Equal revisions
// run again so periodic reconciliation can repair cluster drift.
func (c *Controller) reconcileRevision(deploymentID string, revision uint64, reconcile func() error) error {
	unlock := c.desiredStateLocks.Lock(deploymentID)
	defer unlock()

	return c.reconcileRevisionLocked(deploymentID, revision, reconcile)
}

// reconcileRevisionLocked applies revision ordering while the deployment lock is held.
func (c *Controller) reconcileRevisionLocked(deploymentID string, revision uint64, reconcile func() error) error {
	if _, deleted := c.desiredStateDeleted.Load(deploymentID); deleted {
		return nil
	}
	if applied, ok := c.desiredStateRevisions.Load(deploymentID); ok && revision < applied.(uint64) {
		return nil
	}

	if err := reconcile(); err != nil {
		return err
	}
	c.desiredStateRevisions.Store(deploymentID, revision)
	return nil
}
