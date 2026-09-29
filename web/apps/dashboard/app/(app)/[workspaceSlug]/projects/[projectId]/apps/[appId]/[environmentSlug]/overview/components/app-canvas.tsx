"use client";

import { ago, compact } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/format";
import {
  CanvasCard,
  CanvasCardHeader,
  CanvasConnector,
  CanvasGroup,
  GhostCard,
  MetricHeader,
  MetricRow,
  TONE_TEXT,
  type Tone,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/primitives";
import { CanvasViewport } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/viewport";
import { useOverviewWindow } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/window";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Github,
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconLayers2Outline18,
  IconNodesOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { type ReactNode, useMemo } from "react";
import { RegionFlag } from "../../../components/region-flag";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppEnvironment, useAppScope } from "../../environment-context";
import { usePoliciesData } from "../../policies/hooks/use-policies-data";
import { type CardDomain, useProductionCard } from "./production-card-context";
import { STATUS_META } from "./status";

const POLICY_TYPE_SHORT: Record<Policy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
  firewall: "Firewall",
  openapi: "OpenAPI",
  logging: "Logging",
};

const POLICY_COLS = "grid grid-cols-[minmax(0,1fr)_72px_32px] items-center gap-x-3";

function linkedKeyspaces(policies: PolicyRow[]): string[] {
  return [...new Set(policies.flatMap((p) => (p.type === "keyauth" ? p.keyauth.keyspaces : [])))];
}

type AppCanvasProps = {
  domains: CardDomain[];
  emptyDomain: ReactNode;
  app: ReactNode;
};

export function AppCanvas({ domains, emptyDomain, app }: AppCanvasProps) {
  const { environment } = useAppEnvironment();
  const { rowsByEnv, isLoading: policiesLoading } = usePoliciesData();
  const policies = rowsByEnv[environment.kind];
  const keyAuthIds = useMemo(() => linkedKeyspaces(policies), [policies]);

  const serviceCount = policies.length + keyAuthIds.length;

  return (
    <div className="w-full border-b border-border bg-gray-3/40 last:rounded-b-lg last:border-b-0">
      <CanvasViewport
        label="App canvas"
        className="flex w-full min-w-[980px] items-start px-5 py-5"
      >
        <CanvasGroup icon={<IconEarthOutline18 />} label={`Domains · ${domains.length}`}>
          {domains.length > 0
            ? domains.map((d) => <DomainCard key={d.hostname} domain={d} />)
            : emptyDomain}
        </CanvasGroup>

        <CanvasConnector dashed={domains.length === 0} />

        <CanvasGroup icon={<IconCubeOutline18 />} label="App">
          {app}
        </CanvasGroup>

        <CanvasConnector dashed={serviceCount === 0} />

        <CanvasGroup
          icon={<IconGridOutline18 />}
          label={serviceCount > 0 ? `Services · ${serviceCount}` : "Services"}
        >
          {policiesLoading ? null : (
            <>
              <PoliciesCard policies={policies} />
              <KeyspacesCard keyAuthIds={keyAuthIds} />
            </>
          )}
        </CanvasGroup>
      </CanvasViewport>
    </div>
  );
}

function DomainCard({ domain }: { domain: CardDomain }) {
  return (
    <CanvasCard href={domain.url} external>
      <CanvasCardHeader
        icon={<IconEarthOutline18 />}
        title={domain.hostname}
        mono
        right={
          <span className="text-xs text-gray-9">
            {domain.source === "custom" ? "Custom" : "Generated"}
          </span>
        }
      />
    </CanvasCard>
  );
}

export function AddDomainGhost() {
  const router = useRouter();
  const scope = useAppScope();
  return (
    <GhostCard
      icon={<IconEarthOutline18 />}
      title="Add a custom domain"
      description="Serve this app from your own hostname."
      onClick={() => router.push(`${routes.projects.apps.settings(scope)}#custom-domains`)}
    />
  );
}

export function AppNode() {
  const { deployment, status, isRolledBack, deploymentHref } = useProductionCard();
  const { app } = useAppCurrentDeployment();
  const tone: Tone = status === "failed" || status === "crashing" ? "error" : "default";
  const icon =
    app?.sourceType === "oci" ? (
      <IconLayers2Outline18 />
    ) : app?.repositoryFullName ? (
      <Github />
    ) : (
      <IconTerminalOutline18 />
    );

  const instances = deployment.instances ?? [];
  const running = instances.filter((i) => i.status === "running").length;
  const regions =
    instances.length > 0
      ? [...new Map(instances.map((i) => [i.region.id, i])).values()]
      : deployment.desiredRegions;
  const image = deployment.requestedImage ?? deployment.resolvedImage;

  return (
    <CanvasCard href={deploymentHref} tone={tone}>
      <CanvasCardHeader
        icon={icon}
        title={app?.name ?? "App"}
        right={
          <span className={cn("inline-flex items-center gap-1.5 text-xs", TONE_TEXT[tone])}>
            <span className={cn("size-1.5 rounded-full", STATUS_META[status].dotClass)} />
            {STATUS_META[status].label}
            {isRolledBack && <span className="text-warning-11">· Rolled back</span>}
          </span>
        }
      />
      <div className="flex items-center gap-2 border-b px-3 py-2 text-xs [border-color:var(--divider)]">
        {deployment.source === "git" ? (
          <>
            <IconCodeBranchOutline18 className="size-3.5 shrink-0 text-gray-9" />
            <span className="max-w-24 shrink-0 truncate font-mono text-gray-11">
              {deployment.gitBranch ?? "main"}
            </span>
            <span className="min-w-0 flex-1 truncate text-gray-11">
              {deployment.gitCommitMessage?.split("\n")[0] ?? deployment.gitCommitSha?.slice(0, 7)}
            </span>
          </>
        ) : (
          <>
            <IconLayers2Outline18 className="size-3.5 shrink-0 text-gray-9" />
            <span className="min-w-0 flex-1 truncate font-mono text-gray-11">
              {image ?? "Unknown source"}
            </span>
          </>
        )}
        <span className="shrink-0 text-gray-9">{ago(deployment.createdAt)}</span>
      </div>
      <div className="flex flex-col py-1">
        <Detail label="Regions">
          {regions.length > 0 ? (
            <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1">
              {regions.map((r) => (
                <span key={r.region.id} className="flex items-center gap-1.5">
                  <RegionFlag flagCode={r.flagCode} size="xs" shape="circle" />
                  {r.region.name}
                </span>
              ))}
            </span>
          ) : (
            "—"
          )}
        </Detail>
        <Detail label="Instances">{`${running} running`}</Detail>
        <Detail label="Resources">
          {`${deployment.cpuMillicores / 1000} vCPU · ${deployment.memoryMib} MiB`}
        </Detail>
      </div>
    </CanvasCard>
  );
}

function Detail({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[72px_minmax(0,1fr)] items-start gap-x-3 px-3 py-1.5 text-xs">
      <span className="text-gray-9">{label}</span>
      <span className="min-w-0 text-gray-11 tabular-nums">{children}</span>
    </div>
  );
}

function PoliciesCard({
  policies,
}: {
  policies: PolicyRow[];
}) {
  const router = useRouter();
  const scope = useAppScope();
  const href = routes.projects.apps.policies(scope);
  if (policies.length === 0) {
    return (
      <GhostCard
        icon={<IconShieldKeyOutline18 />}
        title="Add a policy"
        description="Auth, ratelimits and firewall rules at the gateway."
        onClick={() => router.push(href)}
      />
    );
  }
  return (
    <CanvasCard>
      <MetricHeader
        icon={<IconShieldKeyOutline18 />}
        title="Policies"
        href={href}
        columns={["Type", "State"]}
        cols={POLICY_COLS}
      />
      <div className="py-1">
        {policies.map((p) => (
          <MetricRow
            key={p.id}
            href={href}
            cols={POLICY_COLS}
            name={p.name}
            values={[POLICY_TYPE_SHORT[p.type], p.enabled ? "On" : "Off"]}
          />
        ))}
      </div>
    </CanvasCard>
  );
}

function KeyspacesCard({
  keyAuthIds,
}: {
  keyAuthIds: string[];
}) {
  const router = useRouter();
  const scope = useAppScope();
  const project = trpc.deploy.project.overview.useQuery({ projectId: scope.projectId });
  const available = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  const policiesHref = routes.projects.apps.policies(scope);

  if (keyAuthIds.length === 0) {
    return (
      <GhostCard
        icon={<IconNodesOutline18 />}
        title="Protect it with keys"
        description="Add a Key Auth policy to verify API keys."
        onClick={() => router.push(policiesHref)}
      />
    );
  }

  const byKeyAuth = new Map(project.data?.keyspaces.map((k) => [k.keyAuthId, k]));
  return (
    <CanvasCard>
      <MetricHeader
        icon={<IconNodesOutline18 />}
        title="Keyspaces"
        href={routes.apis.list(scope)}
        columns={["Keys", "Verified 7d"]}
      />
      <div className="py-1">
        {keyAuthIds.map((keyAuthId) => {
          const ks = byKeyAuth.get(keyAuthId);
          return (
            <KeyspaceRow
              key={keyAuthId}
              keyAuthId={keyAuthId}
              name={ks?.name ?? available.data?.[keyAuthId]?.api.name ?? keyAuthId}
              keyCount={ks?.keyCount}
              href={ks ? routes.apis.detail({ ...scope, apiId: ks.apiId }) : undefined}
            />
          );
        })}
      </div>
    </CanvasCard>
  );
}

function KeyspaceRow({
  keyAuthId,
  name,
  keyCount,
  href,
}: {
  keyAuthId: string;
  name: string;
  keyCount: number | undefined;
  href: Route | undefined;
}) {
  const window = useOverviewWindow();
  const { data } = trpc.api.overview.timeseries.useQuery(
    { keyspaceId: keyAuthId, ...window, since: "" },
    { trpc: { context: { skipBatch: true } } },
  );
  const total = data?.timeseries?.reduce((a, p) => a + p.y.total, 0);
  return (
    <MetricRow
      href={href}
      name={name}
      values={[keyCount == null ? "—" : compact(keyCount), total == null ? "…" : compact(total)]}
    />
  );
}
