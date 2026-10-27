import {
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
  ResourceListItem,
  ResourceListSearch,
  ResourceListShowMore,
  type ResourceListView,
  ResourceListViewToggle,
  SearchInput,
} from "@unkey/ui";
import { useState } from "react";

const keyspaces = [
  { id: "ks_2tQx8", name: "Production API" },
  { id: "ks_7mLp3", name: "Preview API" },
  { id: "ks_4nWs1", name: "Internal tools" },
];

export default function ResourceListViewToggleExample() {
  const [query, setQuery] = useState("");
  const [view, setView] = useState<ResourceListView>("grid");
  const [visibleCount, setVisibleCount] = useState(2);
  const matches = keyspaces.filter((keyspace) =>
    keyspace.name.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const shown = matches.slice(0, visibleCount);
  const showMore =
    shown.length < matches.length ? (
      <ResourceListShowMore onClick={() => setVisibleCount(matches.length)} />
    ) : null;

  return (
    <ResourceList>
      <ResourceListHeader>
        <ResourceListSearch>
          <SearchInput
            value={query}
            onValueChange={setQuery}
            label="Search keyspaces"
            placeholder="Search keyspaces"
          />
        </ResourceListSearch>
        <ResourceListViewToggle value={view} onValueChange={setView} />
      </ResourceListHeader>
      {view === "grid" ? (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {shown.map((keyspace) => (
              <div key={keyspace.id} className="rounded-lg border bg-raised p-5 shadow-xs">
                <div className="truncate font-medium text-gray-12 text-sm">{keyspace.name}</div>
                <div className="font-mono text-gray-11 text-xs">{keyspace.id}</div>
              </div>
            ))}
          </div>
          {showMore ? <ResourceListFooter variant="bare">{showMore}</ResourceListFooter> : null}
        </>
      ) : (
        <ResourceListContent>
          <ResourceListBody>
            {shown.map((keyspace) => (
              <ResourceListItem key={keyspace.id} className="flex h-12 items-center gap-4 px-4">
                <span className="truncate font-medium text-gray-12 text-sm">{keyspace.name}</span>
                <span className="font-mono text-gray-11 text-xs">{keyspace.id}</span>
              </ResourceListItem>
            ))}
          </ResourceListBody>
          {showMore ? <ResourceListFooter>{showMore}</ResourceListFooter> : null}
        </ResourceListContent>
      )}
    </ResourceList>
  );
}
