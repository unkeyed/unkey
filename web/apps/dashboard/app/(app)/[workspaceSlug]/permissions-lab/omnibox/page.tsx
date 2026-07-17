"use client";

import {
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Separator,
} from "@unkey/ui";
import { useState } from "react";
import { PermissionChip, UrnText } from "../components/urn-display";
import { usePermissionsLab } from "../lib/store";
import { type GrantRecord, OmniboxComposer } from "./composer";

const timeFormat = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

export default function OmniboxPage() {
  const lab = usePermissionsLab();
  const [principalID, setPrincipalID] = useState(() => lab.state.principals[0]?.id ?? "");
  const [recent, setRecent] = useState<GrantRecord[]>([]);

  const principal = lab.state.principals.find((p) => p.id === principalID);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Omnibox Grant</PageHeaderTitle>
          <PageHeaderDescription>
            One input, not a form. Compose a resource path and an action with segment-aware
            autocomplete; invalid grammar is hard to type in the first place.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col gap-10 w-full max-w-4xl mx-auto">
          {principal ? (
            <OmniboxComposer
              principalID={principalID}
              onPrincipalChange={setPrincipalID}
              onGranted={(record) => setRecent((prev) => [record, ...prev].slice(0, 20))}
            />
          ) : (
            <Empty>
              <Empty.Title>No root keys</Empty.Title>
              <Empty.Description>
                This workspace has no principals to grant permissions to. Reset the lab data from
                the overview page.
              </Empty.Description>
            </Empty>
          )}

          {principal && <CurrentAccess principalID={principal.id} />}

          <Separator />

          <section className="flex flex-col gap-3">
            <h2 className="text-xs uppercase tracking-wide text-gray-10">
              Recent grants this session
            </h2>
            {recent.length === 0 ? (
              <p className="text-sm text-gray-9">
                Nothing granted this session yet. Your grants will show up here as you make them.
              </p>
            ) : (
              <ul className="flex flex-col rounded-lg border border-grayA-4 divide-y divide-grayA-3">
                {recent.map((record) => (
                  <li
                    key={record.id}
                    className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5"
                  >
                    <span className="font-mono text-[11px] text-gray-9 tabular-nums">
                      {timeFormat.format(record.at)}
                    </span>
                    <span className="text-sm text-gray-12">{record.principalName}</span>
                    <span className="text-gray-8 text-xs" aria-hidden="true">
                      &rarr;
                    </span>
                    <UrnText value={record.permission} />
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </PageBody>
    </PageContainer>
  );
}

function CurrentAccess({ principalID }: { principalID: string }) {
  const lab = usePermissionsLab();
  const principal = lab.state.principals.find((p) => p.id === principalID);
  if (!principal) {
    return null;
  }

  const direct = principal.permissions;
  const effective = lab.effectivePermissions(principalID);
  const viaRoles = effective.filter((p) => !direct.includes(p));
  const legacy = lab.legacyPermissions(principalID);
  const roleNames = principal.roles.map(
    (roleID) => lab.state.roles.find((r) => r.id === roleID)?.name ?? roleID,
  );

  const revoke = (permission: string) => {
    lab.commit("Revoke via omnibox", [{ op: "remove", principalID, permission }]);
  };

  return (
    <section className="flex flex-col gap-5">
      <div className="flex items-baseline gap-2">
        <h2 className="text-xs uppercase tracking-wide text-gray-10">Current access</h2>
        <span className="text-xs text-gray-9">
          {principal.name} &middot; <span className="font-mono">{principal.id}</span>
        </span>
      </div>

      {direct.length === 0 && viaRoles.length === 0 && legacy.length === 0 ? (
        <p className="text-sm text-gray-9">
          No permissions yet. Grant the first one with the omnibox above.
        </p>
      ) : (
        <div className="flex flex-col gap-5">
          <div className="flex flex-col gap-2">
            <h3 className="text-[11px] uppercase tracking-wide text-gray-9">Direct grants</h3>
            {direct.length === 0 ? (
              <p className="text-xs text-gray-9">None. Everything below comes from roles.</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {direct.map((permission) => (
                  <PermissionChip
                    key={permission}
                    value={permission}
                    onRemove={() => revoke(permission)}
                  />
                ))}
              </div>
            )}
          </div>

          {viaRoles.length > 0 && (
            <div className="flex flex-col gap-2">
              <h3 className="text-[11px] uppercase tracking-wide text-gray-9">
                Via roles ({roleNames.join(", ")})
              </h3>
              <div className="flex flex-wrap gap-2">
                {viaRoles.map((permission) => (
                  <PermissionChip key={permission} value={permission} />
                ))}
              </div>
              <p className="text-xs text-gray-9">
                Inherited grants are managed on the role, not here.
              </p>
            </div>
          )}

          {legacy.length > 0 && (
            <div className="flex flex-col gap-2">
              <h3 className="text-[11px] uppercase tracking-wide text-gray-9">Legacy tuples</h3>
              <div className="flex flex-wrap gap-2">
                {legacy.map((permission) => (
                  <PermissionChip key={permission} value={permission} legacy />
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
