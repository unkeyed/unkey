"use client";

import { ago, compact } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/format";
import {
  CanvasCard,
  CanvasCardHeader,
  CanvasConnector,
  CanvasGroup,
  DetailList,
  DetailRow,
  GhostCard,
  MetricHeader,
  MetricRow,
  TONE_TEXT,
  type Tone,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/primitives";
import { CanvasViewport } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/viewport";
import { useOverviewWindow } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/window";
import { DEPLOYMENT_GROUP_COLOR } from "@/lib/collections/deploy/deployment-status";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Github,
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconHardDriveOutline18,
  IconLayers2Outline18,
  IconLink4Outline18,
  IconLocation2Outline18,
  IconMicrochipOutline18,
  IconNodesOutline18,
  IconPlusOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
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
  app: ReactNode;
};

export function AppCanvas({ domains, app }: AppCanvasProps) {
  const { environment } = useAppEnvironment();
  const { rowsByEnv, isLoading: policiesLoading } = usePoliciesData();
  const policies = rowsByEnv[environment.kind];
  const keyAuthIds = useMemo(() => linkedKeyspaces(policies), [policies]);

  const serviceCount = policies.length + keyAuthIds.length;

  return (
    <div className="w-full border-b bg-gray-3/40 last:rounded-b-lg last:border-b-0">
      <CanvasViewport
        label="App canvas"
        className="flex w-full min-w-[980px] items-start px-5 py-5"
      >
        <CanvasGroup icon={<IconEarthOutline18 />} label={`Domains · ${domains.length}`}>
          <DomainsCard domains={domains} />
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

function DomainsCard({ domains }: { domains: CardDomain[] }) {
  const scope = useAppScope();
  const settingsHref = routes.projects.apps.settings({ ...scope, page: "domains" });
  const platform = domains.filter((d) => d.source === "platform");
  const custom = domains.filter((d) => d.source === "custom");
  return (
    <CanvasCard link={{ href: settingsHref, label: "Domain settings" }}>
      <CanvasCardHeader icon={<IconEarthOutline18 />} title="Domains" />
      <DetailList>
        <DetailRow
          icon={<IconEarthOutline18 />}
          label={platform.length > 1 ? "Unkey domains" : "Unkey domain"}
          value={
            platform.length > 0 ? (
              <StatusText dotClass={DEPLOYMENT_GROUP_COLOR.ready}>Enabled</StatusText>
            ) : (
              <StatusText dotClass="bg-gray-7">Pending</StatusText>
            )
          }
        >
          {platform.length > 0 ? (
            platform.map((d) => <HostnameLink key={d.hostname} domain={d} />)
          ) : (
            <span className="text-xs text-gray-9">Assigned on first deploy</span>
          )}
        </DetailRow>
        <DetailRow
          icon={<IconLink4Outline18 />}
          label="Custom domains"
          value={
            custom.length > 0 ? (
              <StatusText dotClass={DEPLOYMENT_GROUP_COLOR.ready}>Verified</StatusText>
            ) : (
              <Link
                href={settingsHref}
                className="relative z-10 flex items-center justify-end gap-1 text-gray-11 hover:text-gray-12"
              >
                <IconPlusOutline18 className="size-3.5" />
                Add domain
              </Link>
            )
          }
        >
          {custom.length > 0 && custom.map((d) => <HostnameLink key={d.hostname} domain={d} />)}
        </DetailRow>
      </DetailList>
    </CanvasCard>
  );
}

function StatusText({ dotClass, children }: { dotClass: string; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={cn("size-1.5 shrink-0 rounded-full", dotClass)} />
      {children}
    </span>
  );
}

function HostnameLink({ domain }: { domain: CardDomain }) {
  return (
    <a
      href={domain.url}
      target="_blank"
      rel="noopener noreferrer"
      className="relative z-10 w-fit max-w-full truncate font-mono text-xs text-gray-9 hover:text-gray-12 hover:underline"
    >
      {domain.hostname}
    </a>
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
    <CanvasCard
      link={{ href: deploymentHref, label: `${app?.name ?? "App"} deployment` }}
      tone={tone}
    >
      <CanvasCardHeader
        icon={icon}
        title={app?.name ?? "App"}
        right={
          <span
            className={cn("inline-flex items-center gap-1.5 text-xs font-medium", TONE_TEXT[tone])}
          >
            <span className={cn("size-1.5 rounded-full", STATUS_META[status].dotClass)} />
            {STATUS_META[status].label}
            {isRolledBack && <span className="text-warning-11">· Rolled back</span>}
          </span>
        }
      />
      <DetailList>
        {deployment.source === "git" ? (
          <>
            <DetailRow
              icon={<IconCodeBranchOutline18 />}
              label="Branch"
              value={<span className="font-mono">{deployment.gitBranch ?? "main"}</span>}
              hint={
                deployment.gitCommitSha && (
                  <span className="font-mono">{deployment.gitCommitSha.slice(0, 7)}</span>
                )
              }
            />
            <DetailRow
              icon={<IconCodeCommitOutline18 />}
              label="Commit"
              value={deployment.gitCommitMessage?.split("\n")[0] ?? "—"}
              hint={ago(deployment.createdAt)}
            />
          </>
        ) : (
          <DetailRow
            icon={<IconLayers2Outline18 />}
            label="Image"
            value={<span className="font-mono">{image ?? "Unknown"}</span>}
            hint={ago(deployment.createdAt)}
          />
        )}
        <DetailRow
          icon={<IconLocation2Outline18 />}
          label="Regions"
          value={
            regions.length > 0 ? (
              <span className="flex items-center justify-end gap-2.5">
                {regions.map((r) => (
                  <span key={r.region.id} className="flex shrink-0 items-center gap-1.5">
                    <RegionFlag flagCode={r.flagCode} size="xs" shape="circle" />
                    {r.region.name}
                  </span>
                ))}
              </span>
            ) : (
              "—"
            )
          }
        />
        <DetailRow
          icon={<IconHardDriveOutline18 />}
          label="Instances"
          value={running}
          hint="running"
        />
        <DetailRow
          icon={<IconMicrochipOutline18 />}
          label="Resources"
          value={`${deployment.cpuMillicores / 1000} vCPU`}
          hint={`· ${deployment.memoryMib} MiB`}
        />
      </DetailList>
    </CanvasCard>
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
