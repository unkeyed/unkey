package metrics

type noop struct {
}

func NewNoop() Metrics {
	return &noop{}

}

func (n *noop) Close() {}

func (n *noop) ReportHttpRequest(r HttpRequestReport) {}

func (n *noop) ReportCacheHealth(r CacheHealthReport)         {}
func (n *noop) ReportKeyVerification(r KeyVerificationReport) {}
func (n *noop) ReportDatabaseLatency(r DatabaseLatencyReport) {}
func (n *noop) ReportCacheHit(r CacheHitReport)               {}
