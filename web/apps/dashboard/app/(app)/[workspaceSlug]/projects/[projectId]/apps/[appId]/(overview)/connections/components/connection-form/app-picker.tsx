"use client";

import type { App } from "@/lib/collections/deploy/apps";
import { IconCubeOutline18 } from "@unkey/icons";
import { SearchInput } from "@unkey/ui";
import { useState } from "react";

const SOURCE_LABEL: Record<App["sourceType"], string | null> = {
  git: null,
  oci: "Docker image",
  unknown: null,
};

type PickerApp = { id: string; name: string; source: string | null };

export function AppPicker({
  apps,
  projectApps,
  connectedIds,
  onPick,
}: {
  apps: { id: string; name: string }[];
  projectApps: Pick<App, "id" | "sourceType" | "repositoryFullName">[];
  connectedIds: ReadonlySet<string>;
  onPick: (appId: string) => void;
}) {
  const [search, setSearch] = useState("");
  const query = search.trim().toLowerCase();
  const rows: PickerApp[] = apps
    .filter((app) => !connectedIds.has(app.id) && app.name.toLowerCase().includes(query))
    .map((app) => {
      const project = projectApps.find((candidate) => candidate.id === app.id);
      return {
        id: app.id,
        name: app.name,
        source: project ? (project.repositoryFullName ?? SOURCE_LABEL[project.sourceType]) : null,
      };
    });

  return (
    <div
      className="flex flex-col gap-2"
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          const [first] = rows;
          if (first) {
            onPick(first.id);
          }
        }
      }}
    >
      <SearchInput
        label="Search apps"
        placeholder="Search apps"
        value={search}
        onValueChange={setSearch}
      />
      <div className="flex max-h-64 flex-col divide-y divide-grayA-4 overflow-y-auto rounded-lg border border-grayA-4 bg-background">
        {rows.length === 0 ? (
          <p className="px-3 py-2.5 text-sm text-gray-10">
            {query
              ? `No apps match “${search.trim()}”.`
              : "Every app in this project is already connected."}
          </p>
        ) : (
          rows.map((row) => (
            <button
              key={row.id}
              type="button"
              onClick={() => onPick(row.id)}
              className="flex items-center gap-2 px-3 py-2.5 text-left text-sm text-gray-12 transition-colors hover:bg-grayA-2 focus-visible:bg-grayA-3 focus-visible:outline-none"
            >
              <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-11" />
              <span className="truncate">{row.name}</span>
              <span className="ml-auto shrink-0 truncate text-xs text-gray-9">{row.source}</span>
            </button>
          ))
        )}
      </div>
    </div>
  );
}
