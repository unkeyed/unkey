import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_portal/docs")({
  component: DocsPage,
});

/**
 * Placeholder Docs page for the Customer Portal PoC.
 * Full OpenAPI documentation rendering is deferred.
 */
function DocsPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="rounded-lg border border-gray-6 bg-background p-8">
        <h1 className="text-2xl font-semibold text-gray-12">API Documentation</h1>
        <p className="mt-2 text-gray-11">Documentation coming soon</p>
      </div>
    </main>
  );
}
