/**
 * Placeholder Analytics page for the Customer Portal PoC.
 * Full analytics dashboard (charts, metrics, time-series) is deferred.
 */
export default function AnalyticsPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="rounded-lg border border-gray-6 bg-background p-8">
        <h1 className="text-2xl font-semibold text-gray-12">Analytics</h1>
        <p className="mt-2 text-gray-11">Analytics dashboard coming soon</p>

        <div className="mt-6 grid grid-cols-3 gap-4">
          <div className="rounded-md border border-dashed border-gray-7 p-4">
            <span className="text-xs text-gray-9">Total Requests</span>
            <div className="mt-1 text-2xl font-semibold text-gray-9">—</div>
          </div>
          <div className="rounded-md border border-dashed border-gray-7 p-4">
            <span className="text-xs text-gray-9">Success Rate</span>
            <div className="mt-1 text-2xl font-semibold text-gray-9">—</div>
          </div>
          <div className="rounded-md border border-dashed border-gray-7 p-4">
            <span className="text-xs text-gray-9">Error Rate</span>
            <div className="mt-1 text-2xl font-semibold text-gray-9">—</div>
          </div>
        </div>

        <div className="mt-6 rounded-md border border-dashed border-gray-7 p-8 text-center">
          <span className="text-sm text-gray-9">Usage chart will appear here</span>
        </div>
      </div>
    </main>
  );
}
