"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { Magnifier } from "@unkey/icons";
import { Button, Checkbox } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

// Cap the checkbox list so projects with hundreds of deployments don't render
// an unusable wall. `deployments` arrives newest-first; anything older is
// reachable through the free-text id input.
const LATEST_LIMIT = 10;

type DeploymentFilter = { field: string; value: string | number };

type DeploymentIdFilterProps<T extends DeploymentFilter> = {
  filters: T[];
  updateFilters: (filters: T[]) => void;
  createDeploymentFilter: (value: string) => T;
};

export function DeploymentIdFilter<T extends DeploymentFilter>({
  filters,
  updateFilters,
  createDeploymentFilter,
}: DeploymentIdFilterProps<T>) {
  const { deployments } = useProjectData();

  const latest = useMemo(() => deployments.slice(0, LATEST_LIMIT), [deployments]);
  const latestIds = useMemo(() => new Set(latest.map((d) => d.id)), [latest]);

  const activeIds = useMemo(
    () => filters.filter((f) => f.field === "deploymentId").map((f) => String(f.value)),
    [filters],
  );

  const [checkedIds, setCheckedIds] = useState<Set<string>>(
    () => new Set(activeIds.filter((id) => latestIds.has(id))),
  );

  // Full deployment id for one outside the latest list. Matching is exact, so a
  // partial id won't resolve. Seeded from the first active id with no checkbox so
  // it survives a reopen; if several non-latest ids are active at once (e.g. from
  // AI search), only the first round-trips here.
  const [customId, setCustomId] = useState(() => activeIds.find((id) => !latestIds.has(id)) ?? "");

  const toggle = useCallback((id: string) => {
    setCheckedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const allChecked = latest.length > 0 && latest.every((d) => checkedIds.has(d.id));

  const handleSelectAll = useCallback(() => {
    setCheckedIds((prev) => {
      const everyChecked = latest.length > 0 && latest.every((d) => prev.has(d.id));
      return everyChecked ? new Set() : new Set(latest.map((d) => d.id));
    });
  }, [latest]);

  const handleApply = useCallback(() => {
    const otherFilters = filters.filter((f) => f.field !== "deploymentId");

    const values = new Set<string>(checkedIds);
    const trimmed = customId.trim();
    if (trimmed !== "") {
      values.add(trimmed);
    }

    const deploymentFilters = Array.from(values).map(createDeploymentFilter);

    updateFilters([...otherFilters, ...deploymentFilters]);

    setCheckedIds(new Set(Array.from(values).filter((id) => latestIds.has(id))));
    setCustomId(Array.from(values).find((id) => !latestIds.has(id)) ?? "");
  }, [filters, checkedIds, customId, latestIds, createDeploymentFilter, updateFilters]);

  return (
    <div className="flex flex-col p-2">
      <div className="flex flex-col gap-2 font-mono px-2 py-2">
        {latest.length > 0 && (
          <label htmlFor="deployment-all" className="flex items-center gap-[18px] cursor-pointer">
            <Checkbox
              id="deployment-all"
              checked={allChecked}
              className="size-4 rounded-sm border-gray-4 [&_svg]:size-3"
              onClick={(e) => {
                e.stopPropagation();
                handleSelectAll();
              }}
            />
            <span className="text-xs text-accent-12">
              {allChecked ? "Unselect All" : "Select All"}
            </span>
          </label>
        )}

        {latest.map((deployment) => (
          <label
            key={deployment.id}
            htmlFor={`deployment-${deployment.id}`}
            className="flex gap-[18px] items-center py-1 cursor-pointer"
          >
            <Checkbox
              id={`deployment-${deployment.id}`}
              checked={checkedIds.has(deployment.id)}
              className="size-4 rounded-sm border-gray-4 [&_svg]:size-3"
              onClick={(e) => {
                e.stopPropagation();
                toggle(deployment.id);
              }}
            />
            <span className="text-accent-12 text-xs font-medium truncate">
              {deployment.gitBranch}
            </span>
            <span className="text-accent-9 text-xs font-mono">{deployment.id}</span>
          </label>
        ))}

        <div className="flex gap-[18px] items-center py-1">
          <Magnifier className="text-accent-9 shrink-0" iconSize="lg-medium" />
          <input
            type="text"
            value={customId}
            onChange={(e) => setCustomId(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleApply();
              }
            }}
            placeholder="Enter deployment ID"
            className="text-accent-12 text-xs bg-transparent border-b border-gray-6 outline-none flex-1 font-mono placeholder:text-accent-9/40 focus:border-accent-9"
          />
        </div>
      </div>

      <Button
        variant="primary"
        className="mt-2 w-full h-9 rounded-md focus:ring-4 focus:ring-accent-9 focus:ring-offset-2"
        onClick={handleApply}
      >
        Apply Filter
      </Button>
    </div>
  );
}
