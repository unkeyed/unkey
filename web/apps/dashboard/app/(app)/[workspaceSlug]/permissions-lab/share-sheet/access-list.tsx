"use client";

/**
 * "Who has access" rows for the share sheet. Each principal shows the grants
 * that reach the selected resource: the action, whether it is direct or via a
 * role, and whether it is inherited through a pattern. Only direct grants on
 * exactly this resource are revocable here; everything else explains itself
 * through a tooltip so the model stays honest.
 */

import { Button, InfoTooltip, cn } from "@unkey/ui";
import { UrnText } from "../components/urn-display";
import type { AccessRow, MatchedGrant } from "./access-model";
import { isRevocableHere } from "./access-model";

function lockReason(grant: MatchedGrant): string {
  if (!grant.direct && grant.viaRoles.length > 0) {
    const names = grant.viaRoles.map((r) => r.name).join(", ");
    const suffix = grant.inherited
      ? ` It reaches this resource through the pattern ${grant.resourcePattern}.`
      : "";
    return `Granted via role ${names}. Detach the role or edit it to change access.${suffix}`;
  }
  return `Granted by pattern ${grant.resourcePattern}, which covers more than this resource. Edit the grant itself to narrow it.`;
}

function GrantLine({
  grant,
  onRevoke,
}: {
  grant: MatchedGrant;
  onRevoke: () => void;
}) {
  const revocable = isRevocableHere(grant);
  return (
    <div className="flex items-center gap-2 min-w-0">
      <span
        className={cn(
          "shrink-0 rounded px-1.5 py-0.5 font-mono text-[11px]",
          grant.action === "*" ? "bg-errorA-3 text-error-11" : "bg-grayA-3 text-gray-12",
        )}
      >
        {grant.action === "*" ? "all actions" : grant.action}
      </span>

      {grant.direct && (
        <span className="shrink-0 rounded border border-grayA-4 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-gray-11">
          direct
        </span>
      )}
      {grant.viaRoles.map((role) => (
        <span
          key={role.id}
          className="shrink-0 rounded bg-infoA-3 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-info-11"
        >
          via role {role.name}
        </span>
      ))}
      {grant.inherited && (
        <InfoTooltip
          position={{ side: "top" }}
          content={
            <div className="flex flex-col gap-1.5 text-left">
              <span className="text-gray-11">
                Access comes from a wider grant, not from this resource:
              </span>
              <UrnText value={grant.permission} />
            </div>
          }
        >
          <span className="shrink-0 cursor-help rounded bg-warningA-3 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-warning-11 border-b border-dotted border-warning-11/50">
            inherited
          </span>
        </InfoTooltip>
      )}

      <span className="grow" />

      {revocable ? (
        <Button
          variant="ghost"
          size="sm"
          onClick={onRevoke}
          className="shrink-0 text-gray-11 hover:text-error-11"
        >
          Revoke
        </Button>
      ) : (
        <InfoTooltip
          position={{ side: "left" }}
          content={<div className="max-w-[320px] text-left text-gray-12">{lockReason(grant)}</div>}
        >
          <span className="shrink-0 cursor-help text-[11px] text-gray-9 border-b border-dotted border-grayA-6">
            Managed elsewhere
          </span>
        </InfoTooltip>
      )}
    </div>
  );
}

export function AccessList({
  rows,
  onRevoke,
}: {
  rows: AccessRow[];
  onRevoke: (principalID: string, grant: MatchedGrant) => void;
}) {
  return (
    <ul className="divide-y divide-grayA-3">
      {rows.map((row) => (
        <li key={row.principal.id} className="flex gap-3 px-4 py-3">
          <span
            aria-hidden="true"
            className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full bg-grayA-3 text-xs font-medium text-gray-12"
          >
            {row.principal.name.charAt(0).toUpperCase()}
          </span>
          <div className="min-w-0 grow flex flex-col gap-1.5">
            <div className="flex items-baseline gap-2 min-w-0">
              <span className="truncate text-[13px] font-medium text-gray-12">
                {row.principal.name}
              </span>
              <span className="shrink-0 font-mono text-[11px] text-gray-9">{row.principal.id}</span>
            </div>
            <div className="flex flex-col gap-1">
              {row.grants.map((grant) => (
                <GrantLine
                  key={grant.permission}
                  grant={grant}
                  onRevoke={() => onRevoke(row.principal.id, grant)}
                />
              ))}
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}
