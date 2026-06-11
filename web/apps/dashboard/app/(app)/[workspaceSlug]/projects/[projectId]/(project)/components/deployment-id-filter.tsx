"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { Magnifier } from "@unkey/icons";
import { Button, Checkbox } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

const LATEST_LIMIT = 15;

type DeploymentFilter = { field: string; value: string | number };

type DeploymentIdFilterProps<T extends DeploymentFilter> = {
  filters: T[];
  updateFilters: (filters: T[]) => void;
  createDeploymentFilter: (value: string) => T;
};

type Row = { id: string; gitBranch: string | null };

export function DeploymentIdFilter<T extends DeploymentFilter>({
  filters,
  updateFilters,
  createDeploymentFilter,
}: DeploymentIdFilterProps<T>) {
  const { deployments } = useProjectData();

  const byId = useMemo(() => new Map(deployments.map((d) => [d.id, d])), [deployments]);

  const activeIds = useMemo(
    () => filters.filter((f) => f.field === "deploymentId").map((f) => String(f.value)),
    [filters],
  );

  const [checkedIds, setCheckedIds] = useState<Set<string>>(() => new Set(activeIds));
  const [query, setQuery] = useState("");

  // Latest window + checked + committed-filter ids, deduped in order. Including
  // the committed ids keeps an unchecked row visible until Apply removes it.
  const rows = useMemo<Row[]>(() => {
    const latestIds = deployments.slice(0, LATEST_LIMIT).map((d) => d.id);
    const ids = new Set([...latestIds, ...checkedIds, ...activeIds]);
    return [...ids].map((id) => ({ id, gitBranch: byId.get(id)?.gitBranch ?? null }));
  }, [deployments, checkedIds, activeIds, byId]);

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

  const allChecked = rows.length > 0 && rows.every((r) => checkedIds.has(r.id));

  const handleSelectAll = useCallback(() => {
    setCheckedIds((prev) => {
      const next = new Set(prev);
      for (const r of rows) {
        if (allChecked) {
          next.delete(r.id);
        } else {
          next.add(r.id);
        }
      }
      return next;
    });
  }, [rows, allChecked]);

  // Only an exact id in the loaded collection is accepted; typing one is how
  // deployments outside the latest window are reached.
  const queryExactId = useMemo(() => {
    const q = query.trim();
    return byId.has(q) ? q : null;
  }, [query, byId]);

  const selectFromQuery = useCallback(() => {
    if (queryExactId !== null) {
      setCheckedIds((prev) => new Set(prev).add(queryExactId));
      setQuery("");
    }
  }, [queryExactId]);

  const nextIds = useMemo(() => {
    const next = new Set(checkedIds);
    if (queryExactId !== null) {
      next.add(queryExactId);
    }
    return next;
  }, [checkedIds, queryExactId]);

  const hasChanges = nextIds.size !== activeIds.length || activeIds.some((id) => !nextIds.has(id));

  const noMatch = query.trim() !== "" && queryExactId === null;

  const handleApply = useCallback(() => {
    const otherFilters = filters.filter((f) => f.field !== "deploymentId");
    updateFilters([...otherFilters, ...Array.from(nextIds).map(createDeploymentFilter)]);
    setCheckedIds(nextIds);
    setQuery("");
  }, [filters, nextIds, createDeploymentFilter, updateFilters]);

  return (
    <div className="flex flex-col p-2">
      <div className="flex flex-col gap-2 font-mono px-2 py-2 max-h-[480px] overflow-y-auto">
        {rows.length > 0 ? (
          <>
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

            {rows.map((row) => (
              <label
                key={row.id}
                htmlFor={`deployment-${row.id}`}
                className="flex gap-[18px] items-center py-1 cursor-pointer"
              >
                <Checkbox
                  id={`deployment-${row.id}`}
                  checked={checkedIds.has(row.id)}
                  className="size-4 rounded-sm border-gray-4 [&_svg]:size-3"
                  onClick={(e) => {
                    e.stopPropagation();
                    toggle(row.id);
                  }}
                />
                {row.gitBranch && (
                  <span className="text-accent-12 text-xs font-medium truncate">
                    {row.gitBranch}
                  </span>
                )}
                <span className="text-accent-9 text-xs font-mono truncate">{row.id}</span>
              </label>
            ))}
          </>
        ) : (
          <span className="text-accent-9 text-xs py-1">No deployments found</span>
        )}
      </div>

      <div className="flex gap-[18px] items-center px-2 py-1">
        <Magnifier className="text-accent-9 shrink-0" iconSize="lg-medium" />
        <input
          type="text"
          aria-label="Search deployments"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              selectFromQuery();
            }
          }}
          placeholder="Search deployments"
          className="text-accent-12 text-xs bg-transparent border-b border-gray-6 outline-none flex-1 font-mono placeholder:text-accent-9/40 focus:border-accent-9"
        />
      </div>

      {noMatch && (
        <span className="text-error-11 text-xs px-2 pt-1">No match in the deployments</span>
      )}

      <Button
        variant="primary"
        disabled={!hasChanges}
        className="mt-2 w-full h-9 rounded-md focus:ring-4 focus:ring-accent-9 focus:ring-offset-2"
        onClick={handleApply}
      >
        Apply Filter
      </Button>
    </div>
  );
}
