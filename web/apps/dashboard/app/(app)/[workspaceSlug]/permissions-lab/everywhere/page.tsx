"use client";

/**
 * Concept: "No permissions page at all". Permission management dissolves into
 * surfaces people already visit. This simulates a root key detail page where
 * grants live inline (with blast-radius hover cards and an inline composer),
 * plus a global command palette on Cmd+K / Ctrl+K.
 */

import {
  Badge,
  CopyButton,
  Empty,
  KeyboardButton,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { type MockRole, WORKSPACE_NAME } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { AddGrantPopover } from "./add-grant-popover";
import { CommandPalette } from "./command-palette";
import { GrantChip, LegacyGrantChip } from "./grant-chip";

const DEFAULT_PRINCIPAL_ID = "unkey_root_payments";

function formatDate(isoDate: string): string {
  return new Date(`${isoDate}T00:00:00Z`).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}

export default function EverywherePage() {
  const lab = usePermissionsLab();
  const [principalID, setPrincipalID] = useState(DEFAULT_PRINCIPAL_ID);
  const [paletteOpen, setPaletteOpen] = useState(false);

  const principal =
    lab.state.principals.find((p) => p.id === principalID) ?? lab.state.principals[0];

  if (!principal) {
    return (
      <Empty>
        <Empty.Title>No root keys</Empty.Title>
        <Empty.Description>
          The mock workspace has no principals. Reset the lab data from the overview page.
        </Empty.Description>
      </Empty>
    );
  }

  const roles: MockRole[] = principal.roles.flatMap((roleID) => {
    const role = lab.state.roles.find((r) => r.id === roleID);
    return role ? [role] : [];
  });
  const legacy = lab.legacyPermissions(principal.id);

  const removeGrant = (permission: string) => {
    lab.commit(`Revoke ${permission} from ${principal.name}`, [
      { op: "remove", principalID: principal.id, permission },
    ]);
    toast.success("Permission revoked", { description: permission });
  };

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{principal.name}</PageHeaderTitle>
          <PageHeaderDescription>
            Simulated root key detail page for {WORKSPACE_NAME}. There is no permissions page:
            grants live here, and everywhere else, through the command palette.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Select
            value={principal.id}
            items={lab.state.principals.map((p) => ({ value: p.id, label: p.name }))}
            onValueChange={(next) => {
              if (typeof next === "string") {
                setPrincipalID(next);
              }
            }}
          >
            <SelectTrigger className="w-52" aria-label="Switch root key">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {lab.state.principals.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col gap-6">
          <div className="rounded-lg border border-grayA-4 bg-grayA-2 p-4 grid grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="flex flex-col gap-1 min-w-0">
              <span className="text-[11px] uppercase tracking-wide text-gray-9">Key ID</span>
              <div className="flex items-center gap-1.5 min-w-0">
                <span className="font-mono text-xs text-gray-12 truncate">{principal.id}</span>
                <CopyButton value={principal.id} className="size-6 shrink-0" />
              </div>
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-[11px] uppercase tracking-wide text-gray-9">Created</span>
              <span className="text-xs text-gray-12">{formatDate(principal.createdAt)}</span>
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-[11px] uppercase tracking-wide text-gray-9">Kind</span>
              <span>
                <Badge variant="secondary" size="sm">
                  root key
                </Badge>
              </span>
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-[11px] uppercase tracking-wide text-gray-9">Roles</span>
              {roles.length === 0 ? (
                <span className="text-xs text-gray-9">None</span>
              ) : (
                <div className="flex flex-wrap gap-1">
                  {roles.map((role) => (
                    <Badge key={role.id} variant="primary" size="sm" font="mono">
                      {role.name}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="flex flex-col gap-4">
            <div className="flex items-baseline justify-between gap-3">
              <h2 className="text-sm font-medium text-gray-12">Permissions</h2>
              <span className="text-xs text-gray-9">Hover a grant to see its blast radius</span>
            </div>

            <div className="flex flex-col gap-2">
              <span className="text-xs font-medium text-gray-10">Direct grants</span>
              <div className="flex flex-wrap items-center gap-2">
                {principal.permissions.map((permission) => (
                  <GrantChip
                    key={permission}
                    value={permission}
                    source="Direct grant"
                    onRemove={() => removeGrant(permission)}
                  />
                ))}
                <AddGrantPopover principalID={principal.id} principalName={principal.name} />
              </div>
              {principal.permissions.length === 0 && (
                <p className="text-xs text-gray-9">
                  No direct grants. Everything this key can do comes from its roles.
                </p>
              )}
            </div>

            {roles.map((role) => (
              <div key={role.id} className="flex flex-col gap-2">
                <div className="flex items-baseline gap-2">
                  <span className="text-xs font-medium text-gray-10">
                    From role <span className="font-mono">{role.name}</span>
                  </span>
                  <span className="text-[11px] text-gray-9">
                    managed on the role, not on this key
                  </span>
                </div>
                {role.permissions.length === 0 ? (
                  <p className="text-xs text-gray-9">This role has no permissions.</p>
                ) : (
                  <div className="flex flex-wrap items-center gap-2">
                    {role.permissions.map((permission) => (
                      <GrantChip
                        key={permission}
                        value={permission}
                        source={`Role ${role.name}`}
                        dimmed
                      />
                    ))}
                  </div>
                )}
              </div>
            ))}

            {legacy.length > 0 && (
              <div className="flex flex-col gap-2">
                <div className="flex items-baseline gap-2">
                  <span className="text-xs font-medium text-gray-10">Legacy grants</span>
                  <span className="text-[11px] text-gray-9">exact-string match only</span>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {legacy.map((permission) => (
                    <LegacyGrantChip key={permission} value={permission} />
                  ))}
                </div>
              </div>
            )}
          </div>

          <button
            type="button"
            onClick={() => setPaletteOpen(true)}
            className="flex items-center justify-center gap-2 rounded-lg border border-grayA-4 bg-grayA-2 px-4 py-3 transition-colors hover:bg-grayA-3 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-6"
          >
            <span className="text-xs text-gray-10">Press</span>
            <KeyboardButton shortcut="K" modifierKey="⌘" className="max-md:flex" />
            <span className="text-xs text-gray-10">to manage permissions from anywhere</span>
          </button>
        </div>
      </PageBody>

      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </PageContainer>
  );
}
