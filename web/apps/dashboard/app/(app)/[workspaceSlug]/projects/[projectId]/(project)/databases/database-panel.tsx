"use client";

import { Planetscale } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerDescription,
  Button,
  Loading,
  SettingsDangerZone,
  SettingsGroups,
  SettingsZoneRow,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { DatabaseActivity } from "./database-activity";
import { type Database, planetscaleUrls, regionLabel } from "./database-model";
import { DatabaseConfiguration } from "./database-panel-sections";
import { DatabaseStatusBadge } from "./database-status-badge";
import { DeleteDatabaseDialog } from "./delete-database-dialog";
import { EngineTile } from "./engine-icon";

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

  return (
    <>
      <SlidePanelHeader className="items-center">
        <div className="flex min-w-0 items-center gap-3">
          <EngineTile engine={db.engine} size="md" />
          <SlidePanelTitle className="truncate">{db.name}</SlidePanelTitle>
          <DatabaseStatusBadge status={db.status} />
        </div>
        <SlidePanelCloseButton />
      </SlidePanelHeader>

      <SlidePanelContent className="overflow-y-auto">
        <SettingsGroups className="pt-6 pb-6">
          {match(db)
            .with({ status: "provisioning" }, () => (
              <AlertBanner>
                <Loading className="text-gray-11" />
                <AlertBannerDescription>
                  Provisioning {db.name} in {regionLabel[db.region]}.
                </AlertBannerDescription>
              </AlertBanner>
            ))
            .with({ status: "ready" }, ({ instance }) => (
              <DatabaseActivity vitals={instance.vitals} />
            ))
            .exhaustive()}
          <DatabaseConfiguration db={db} />

          <SettingsDangerZone>
            <SettingsZoneRow
              title="Delete this database"
              description="Deletes the database, its branches and backups."
              action={{ label: "Delete database", onClick: () => setConfirmDelete(true) }}
            />
          </SettingsDangerZone>
        </SettingsGroups>
      </SlidePanelContent>

      <SlidePanelFooter>
        <Button
          variant="primary"
          size="md"
          className="w-full"
          render={
            <Link
              href={planetscaleUrls.database({ org: db.planetscale.org, name: db.name })}
              target="_blank"
              rel="noopener noreferrer"
            />
          }
        >
          <Planetscale className="size-3.5" />
          Manage in PlanetScale
        </Button>
      </SlidePanelFooter>

      <DeleteDatabaseDialog
        db={db}
        isOpen={confirmDelete}
        onOpenChange={setConfirmDelete}
        onDelete={onDelete}
      />
    </>
  );
}
