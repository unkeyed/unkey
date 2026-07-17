"use client";

/**
 * Left rail of the share sheet: a browsable, filterable list of the
 * interesting concrete resources in the mock workspace, grouped by family.
 * Selecting a row drives the main panel.
 */

import { Input, cn } from "@unkey/ui";
import { useMemo, useState } from "react";
import { ALL_RESOURCES, type ConcreteResource, KEYSPACES } from "../lib/mock-data";

const TYPE_SHORT: Record<string, string> = {
  keyspace: "keyspace",
  key: "key",
  ratelimit_namespace: "namespace",
  project: "project",
  app: "app",
  environment: "env",
};

interface RailGroup {
  title: string;
  resources: ConcreteResource[];
}

function buildGroups(): RailGroup[] {
  const byPath = new Map(ALL_RESOURCES.map((r) => [r.path, r]));

  const sampleKeys: ConcreteResource[] = [];
  for (const ks of KEYSPACES) {
    for (const key of ks.keys.slice(0, 2)) {
      const resource = byPath.get(`keyspaces/${ks.id}/keys/${key.id}`);
      if (resource) {
        sampleKeys.push(resource);
      }
    }
  }

  return [
    {
      title: "Keyspaces",
      resources: ALL_RESOURCES.filter((r) => r.type === "keyspace"),
    },
    { title: "Keys", resources: sampleKeys },
    {
      title: "Ratelimit namespaces",
      resources: ALL_RESOURCES.filter((r) => r.type === "ratelimit_namespace"),
    },
    {
      title: "Projects and apps",
      resources: ALL_RESOURCES.filter(
        (r) => r.type === "project" || r.type === "app" || r.type === "environment",
      ),
    },
  ];
}

const GROUPS = buildGroups();

export function ResourceRail({
  selectedPath,
  onSelect,
}: {
  selectedPath: string;
  onSelect: (path: string) => void;
}) {
  const [query, setQuery] = useState("");

  const groups = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (q === "") {
      return GROUPS;
    }
    return GROUPS.map((group) => ({
      ...group,
      resources: group.resources.filter(
        (r) => r.label.toLowerCase().includes(q) || r.path.toLowerCase().includes(q),
      ),
    })).filter((group) => group.resources.length > 0);
  }, [query]);

  return (
    <aside className="w-72 shrink-0 rounded-lg border border-grayA-4 flex flex-col overflow-hidden">
      <div className="p-2 border-b border-grayA-4">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Filter resources"
          aria-label="Filter resources"
        />
      </div>
      <div className="overflow-y-auto max-h-[70vh] py-1.5">
        {groups.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-gray-10">
            No resources match &quot;{query.trim()}&quot;
          </p>
        ) : (
          groups.map((group) => (
            <div key={group.title} className="mb-2 last:mb-0">
              <div className="px-3 pt-2 pb-1 text-[11px] uppercase tracking-wide text-gray-9">
                {group.title}
              </div>
              <ul>
                {group.resources.map((resource) => {
                  // Indent relative to the shallowest resource in the group so
                  // nested families (project > app > env) read as a tree.
                  const groupMinDepth = Math.min(
                    ...group.resources.map((r) => r.path.split("/").length),
                  );
                  const depth = (resource.path.split("/").length - groupMinDepth) / 2;
                  const selected = resource.path === selectedPath;
                  return (
                    <li key={resource.path}>
                      <button
                        type="button"
                        onClick={() => onSelect(resource.path)}
                        aria-current={selected ? "true" : undefined}
                        style={{ paddingLeft: 12 + depth * 12 }}
                        className={cn(
                          "w-full flex items-center justify-between gap-2 pr-3 py-1.5 text-left transition-colors",
                          selected
                            ? "bg-grayA-3 text-gray-12"
                            : "text-gray-11 hover:bg-grayA-2 hover:text-gray-12",
                        )}
                      >
                        <span className="truncate text-[13px]">{resource.label}</span>
                        <span className="shrink-0 font-mono text-[10px] text-gray-9">
                          {TYPE_SHORT[resource.type] ?? resource.type}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))
        )}
      </div>
    </aside>
  );
}
