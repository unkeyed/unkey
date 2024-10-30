package gossip

// _testSimulateFailure is a test helper function that simulates a failure in the cluster
// by shutting down the connect server, so other members can no longer ping it.
func (s *clusterServer) _testSimulateFailure() {
	close(s.close)

}
