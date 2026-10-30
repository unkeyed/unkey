import {
  ResourceList,
  ResourceListBody,
  ResourceListFooter,
  ResourceListGrid,
  ResourceListHeader,
  ResourceListSearch,
  ResourceListShowMore,
  ResourceListTable,
  ResourceListTableRow,
  type ResourceListView,
  ResourceListViewToggle,
  SearchInput,
} from "@unkey/ui";
import { useState } from "react";

const columns = [
  { id: "name", header: "Keyspace", width: "minmax(0,1fr)" },
  { id: "id", header: "ID", width: "minmax(0,1fr)" },
];

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
        <ResourceListViewToggle className="md:ml-auto" value={view} onValueChange={setView} />
      </ResourceListHeader>
      {view === "grid" ? (
        <>
          <ResourceListGrid>
            {shown.map((keyspace) => (
              <div key={keyspace.id} className="rounded-lg border bg-raised p-5 shadow-xs">
                <div className="truncate font-medium text-gray-12 text-sm">{keyspace.name}</div>
                <div className="font-mono text-gray-11 text-xs">{keyspace.id}</div>
              </div>
            ))}
          </ResourceListGrid>
          {showMore ? <ResourceListFooter variant="bare">{showMore}</ResourceListFooter> : null}
        </>
      ) : (
        <ResourceListTable
          columns={columns}
          minWidth={360}
          footer={showMore ? <ResourceListFooter>{showMore}</ResourceListFooter> : null}
        >
          <ResourceListBody>
            {shown.map((keyspace) => (
              <ResourceListTableRow
                key={keyspace.id}
                link={
                  <a href={`#${keyspace.id}`}>
                    <span className="sr-only">View {keyspace.name}</span>
                  </a>
                }
              >
                <span className="truncate font-medium text-sm">{keyspace.name}</span>
                <span className="font-mono text-gray-11">{keyspace.id}</span>
              </ResourceListTableRow>
            ))}
          </ResourceListBody>
        </ResourceListTable>
      )}
    </ResourceList>
  );
}
