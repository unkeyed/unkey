import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_portal/keys")({
  component: KeysPage,
});

/**
 * Placeholder Keys page for the Customer Portal PoC.
 * Full key management UI (list, create, revoke) will use @unkey/api SDK.
 */
function KeysPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="rounded-lg border border-gray-6 bg-background p-8">
        <h1 className="font-semibold text-2xl text-gray-12">API Keys</h1>
        <p className="mt-2 text-gray-11">Key management coming soon</p>

        <div className="mt-6 flex items-center justify-between rounded-md border border-gray-7 border-dashed p-4">
          <span className="text-gray-9 text-sm">Your API keys will appear here</span>
          <span className="rounded-md bg-gray-3 px-3 py-1.5 text-gray-9 text-sm">+ Create Key</span>
        </div>
      </div>
    </main>
  );
}
