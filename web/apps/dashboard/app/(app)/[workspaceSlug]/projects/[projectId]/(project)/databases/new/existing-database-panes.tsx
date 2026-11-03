"use client";

import { PaneSubmit } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/pane-actions";
import type { EnvironmentKind } from "@/lib/queries/environments";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import { Button, Loading } from "@unkey/ui";
import Link from "next/link";
import {
  type DatabaseStatus,
  type EnvironmentBranches,
  type PlanetscaleDatabase,
  clusterCopy,
  engineCopy,
  planetscaleUrls,
  regionLabel,
} from "../database-model";
import { type EnvironmentPlan, branchOptions, sourceFlow, withBranch } from "./create-flow-model";
import { PickerField } from "./fields";
import { PreviewItem } from "./preview-item";
import { ReviewNote, ReviewSummary, SummaryList, SummaryRow } from "./review";

const planetscaleState: Record<DatabaseStatus, string> = {
  provisioning: "Initializing",
  ready: "Ready",
};

export function BranchesPane({
  database,
  plan,
  branches,
  onPlan,
  onBranches,
  onContinue,
}: {
  database: PlanetscaleDatabase;
  plan: EnvironmentPlan;
  branches: Record<EnvironmentKind, string>;
  onPlan: (plan: EnvironmentPlan) => void;
  onBranches: (branches: Record<EnvironmentKind, string>) => void;
  onContinue: () => void;
}) {
  const options = branchOptions(database, branches.production);
  const onBranch = (env: EnvironmentKind) => (branch: string) =>
    onBranches(withBranch(database, branches, env, branch));
  return (
    <>
      <PickerField
        label="Production branch"
        value={branches.production}
        options={options.production}
        onChange={onBranch("production")}
      />
      <PreviewItem
        source="existing"
        plan={plan}
        productionBranch={branches.production}
        onPlan={onPlan}
      >
        <PickerField
          label="Branch"
          value={branches.preview}
          options={options.preview}
          onChange={onBranch("preview")}
        />
      </PreviewItem>
      <PaneSubmit onClick={onContinue} />
    </>
  );
}

export function SelectPane({
  databases,
  org,
  selected,
  onSelect,
  onContinue,
}: {
  databases: PlanetscaleDatabase[];
  org: string;
  selected: PlanetscaleDatabase | null;
  onSelect: (name: string) => void;
  onContinue: () => void;
}) {
  const initializing = selected?.state === "provisioning";
  return (
    <>
      {databases.length === 0 ? (
        <p className="rounded-lg border border-dashed px-4 py-3 text-sm text-gray-11">
          No databases to link in {org}. Go back and pick New database.
        </p>
      ) : (
        <PickerField
          label="PlanetScale database"
          value={selected?.name ?? null}
          placeholder="Select a database"
          options={databases.map((db) => ({ value: db.name, label: db.name }))}
          onChange={onSelect}
        />
      )}
      {selected ? (
        <SummaryList>
          <SummaryRow label="Engine">{engineCopy[selected.engine].label}</SummaryRow>
          <SummaryRow label="Region">{regionLabel[selected.region]}</SummaryRow>
          <SummaryRow label="Cluster">{clusterCopy[selected.cluster].label}</SummaryRow>
          <SummaryRow label="Status">{planetscaleState[selected.state]}</SummaryRow>
        </SummaryList>
      ) : null}
      <PaneSubmit
        disabled={selected === null || initializing}
        onClick={onContinue}
        notice={
          initializing ? (
            <span className="flex items-center gap-2 text-xs text-gray-11">
              <Loading size={12} />
              Still initializing on PlanetScale…
            </span>
          ) : null
        }
      />
    </>
  );
}

export function ReviewExistingPane({
  database,
  branches,
  submitting,
  onSubmit,
}: {
  database: PlanetscaleDatabase;
  branches: EnvironmentBranches;
  submitting: boolean;
  onSubmit: () => void;
}) {
  return (
    <>
      <ReviewSummary
        nameLabel="Database"
        name={database.name}
        org={database.org}
        engine={database.engine}
        region={database.region}
        branches={branches}
        isNew={(branch) => !database.branches.includes(branch)}
      />
      <ReviewNote>
        <Button
          variant="outline"
          size="sm"
          render={
            <Link
              href={planetscaleUrls.database({ org: database.org, name: database.name })}
              target="_blank"
              rel="noopener noreferrer"
            />
          }
        >
          Open in PlanetScale
          <IconArrowUpRightOutline12 className="size-3! text-gray-11" />
        </Button>
      </ReviewNote>
      <PaneSubmit loading={submitting} onClick={onSubmit}>
        {sourceFlow.existing.submit}
      </PaneSubmit>
    </>
  );
}
