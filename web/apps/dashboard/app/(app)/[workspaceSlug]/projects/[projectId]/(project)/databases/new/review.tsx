"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { ENVIRONMENT_KINDS } from "@/lib/queries/environments";
import type { ReactNode } from "react";
import {
  type Engine,
  type EnvironmentBranches,
  type RegionId,
  branchSummary,
  engineCopy,
  regionLabel,
} from "../database-model";

export function SummaryRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 px-4 py-3 text-sm">
      <span className="font-medium text-gray-12">{label}</span>
      <span className="min-w-0 text-right text-gray-11">{children}</span>
    </div>
  );
}

export function SummaryList({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col divide-y divide-grayA-4 rounded-lg border bg-raised">
      {children}
    </div>
  );
}

export function ReviewNote({ children }: { children?: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <p className="text-xs leading-5 text-gray-11">
        Size, backups and billing are managed on PlanetScale.
      </p>
      {children}
    </div>
  );
}

export function ReviewSummary({
  nameLabel,
  name,
  org,
  engine,
  region,
  branches,
  price,
  isNew,
}: {
  nameLabel: string;
  name: string;
  org: string;
  engine: Engine;
  region: RegionId;
  branches: EnvironmentBranches;
  price?: number;
  isNew?: (branch: string) => boolean;
}) {
  return (
    <SummaryList>
      <SummaryRow label={nameLabel}>{name}</SummaryRow>
      <SummaryRow label="Organization">{org}</SummaryRow>
      <SummaryRow label="Engine">{engineCopy[engine].label}</SummaryRow>
      <SummaryRow label="Region">{regionLabel[region]}</SummaryRow>
      {ENVIRONMENT_KINDS.map((env) => (
        <SummaryRow key={env} label={environmentName(env)}>
          {branchSummary(branches, env)}
          {isNew?.(branches[env].branch) ? " · New branch" : null}
        </SummaryRow>
      ))}
      <SummaryRow label="Billing">
        Billed by PlanetScale to {org}
        {price === undefined ? null : ` · $${price}/mo`}
      </SummaryRow>
    </SummaryList>
  );
}
