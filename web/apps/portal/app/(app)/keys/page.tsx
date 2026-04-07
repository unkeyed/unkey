/**
 * Placeholder Keys page for the Customer Portal PoC.
 * Full key management UI (list, create, revoke) is deferred.
 */
export default function KeysPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="rounded-lg border border-gray-6 bg-background p-8">
        <h1 className="text-2xl font-semibold text-gray-12">API Keys</h1>
        <p className="mt-2 text-gray-11">Key management coming soon</p>

        <div className="mt-6 flex items-center justify-between rounded-md border border-dashed border-gray-7 p-4">
          <span className="text-sm text-gray-9">Your API keys will appear here</span>
          <span className="rounded-md bg-gray-3 px-3 py-1.5 text-sm text-gray-9">+ Create Key</span>
        </div>
      </div>
    </main>
  );
}
