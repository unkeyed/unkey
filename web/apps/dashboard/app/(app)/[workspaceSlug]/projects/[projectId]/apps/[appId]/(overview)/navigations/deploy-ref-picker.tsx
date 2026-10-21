"use client";

import {
  type BranchOption,
  useBranchOptions,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/use-branch-options";
import type { BranchSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/use-search-branches";
import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { FormCombobox } from "@/components/ui/form-combobox";
import { productionFirst } from "@/lib/collections/deploy/environments";
import { IconChevronDownOutline12, IconCodeBranchOutline18 } from "@unkey/icons";
import {
  Button,
  FormDescription,
  FormLabel,
  RelativeTime,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useState } from "react";
import type { TargetEnvironment } from "./deploy-ref-model";

type DeployRefPickerProps = {
  value: string;
  environment: string;
  environments: TargetEnvironment[];
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
  environment,
  environments,
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
          onChange={(e) => setQuery(e.currentTarget.value)}
          loading={branchesLoading || isSearching}
          creatable
          createPosition="last"
          leftIcon={<IconCodeBranchOutline18 className="text-gray-11" />}
          placeholder={<span className="text-gray-9">Select a branch</span>}
          searchPlaceholder="Search branches…"
          emptyMessage="No branches found."
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => onRefChange(productionBranch)}
          className="h-5 gap-1 px-1.5 font-mono text-xs [&_svg]:size-3"
        >
          <IconCodeBranchOutline18 />
          {productionBranch}
          <span className="sr-only">, production branch</span>
        </Button>
      </div>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <FormLabel label="Environment" htmlFor="deploy-environment" />
        <Select
          value={environment}
          onValueChange={(slug) => slug && onEnvironmentChange(slug)}
          disabled={!value}
        >
          <SelectTrigger
            id="deploy-environment"
            className={environment ? "capitalize" : undefined}
            variant={environmentError ? "error" : "default"}
            rightIcon={<IconChevronDownOutline12 className="absolute right-3 size-3 opacity-70" />}
          >
            <SelectValue placeholder="Set from the branch" />
          </SelectTrigger>
          <SelectContent>
            {productionFirst(environments).map((env) => (
              <SelectItem key={env.id} value={env.slug}>
                <EnvironmentLabel environment={env} className="font-sans text-sm text-gray-12" />
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <FormDescription
          error={environmentError}
          descriptionId="deploy-environment-description"
          errorId="deploy-environment-error"
        />
      </fieldset>
    </div>
  );
}
