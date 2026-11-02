"use client";

import { PaneSubmit } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/pane-actions";
import { PaneRow } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/pane-row";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  IconCheckOutline18,
  IconChevronDownOutline18,
  IconCircleCheckOutline18,
  IconDatabaseOutline18,
  IconPlusOutline18,
  Planetscale,
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { type PlanetscaleOrg, planetscaleUrls } from "../database-model";
import { SOURCE_CHOICES, type SourceChoice, sourceFlow } from "./create-flow-model";
import { PickerField } from "./fields";

const sourceIcon: Record<SourceChoice, ReactNode> = {
  new: <IconPlusOutline18 className="size-4" />,
  existing: <IconDatabaseOutline18 className="size-4" />,
};

const PLANETSCALE_SCOPES = [
  "Read organizations",
  "Read databases",
  "Read branches",
  "Create databases",
  "Manage passwords",
  "Manage production branch passwords",
];

export function AccountMenu({ org, onDisconnect }: { org: string; onDisconnect: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="sm" className="shrink-0">
            <Planetscale className="size-3.5" />
            Account
            <IconChevronDownOutline18 className="text-gray-9" />
          </Button>
        }
      />
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuItem
          className="cursor-pointer"
          render={
            <Link href={planetscaleUrls.settings(org)} target="_blank" rel="noopener noreferrer">
              Manage permissions
            </Link>
          }
        />
        <DropdownMenuItem
          className="cursor-pointer text-error-11 focus:bg-error-3 focus:text-error-11"
          onClick={onDisconnect}
        >
          Disconnect
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ConnectPane({
  connecting,
  onConnect,
}: {
  connecting: boolean;
  onConnect: () => void;
}) {
  return (
    <>
      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium text-gray-12">Unkey asks for access to</span>
        <ul className="flex flex-col gap-2 rounded-lg border bg-raised px-4 py-3">
          {PLANETSCALE_SCOPES.map((scope) => (
            <li key={scope} className="flex items-center gap-2 text-sm text-gray-11">
              <IconCheckOutline18 className="size-4 text-gray-12" />
              {scope}
            </li>
          ))}
        </ul>
      </div>
      <PaneSubmit loading={connecting} onClick={onConnect}>
        Connect PlanetScale
      </PaneSubmit>
    </>
  );
}

export function SourcePane({
  source,
  org,
  orgs,
  onSource,
  onOrg,
  onContinue,
}: {
  source: SourceChoice | null;
  org: string;
  orgs: PlanetscaleOrg[];
  onSource: (source: SourceChoice) => void;
  onOrg: (org: string) => void;
  onContinue: () => void;
}) {
  return (
    <>
      <div className="flex flex-col gap-2">
        {SOURCE_CHOICES.map((choice) => (
          <PaneRow
            key={choice}
            variant="outline"
            icon={sourceIcon[choice]}
            label={sourceFlow[choice].label}
            hint={sourceFlow[choice].hint}
            trailing={
              source === choice ? (
                <IconCircleCheckOutline18 className="size-4 text-gray-12" />
              ) : null
            }
            selected={source === choice}
            onClick={() => onSource(choice)}
          />
        ))}
      </div>
      <PickerField
        label="Organization"
        value={org}
        options={orgs.map(({ name }) => ({ value: name, label: name }))}
        onChange={onOrg}
      />
      <PaneSubmit disabled={source === null} onClick={onContinue} />
    </>
  );
}
