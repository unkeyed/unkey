"use client";
import { trpc } from "@/lib/trpc/client";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  const { data, isLoading, error } = trpc.settings.rootKeys.query.useQuery({
    limit: 10,
  });

  return (
    <div>
      <Navigation
        workspace={{
          id: "will-add-soon",
          name: "will-add-soon",
        }}
        activePage={{
          href: "root-keys",
          text: "Root Keys",
        }}
      />
      <div className="flex flex-col p-6">
        <h1 className="text-2xl font-bold mb-4 text-foreground">Root Keys</h1>

        {isLoading && <div className="text-grayA-11">Loading root keys...</div>}

        {error && (
          <div className="text-red-11 bg-red-2 border border-red-6 rounded-md p-3">
            Error loading root keys: {error.message}
          </div>
        )}

        {data && (
          <div className="space-y-4">
            <div className="text-sm text-grayA-11">
              Showing {data.keys.length} of {data.total} keys
              {data.hasMore && " (more available)"}
            </div>

            <div className="space-y-3">
              {data.keys.map((key) => (
                <pre key={key.id} className="border-b ">
                  {JSON.stringify(key, null, 2)}
                </pre>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
