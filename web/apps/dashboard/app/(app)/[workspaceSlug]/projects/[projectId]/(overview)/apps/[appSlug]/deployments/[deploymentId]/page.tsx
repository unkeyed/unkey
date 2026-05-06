"use client";

import { useV2bDeploymentsVariant } from "@/hooks/use-v2b-deployments-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { isFakeDeployment } from "@/lib/fake-deployments";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import { ChevronRight } from "@unkey/icons";
import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { useState } from "react";
import { DeploymentLayoutProvider } from "../../../../deployments/[deploymentId]/layout-provider";
import LegacyDeploymentOverview from "../../../../deployments/[deploymentId]/page";
import { FakeDetail } from "./_fake-detail";

/**
 * Per-app deployment detail route.
 *
 * Layout: the outer app-leaf sidebar is contributed by the v2b navbar
 * variant. This page renders a thin tab row + the deployment body. Only
 * the "Overview" tab exists today; the row is kept so adding Logs /
 * Settings later is a 1-line change.
 *
 * Real deployments delegate to the existing legacy detail composition
 * via `DeploymentLayoutProvider` + the legacy `page.tsx` default
 * export, so we inherit build / approval / cancelled / ready / progress
 * views with zero duplication. Mock deployments (id prefix
 * `dep_fake_`) render a smaller `FakeDetail` placeholder because the
 * legacy composition reads from real-data collections.
 */
export default function AppDeploymentDetailPage() {
  const params = useParams();
  const deploymentId = typeof params?.deploymentId === "string" ? params.deploymentId : "";
  const isFake = isFakeDeployment({ id: deploymentId });
  const { variant: subVariant } = useV2bDeploymentsVariant();
  // `c` promotes the deployment crumb up into V2BTopHeader; `d` swaps it
  // for an eyebrow header; `e` (drawer) and `f` (split) provide their own
  // chrome via the layout. Only `a` and `b` keep the original in-page row.
  // Sub-variant `g` puts the tabs into the contextual sidebar instead
  // of the page, so all in-page chrome (breadcrumb / eyebrow / tab row)
  // is suppressed and the active tab is sourced from `?tab=`.
  const isSidebarDrill = subVariant === "g";
  const headerKind: "crumb" | "eyebrow" | "none" = isSidebarDrill
    ? "none"
    : subVariant === "a" || subVariant === "b"
      ? "crumb"
      : subVariant === "d"
        ? "eyebrow"
        : "none";

  // Railless (b) and eyebrow (d) get the expanded in-page tab set with
  // prototype placeholder content. The others keep a single Overview
  // tab so the existing detail body stays the focus. `g` renders no
  // in-page tab row at all — the sidebar carries them.
  const expandedTabs = subVariant === "b" || subVariant === "d";
  const tabs: Array<{ id: TabId; label: string }> = isSidebarDrill
    ? []
    : expandedTabs
      ? [
          { id: "deployment", label: "Deployment" },
          { id: "logs", label: "Logs" },
          { id: "resources", label: "Resources" },
          { id: "source", label: "Source" },
        ]
      : [{ id: "deployment", label: "Overview" }];

  const searchParams = useSearchParams();
  const urlTab = searchParams?.get("tab");
  const [stateTab, setStateTab] = useState<TabId>("deployment");
  const activeTab: TabId = isSidebarDrill ? toTabId(urlTab) : stateTab;

  return (
    <div className="flex h-full min-h-0 w-full flex-col">
      {headerKind === "crumb" ? <DetailBreadcrumb deploymentId={deploymentId} /> : null}
      {headerKind === "eyebrow" ? <EyebrowHeader deploymentId={deploymentId} /> : null}
      {tabs.length > 0 ? (
        <DetailTabs tabs={tabs} activeTab={activeTab} onChange={setStateTab} />
      ) : null}
      <div className="flex-1 overflow-auto">
        {activeTab === "deployment" ? (
          isFake ? (
            <FakeDetail deploymentId={deploymentId} />
          ) : (
            <DeploymentLayoutProvider deploymentId={deploymentId}>
              <LegacyDeploymentOverview />
            </DeploymentLayoutProvider>
          )
        ) : activeTab === "logs" ? (
          <FakeLogs deploymentId={deploymentId} />
        ) : activeTab === "resources" ? (
          <FakeResources />
        ) : (
          <FakeSource />
        )}
      </div>
    </div>
  );
}

function toTabId(value: string | null | undefined): TabId {
  if (value === "logs" || value === "resources" || value === "source") {
    return value;
  }
  return "deployment";
}

function DetailBreadcrumb({ deploymentId }: { deploymentId: string }) {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;

  return (
    <div className="flex shrink-0 items-center gap-1 px-4 pt-3">
      <Link
        href={listHref}
        className="rounded px-1 py-0.5 text-xs font-medium text-gray-11 hover:bg-grayA-3 hover:text-accent-12"
      >
        Deployments
      </Link>
      <ChevronRight className="size-2.5 text-gray-9" iconSize="sm-thin" />
      <span className="px-1 py-0.5 font-mono text-xs font-medium text-accent-12">
        {shortenId(deploymentId)}
      </span>
    </div>
  );
}

/**
 * Sub-variant `d`: a slim back-link kicker above a page H1, in lieu of
 * a breadcrumb chain. The workspace/project/app context still lives in
 * V2BTopHeader, so one back-action is enough wayfinding.
 */
function EyebrowHeader({ deploymentId }: { deploymentId: string }) {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;

  return (
    <div className="flex shrink-0 flex-col gap-1 border-b border-grayA-4 px-6 py-4">
      <Link
        href={listHref}
        className="inline-flex w-fit items-center gap-1 text-[12px] font-medium text-gray-11 hover:text-accent-12"
      >
        <ChevronRight className="size-3 rotate-180" iconSize="sm-thin" />
        Deployments
      </Link>
      <h1 className="font-mono text-lg font-semibold text-accent-12">{shortenId(deploymentId)}</h1>
    </div>
  );
}

type TabId = "deployment" | "logs" | "resources" | "source";

function DetailTabs({
  tabs,
  activeTab,
  onChange,
}: {
  tabs: Array<{ id: TabId; label: string }>;
  activeTab: TabId;
  onChange: (id: TabId) => void;
}) {
  return (
    <div className="border-b border-grayA-4 px-4">
      <nav className="-ml-2 flex items-center gap-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            className={cn(
              "relative px-3 py-3 text-[13px] font-medium transition-colors",
              tab.id === activeTab
                ? "text-accent-12 after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-accent-12"
                : "text-gray-11 hover:text-accent-12",
            )}
          >
            {tab.label}
          </button>
        ))}
      </nav>
    </div>
  );
}

function FakeLogs({ deploymentId }: { deploymentId: string }) {
  const lines = [
    { ts: "12:14:08.412", level: "info", msg: `boot deployment=${shortenId(deploymentId)}` },
    { ts: "12:14:08.418", level: "info", msg: "loading runtime config from /etc/unkey" },
    { ts: "12:14:08.421", level: "info", msg: "vault: ok" },
    { ts: "12:14:08.430", level: "info", msg: "mysql: connected (pool=20, idle=20)" },
    { ts: "12:14:08.514", level: "warn", msg: "clickhouse: handshake retry 1/3" },
    { ts: "12:14:08.612", level: "info", msg: "clickhouse: connected" },
    { ts: "12:14:08.701", level: "info", msg: "listening :7070" },
    { ts: "12:14:09.044", level: "info", msg: "GET /v1/keys.verifyKey 200 12ms" },
    { ts: "12:14:09.220", level: "info", msg: "POST /v1/keys.createKey 200 24ms" },
    { ts: "12:14:09.388", level: "error", msg: "ratelimit: store unreachable, falling back to local" },
    { ts: "12:14:09.402", level: "info", msg: "ratelimit: store recovered" },
    { ts: "12:14:09.910", level: "info", msg: "GET /v1/apis.getApi 200 8ms" },
  ];
  return (
    <div className="px-6 py-6">
      <div className="overflow-hidden rounded-md border border-grayA-4 bg-gray-2 font-mono text-[12px] leading-relaxed">
        {lines.map((l, i) => (
          <div
            key={i}
            className={cn(
              "flex gap-3 px-3 py-1",
              i % 2 === 0 ? "bg-gray-1" : "bg-gray-2",
            )}
          >
            <span className="shrink-0 text-gray-9">{l.ts}</span>
            <span
              className={cn(
                "shrink-0 uppercase",
                l.level === "error"
                  ? "text-error-11"
                  : l.level === "warn"
                    ? "text-warning-11"
                    : "text-gray-11",
              )}
            >
              {l.level}
            </span>
            <span className="text-accent-12">{l.msg}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function FakeResources() {
  const rows = [
    { name: "api", type: "Service", region: "iad1", status: "Healthy", cpu: "12%", mem: "184 MB" },
    { name: "ratelimit", type: "Worker", region: "iad1", status: "Healthy", cpu: "4%", mem: "62 MB" },
    { name: "vault", type: "Service", region: "iad1", status: "Healthy", cpu: "2%", mem: "48 MB" },
    { name: "mysql", type: "Database", region: "iad1", status: "Healthy", cpu: "—", mem: "—" },
    { name: "clickhouse", type: "Database", region: "iad1", status: "Degraded", cpu: "—", mem: "—" },
  ];
  return (
    <div className="px-6 py-6">
      <div className="overflow-hidden rounded-md border border-grayA-4">
        <table className="w-full text-[13px]">
          <thead className="bg-gray-2 text-gray-11">
            <tr>
              {["Name", "Type", "Region", "Status", "CPU", "Memory"].map((h) => (
                <th key={h} className="px-3 py-2 text-left font-medium">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.name} className="border-t border-grayA-4">
                <td className="px-3 py-2 font-mono text-accent-12">{r.name}</td>
                <td className="px-3 py-2 text-gray-11">{r.type}</td>
                <td className="px-3 py-2 text-gray-11">{r.region}</td>
                <td className="px-3 py-2">
                  <span
                    className={cn(
                      "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
                      r.status === "Healthy"
                        ? "bg-success-3 text-success-11"
                        : "bg-warning-3 text-warning-11",
                    )}
                  >
                    {r.status}
                  </span>
                </td>
                <td className="px-3 py-2 font-mono text-gray-11">{r.cpu}</td>
                <td className="px-3 py-2 font-mono text-gray-11">{r.mem}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function FakeSource() {
  return (
    <div className="px-6 py-6">
      <div className="mb-4 flex items-center gap-3 text-[13px]">
        <span className="rounded-full bg-grayA-3 px-2 py-0.5 font-mono text-[11px] text-accent-12">
          main
        </span>
        <span className="font-mono text-gray-11">a3d3976</span>
        <span className="text-gray-11">·</span>
        <span className="text-accent-12">fix(dashboard): call vault encryptBulk for sequential bulk insert</span>
      </div>
      <pre className="overflow-auto rounded-md border border-grayA-4 bg-gray-2 px-4 py-3 font-mono text-[12px] leading-relaxed text-accent-12">
{`diff --git a/web/apps/dashboard/lib/env-vars.ts b/web/apps/dashboard/lib/env-vars.ts
@@ -42,8 +42,12 @@ export async function setEnvVars(input: Input) {
-  for (const v of input.values) {
-    await vault.encrypt({ value: v.value });
-  }
+  const encrypted = await vault.encryptBulk({
+    values: input.values.map((v) => v.value),
+  });
+  for (let i = 0; i < input.values.length; i++) {
+    input.values[i].ciphertext = encrypted[i];
+  }
   await db.transaction(async (tx) => {
     await tx.insert(envVars).values(input.values);
   });
`}
      </pre>
    </div>
  );
}
