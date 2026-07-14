import {
  Button,
  Input,
  ResourceList,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
} from "@unkey/ui";
import { useState } from "react";

const keys = [
  { name: "Production API", id: "key_2tQx8", status: "Active" },
  { name: "Preview API", id: "key_7mLp3", status: "Active" },
  { name: "Legacy integration", id: "key_4nWs1", status: "Disabled" },
];

export function ResourceListExample() {
  const [query, setQuery] = useState("");
  const [activeOnly, setActiveOnly] = useState(false);
  const [visibleCount, setVisibleCount] = useState(2);
  const filteredKeys = keys.filter(
    (key) =>
      (!activeOnly || key.status === "Active") &&
      `${key.name} ${key.id}`.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const visibleKeys = filteredKeys.slice(0, visibleCount);

  return (
    <ResourceList className="max-w-2xl">
      <ResourceListHeader>
        <Input
          aria-label="Search API keys"
          placeholder="Search API keys"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <Button
          variant="outline"
          size="lg"
          aria-pressed={activeOnly}
          className={activeOnly ? "bg-grayA-3" : undefined}
          onClick={() => setActiveOnly((current) => !current)}
        >
          Active only
        </Button>
      </ResourceListHeader>
      <ResourceListContent>
        <ul className="divide-y divide-grayA-4">
          {visibleKeys.map((key) => (
            <li key={key.id} className="relative flex items-center gap-4 px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="truncate font-medium text-accent-12 text-sm">{key.name}</div>
                <div className="font-mono text-gray-9 text-xs">{key.id}</div>
              </div>
              <span className="text-gray-11 text-xs">{key.status}</span>
            </li>
          ))}
          {visibleKeys.length === 0 ? (
            <li className="px-4 py-8 text-center text-gray-10 text-sm">No API keys found</li>
          ) : null}
        </ul>
        {visibleKeys.length < filteredKeys.length ? (
          <ResourceListFooter>
            <Button variant="outline" onClick={() => setVisibleCount(filteredKeys.length)}>
              Load more
            </Button>
          </ResourceListFooter>
        ) : null}
      </ResourceListContent>
    </ResourceList>
  );
}
