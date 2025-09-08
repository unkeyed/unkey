import { collection } from "@/lib/collections";
import { ilike, useLiveQuery } from "@tanstack/react-db";
import { Bookmark } from "@unkey/icons";
import { Button, CopyButton, Empty } from "@unkey/ui";
import { useNamespaceListFilters } from "../hooks/use-namespace-list-filters";
import { NamespaceCard } from "./namespace-card";

const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v2/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

export const NamespaceList = () => {
  const { filters } = useNamespaceListFilters();

  const nameFilter = filters.find((filter) => filter.field === "query")?.value ?? "";

  const { data: namespaces } = useLiveQuery(
    (q) =>
      q
        .from({ namespace: collection.ratelimitNamespaces })
        .where(({ namespace }) => ilike(namespace.name, `%${nameFilter}%`))
        .orderBy(({ namespace }) => namespace.id, "desc"),
    [nameFilter],
  );

  if (namespaces.length === 0) {
    return (
      <div className="w-full flex justify-center items-center h-full p-4">
        <Empty className="w-[600px] flex items-start">
          <Empty.Icon />
          <Empty.Title>No Namespaces found</Empty.Title>
          <Empty.Description className="text-left">
            You haven't created any Namespaces yet. Create one by performing a limit request as
            shown below.
          </Empty.Description>
          <div className="w-full mt-8 mb-8">
            <div className="flex items-start gap-4 p-4 bg-gray-2 border border-gray-6 rounded-lg">
              <pre className="flex-1 text-xs text-left overflow-x-auto">
                <code>{EXAMPLE_SNIPPET}</code>
              </pre>
              <CopyButton value={EXAMPLE_SNIPPET} />
            </div>
          </div>
          <Empty.Actions className="mt-4 justify-start">
            <a href="/docs/ratelimiting/introduction" target="_blank" rel="noopener noreferrer">
              <Button className="flex items-center gap-2">
                <Bookmark className="w-4 h-4" />
                Read the docs
              </Button>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5">
        {namespaces.map((namespace) => (
          <NamespaceCard namespace={namespace} key={namespace.id} />
        ))}
      </div>
    </div>
  );
};
