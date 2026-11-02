"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  DetailList,
  DetailRow,
  Loading,
  SettingsDangerZone,
  SettingsGroup,
  SettingsGroupTitle,
  SettingsGroups,
  SettingsZoneRow,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import {
  type Database,
  branchSummary,
  engineCopy,
  planetscaleUrls,
  regionLabel,
  statusCopy,
} from "./database-model";

type Props = {
  db: Database | undefined;
  onClose: () => void;
  onDelete: (db: Database) => void;
};

export function DatabasePanel({ db, onClose, onDelete }: Props) {
  return (
    <SlidePanel isOpen={db !== undefined} onClose={onClose} widthClassName="w-160">
      {db && <DatabaseDetails db={db} onDelete={onDelete} />}
    </SlidePanel>
  );
}

function DatabaseDetails({ db, onDelete }: { db: Database; onDelete: (db: Database) => void }) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const status = statusCopy[db.status];

  return (
    <>
      <SlidePanelHeader className="items-center">
        <div className="flex min-w-0 items-center gap-3">
          <SlidePanelTitle className="truncate">{db.name}</SlidePanelTitle>
          <Badge variant={status.badge} size="sm">
            {status.label}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            render={
              <Link
                href={planetscaleUrls.database({ org: db.planetscale.org, name: db.name })}
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            Manage in PlanetScale
            <IconArrowUpRightOutline12 className="size-3! text-gray-11" />
          </Button>
          <SlidePanelCloseButton />
        </div>
      </SlidePanelHeader>

      <SlidePanelContent className="overflow-y-auto">
        <SettingsGroups className="pt-4 pb-6">
          {db.status === "provisioning" ? (
            <AlertBanner>
              <Loading className="text-gray-11" />
              <AlertBannerDescription>
                Provisioning {db.name} in {regionLabel[db.region]}.
              </AlertBannerDescription>
            </AlertBanner>
          ) : null}

          <SettingsGroup>
            <SettingsGroupTitle>Configuration</SettingsGroupTitle>
            <DetailList>
              <DetailRow label="Engine">
                {engineCopy[db.engine].label} {engineCopy[db.engine].version}
              </DetailRow>
              <DetailRow label="Region">{regionLabel[db.region]}</DetailRow>
              {ENVIRONMENT_KINDS.map((env) => (
                <DetailRow key={env} label={environmentName(env)}>
                  {branchSummary(db.planetscale.branches, env)}
                </DetailRow>
              ))}
            </DetailList>
          </SettingsGroup>

          <SettingsDangerZone>
            <SettingsZoneRow
              title="Delete this database"
              description="Deletes the database, its branches and backups."
              action={{ label: "Delete database", onClick: () => setConfirmDelete(true) }}
            />
          </SettingsDangerZone>
        </SettingsGroups>
      </SlidePanelContent>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {db.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This deletes all data, branches and backups. You cannot undo this action.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep database</AlertDialogCancel>
            <AlertDialogAction color="danger" onClick={() => onDelete(db)}>
              Delete database
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
