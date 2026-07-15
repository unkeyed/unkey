import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_portal/analytics")({
  component: AnalyticsPage,
});

/**
 * Placeholder Analytics page for the Customer Portal PoC.
 * Full analytics dashboard will use @unkey/api SDK for verification data.
 */
function AnalyticsPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="rounded-lg border border-gray-6 bg-background p-8">
        <h1 className="font-semibold text-2xl text-gray-12">Analytics</h1>
        <p className="mt-2 text-gray-11">Analytics dashboard coming soon</p>

        <div className="mt-6 grid grid-cols-3 gap-4">
          <div className="rounded-md border border-gray-7 border-dashed p-4">
            <span className="text-gray-9 text-xs">Total Requests</span>
            <div className="mt-1 font-semibold text-2xl text-gray-9">—</div>
          </div>
          <div className="rounded-md border border-gray-7 border-dashed p-4">
            <span className="text-gray-9 text-xs">Success Rate</span>
            <div className="mt-1 font-semibold text-2xl text-gray-9">—</div>
          </div>
          <div className="rounded-md border border-gray-7 border-dashed p-4">
            <span className="text-gray-9 text-xs">Error Rate</span>
            <div className="mt-1 font-semibold text-2xl text-gray-9">—</div>
          </div>
        </div>

        <div className="mt-6 rounded-md border border-gray-7 border-dashed p-8 text-center">
          <span className="text-gray-9 text-sm">Usage chart will appear here</span>
        </div>
      </div>
    </main>
  );
}
