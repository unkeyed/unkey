"use client";

import { ChevronDown, CodeBranch, Magnifier } from "@unkey/icons";
import { Checkbox, FormInput, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import { useProjectData } from "../../../../data-provider";
import { useFilters } from "../../../hooks/use-filters";

export function BranchSelect() {
  const { deployments } = useProjectData();
  const { filters, updateFilters } = useFilters();
  const [search, setSearch] = useState("");

  const selectedBranches = filters
    .filter((f) => f.field === "branch")
    .map((f) => f.value as string);

  const branches = useMemo(() => {
    const seen = new Set<string>();
    const ordered: string[] = [];
    for (const branch of selectedBranches) {
      if (!seen.has(branch)) {
        seen.add(branch);
        ordered.push(branch);
      }
    }
    for (const d of deployments) {
      const b = d.gitBranch;
      if (b && !seen.has(b)) {
        seen.add(b);
        ordered.push(b);
      }
    }
    return ordered;
  }, [deployments, selectedBranches]);

  const visibleBranches = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) {
      return branches;
    }
    return branches.filter((b) => b.toLowerCase().includes(q));
  }, [branches, search]);

  const toggleBranch = (branch: string) => {
    const isSelected = selectedBranches.includes(branch);
    const otherFilters = filters.filter((f) => f.field !== "branch");
    const currentBranchFilters = filters.filter((f) => f.field === "branch");

    if (isSelected) {
      const remaining = currentBranchFilters.filter((f) => f.value !== branch);
      updateFilters([...otherFilters, ...remaining]);
    } else {
      updateFilters([
        ...otherFilters,
        ...currentBranchFilters,
        {
          field: "branch",
          id: crypto.randomUUID(),
          operator: "is",
          value: branch,
        },
      ]);
    }
  };

  const count = selectedBranches.length;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex items-center gap-2 h-9 px-3 w-full",
            "bg-gray-1 border border-grayA-4 rounded-lg",
            "text-[13px] text-accent-12 font-normal",
            "hover:bg-gray-2 transition-colors",
            count > 0 && "bg-gray-2",
          )}
        >
          <CodeBranch iconSize="md-medium" className="text-gray-9 shrink-0" />
          <span className="truncate">
            Branch
            {count > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-medium">
                {count}
              </span>
            )}
          </span>
          <ChevronDown className="ml-auto shrink-0" iconSize="md-medium" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-1">
        <div className="p-1">
          <FormInput
            placeholder="Search branches..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="[&_input]:h-8 [&_input]:text-[13px]"
            leftIcon={<Magnifier iconSize="md-medium" className="text-gray-9" />}
          />
        </div>
        <div className="max-h-64 overflow-y-auto">
          {visibleBranches.length === 0 ? (
            <div className="px-2 py-3 text-[13px] text-gray-9 text-center">
              {branches.length === 0 ? "No branches yet" : "No matching branches"}
            </div>
          ) : (
            visibleBranches.map((branch) => (
              <button
                type="button"
                key={branch}
                className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-gray-3 cursor-pointer text-[13px] w-full"
                onClick={() => toggleBranch(branch)}
              >
                <Checkbox
                  variant="primary"
                  size="md"
                  checked={selectedBranches.includes(branch)}
                  tabIndex={-1}
                />
                <span className="text-accent-12 truncate">{branch}</span>
              </button>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
