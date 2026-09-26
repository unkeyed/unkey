"use client";

import { routes } from "@/lib/navigation/routes";
import { cn } from "@unkey/ui/src/lib/utils";
import { useParams } from "next/navigation";
import { useState } from "react";

type Scenario = {
  group: string;
  label: string;
  note: string;
  orgId: string;
  workspaceSlug: string;
  projectId: string;
};

// Real seeded projects in the shared local DB, one per state the overview has to handle.
export const SCENARIOS: Scenario[] = [
  {
    group: "New",
    label: "Empty project",
    note: "s01-empty",
    orgId: "org_s01empty",
    workspaceSlug: "s01-empty",
    projectId: "proj_s01empty",
  },
  {
    group: "Migrated",
    label: "Keyspaces + ratelimits",
    note: "fireworks · 6 ks, 3 rl",
    orgId: "org_fireworks",
    workspaceSlug: "fireworks",
    projectId: "proj_fireworks",
  },
  {
    group: "Migrated",
    label: "Many keyspaces",
    note: "akuru · 15 ks",
    orgId: "org_akuru",
    workspaceSlug: "akuru",
    projectId: "proj_akuru",
  },
  {
    group: "Active",
    label: "Apps + ratelimit traffic",
    note: "local · container-demo",
    orgId: "org_localdefault",
    workspaceSlug: "local",
    projectId: "proj_ocidemo",
  },
  {
    group: "Active",
    label: "App + keyspace traffic",
    note: "local · default",
    orgId: "org_localdefault",
    workspaceSlug: "local",
    projectId: "proj_UEEUSuMIZ",
  },
  {
    group: "Active",
    label: "App + 3 keyspaces",
    note: "daymoon · default",
    orgId: "org_daymoon",
    workspaceSlug: "daymoon",
    projectId: "proj_daymoon",
  },
  {
    group: "Active",
    label: "SteamSets backend (real shape)",
    note: "16 apps · 2 ks · 7 rl",
    orgId: "org_steamsets",
    workspaceSlug: "steamsets",
    projectId: "proj_steamsets_1",
  },
  {
    group: "Active",
    label: "SteamSets farminspect",
    note: "2 apps · 1 ks",
    orgId: "org_steamsets",
    workspaceSlug: "steamsets",
    projectId: "proj_steamsets_2",
  },
  {
    group: "Active",
    label: "App linked to keyspace",
    note: "pro · platform",
    orgId: "org_pro",
    workspaceSlug: "pro",
    projectId: "proj_pro_platform",
  },
  {
    group: "Problems",
    label: "Every deploy status",
    note: "s07-status · 13 apps",
    orgId: "org_s07status",
    workspaceSlug: "s07-status",
    projectId: "proj_s07a",
  },
  {
    group: "Problems",
    label: "Build failed",
    note: "s11-newer",
    orgId: "org_s11newer",
    workspaceSlug: "s11-newer",
    projectId: "proj_s11c",
  },
  {
    group: "Problems",
    label: "Awaiting approval",
    note: "s11-newer",
    orgId: "org_s11newer",
    workspaceSlug: "s11-newer",
    projectId: "proj_s11f",
  },
  {
    group: "Problems",
    label: "Long names",
    note: "s10-edges",
    orgId: "org_s10edges",
    workspaceSlug: "s10-edges",
    projectId: "proj_s10a",
  },
  {
    group: "Stress",
    label: "20 apps",
    note: "s06-many",
    orgId: "org_s06many",
    workspaceSlug: "s06-many",
    projectId: "proj_s06a",
  },
];

export function ScenarioSwitcher() {
  const params = useParams();
  const [open, setOpen] = useState(false);
  const current = typeof params?.projectId === "string" ? params.projectId : "";
  const groups = [...new Set(SCENARIOS.map((s) => s.group))];

  return (
    <div className="fixed right-4 bottom-4 z-40 flex flex-col items-end gap-2">
      {open && (
        <div className="w-80 overflow-hidden rounded-xl border border-border bg-raised p-1.5 shadow-floating">
          {groups.map((g) => (
            <div key={g} className="mb-1">
              <div className="px-2 pt-2 pb-1 text-[10px] font-medium uppercase tracking-wide text-gray-9">
                {g}
              </div>
              {SCENARIOS.filter((s) => s.group === g).map((s) => (
                <a
                  key={s.projectId}
                  href={routes.auth.switchOrganization({
                    organizationId: s.orgId,
                    returnTo: routes.projects.overview({
                      workspaceSlug: s.workspaceSlug,
                      projectId: s.projectId,
                    }),
                  })}
                  className={cn(
                    "flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-[13px] text-gray-12 hover:bg-grayA-3",
                    s.projectId === current && "bg-grayA-3",
                  )}
                >
                  <span>{s.label}</span>
                  <span className="text-[11px] text-gray-9">{s.note}</span>
                </a>
              ))}
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="rounded-full border border-border bg-raised px-3 py-1.5 text-[11px] font-medium text-gray-11 shadow-floating hover:text-gray-12"
      >
        Scenarios
      </button>
    </div>
  );
}
