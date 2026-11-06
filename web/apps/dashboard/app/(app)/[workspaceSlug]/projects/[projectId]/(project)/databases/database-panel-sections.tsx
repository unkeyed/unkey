"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { ENVIRONMENT_KINDS, type EnvironmentKind } from "@/lib/queries/environments";
import {
  IconCalendarClockOutline18,
  IconClockRotateClockwiseOutline18,
  IconCodeBranchOutline18,
  IconEarthOutline18,
  IconTableCodeOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, RelativeTime, SettingsGroup, SettingsGroupTitle } from "@unkey/ui";
import {
  type Database,
  type DatabaseInstance,
  type EnvironmentBranches,
  clusterCopy,
  engineCopy,
  regionLabel,
  sharesProductionBranch,
} from "./database-model";
import { engineIcon } from "./engine-icon";
import { InfoCard, InfoCell, InfoCells, InfoRow } from "./info-card";

export function DatabaseConfiguration({ db }: { db: Database }) {
  const engine = engineCopy[db.engine];
  const EngineIcon = engineIcon[db.engine];
  const compute = db.planetscale.branches.production.compute;
  const ready = match(db)
    .with({ status: "ready" }, ({ instance }) => ({
      cells: (
        <InfoCells columns={2}>
          <InfoCell
            icon={IconTableCodeOutline18}
            label="Tables"
            value={instance.tableCount.toLocaleString("en")}
          />
          <InfoCell
            icon={IconCodeBranchOutline18}
            label="Branches"
            value={instance.branchCount.toLocaleString("en")}
          />
        </InfoCells>
      ),
      operations: <OperationRows instance={instance} />,
    }))
    .with({ status: "provisioning" }, () => ({ cells: null, operations: null }))
    .exhaustive();
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Configuration</SettingsGroupTitle>
      <InfoCard>
        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
          <span className="flex items-center gap-2 font-medium text-gray-12">
            <EngineIcon className="size-4" />
            {engine.label}
          </span>
          {compute ? (
            <span className="text-gray-11">{clusterCopy[compute.cluster].label}</span>
          ) : null}
        </div>
        {ready.cells}
        <InfoRow icon={IconEarthOutline18} label="Region">
          {regionLabel[db.region]}
        </InfoRow>
        {ENVIRONMENT_KINDS.map((env) => (
          <InfoRow key={env} icon={IconCodeBranchOutline18} label={environmentName(env)}>
            <EnvironmentBranch branches={db.planetscale.branches} env={env} />
          </InfoRow>
        ))}
        {ready.operations}
      </InfoCard>
    </SettingsGroup>
  );
}

function EnvironmentBranch({
  branches,
  env,
}: { branches: EnvironmentBranches; env: EnvironmentKind }) {
  const { branch, compute } = branches[env];
  if (sharesProductionBranch(branches, env)) {
    return <span className="text-gray-11">Same as production</span>;
  }
  return (
    <>
      <Badge variant="code">{branch}</Badge>
      {compute ? <span className="font-mono text-xs text-gray-11">{compute.size}</span> : null}
    </>
  );
}

function OperationRows({ instance }: { instance: DatabaseInstance }) {
  return (
    <>
      <InfoRow icon={IconClockRotateClockwiseOutline18} label="Last backup">
        {instance.lastBackupAt ? <RelativeTime value={instance.lastBackupAt} /> : "No backups yet"}
      </InfoRow>
      <InfoRow icon={IconCalendarClockOutline18} label="Next maintenance">
        {instance.maintenanceAt ? <RelativeTime value={instance.maintenanceAt} /> : "Not scheduled"}
      </InfoRow>
    </>
  );
}
