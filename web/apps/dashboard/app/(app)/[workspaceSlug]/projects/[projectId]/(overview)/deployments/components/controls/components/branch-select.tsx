"use client";

import { CodeBranch, Magnifier } from "@unkey/icons";
import { Checkbox, FormInput, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { useState } from "react";
import { useProjectData } from "../../../../data-provider";
import { useFilters } from "../../../hooks/use-filters";
import { FilterTriggerButton } from "./filter-trigger-button";

export function BranchSelect() {
  const { deployments } = useProjectData();
  const { filters, toggleArrayFilter } = useFilters();
  const [search, setSearch] = useState("");

  const selectedBranches = filters.flatMap((f) =>
    f.field === "branch" && typeof f.value === "string" ? [f.value] : [],
  );

  const branches: string[] = [];
  const seen = new Set<string>();
  for (const b of selectedBranches) {
    if (!seen.has(b)) {
      seen.add(b);
      branches.push(b);
    }
  }
  for (const d of deployments) {
    if (d.gitBranch && !seen.has(d.gitBranch)) {
      seen.add(d.gitBranch);
      branches.push(d.gitBranch);
    }
  }

  const q = search.trim().toLowerCase();
  const visibleBranches = q ? branches.filter((b) => b.toLowerCase().includes(q)) : branches;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <FilterTriggerButton
          icon={<CodeBranch iconSize="md-medium" className="text-gray-9 shrink-0" />}
          label="Branch"
          count={selectedBranches.length}
          isActive={selectedBranches.length > 0}
        />
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
                onClick={() => toggleArrayFilter("branch", branch)}
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
