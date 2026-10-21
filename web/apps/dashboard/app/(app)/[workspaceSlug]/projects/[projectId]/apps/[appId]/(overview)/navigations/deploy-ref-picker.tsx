"use client";

import {
  type BranchOption,
  useBranchOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/use-branch-options";
import type { BranchSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/use-search-branches";
import { EnvironmentSelect } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-select";
import { FormCombobox } from "@/components/ui/form-combobox";
import { IconCodeBranchOutline18 } from "@unkey/icons";
import { Button, FormDescription, FormLabel, RelativeTime } from "@unkey/ui";
import { useState } from "react";

type DeployRefPickerProps = {
  value: string;
  environmentId: string;
  productionBranch: string;
  source: BranchSource | null;
  branches: BranchOption[];
  branchesLoading: boolean;
  refError?: string;
  environmentError?: string;
  onRefChange: (ref: string) => void;
  onEnvironmentChange: (slug: string) => void;
};

export function DeployRefPicker({
  value,
  environmentId,
  productionBranch,
  source,
  branches,
  branchesLoading,
  refError,
  environmentError,
  onRefChange,
  onEnvironmentChange,
}: DeployRefPickerProps) {
  const [query, setQuery] = useState("");
  const { options, isSearching } = useBranchOptions({
    source,
    branches,
    pinned: productionBranch,
    query,
  });

  const recentBranch = lastPushedBranch(branches, productionBranch);
  const quickPicks = [
    { name: productionBranch, hint: "production branch" },
    ...(recentBranch ? [{ name: recentBranch, hint: "last pushed branch" }] : []),
  ];

  const items = options.map((branch) => ({
    value: branch.name,
    selectedLabel: branch.name,
    label: (
      <span className="flex w-full items-center justify-between gap-2">
        <span className="truncate">{branch.name}</span>
        {branch.name === productionBranch ? (
          <span className="shrink-0 text-xs text-gray-9">production</span>
        ) : branch.lastPushDate ? (
          <span className="shrink-0 text-xs text-gray-9">
            <RelativeTime value={branch.lastPushDate} />
          </span>
        ) : null}
      </span>
    ),
  }));

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col items-start gap-2">
        <FormCombobox
          label="Branch"
          description="Pick a branch, or paste a full commit SHA."
          descriptionPosition="label"
          className="w-full"
          error={refError}
          options={items}
          value={value}
          onSelect={onRefChange}
          onQueryChange={setQuery}
          loading={branchesLoading || isSearching}
          creatable
          createPosition="last"
          leftIcon={<IconCodeBranchOutline18 className="text-gray-11" />}
          placeholder={<span className="text-gray-9">Select a branch</span>}
          searchPlaceholder="Search branches…"
          emptyMessage="No branches found."
        />
        <div className="flex max-w-full flex-wrap gap-1.5">
          {quickPicks.map((pick) => (
            <Button
              key={pick.name}
              type="button"
              variant="outline"
              onClick={() => onRefChange(pick.name)}
              className="h-5 max-w-full gap-1 px-1.5 font-mono text-xs [&_svg]:size-3"
            >
              <IconCodeBranchOutline18 />
              <span className="truncate">{pick.name}</span>
              <span className="sr-only">, {pick.hint}</span>
            </Button>
          ))}
        </div>
      </div>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <FormLabel label="Environment" htmlFor="deploy-environment" />
        <EnvironmentSelect
          id="deploy-environment"
          value={environmentId}
          onValueChange={onEnvironmentChange}
          includeAll={false}
          placeholder="Set from the branch"
          disabled={!value}
          variant={environmentError ? "error" : "default"}
        />
        <FormDescription
          error={environmentError}
          descriptionId="deploy-environment-description"
          errorId="deploy-environment-error"
        />
      </fieldset>
    </div>
  );
}

function lastPushedBranch(branches: readonly BranchOption[], productionBranch: string) {
  let latest: BranchOption | null = null;
  for (const branch of branches) {
    if (branch.name === productionBranch || !branch.lastPushDate) {
      continue;
    }
    if (!latest?.lastPushDate || branch.lastPushDate > latest.lastPushDate) {
      latest = branch;
    }
  }
  return latest?.name ?? null;
}
